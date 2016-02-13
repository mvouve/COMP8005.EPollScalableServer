package main

// #include <sys/select.h>
import "C"

import (
	"container/list"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type connectionInfo struct {
	fileDescriptor     int       // connections file descriptor
	timeStamp          time.Time // the time the connection ended
	hostName           string    // the remote host name
	ammountOfData      int       // the ammount of data transfered to/from the host
	numberOfRequests   int       // the total requests sent to the server from this client
	connectionsAtClose int       // the total number of connections being sustained when the connection was closed.
}

type serverInfo struct {
	serverConnection chan int            // channel used to inform main loop of new connections
	connectInfo      chan connectionInfo // channel to connection info of closing connections to main loop
	listener         int                 // listening socket
}

const newConnectionConst = 1
const finishedConnectionConst = -1
const epollQueueLen = 1000000

/*******************************************************************************
 * Author Marc Vouve
 *
 * Designer Marc Vouve
 *
 * Date: February 6 2016
 *
 * Params: listenFd: The file descriptor of the listening host
 *
 * Return: connectionInfo of the new connection made
 *
 * Notes: This is a helper function for when a new connection is detected by the
 *        observer loop
 *
 ******************************************************************************/
func newConnection(listenFd int) (connectionInfo, error) {
	newFileDescriptor, socketAddr, err := syscall.Accept(int(listenFd))
	if err != nil {
		return connectionInfo{}, err
	}
	syscall.SetNonblock(newFileDescriptor, true)
	var hostname string
	switch socketAddr := socketAddr.(type) {
	default:
		return connectionInfo{}, err
	case *syscall.SockaddrInet4:
		hostname = net.IPv4(socketAddr.Addr[0], socketAddr.Addr[1], socketAddr.Addr[2], socketAddr.Addr[3]).String()
		hostname += ":" + strconv.FormatInt(int64(socketAddr.Port), 10)
	case *syscall.SockaddrInet6:
		hostname = net.IP(socketAddr.Addr[0:16]).String() + ":" + string(socketAddr.Port)
	}

	return connectionInfo{fileDescriptor: newFileDescriptor, hostName: hostname}, nil
}

/* Author: Marc Vouve
 *
 * Designer: Marc Vouve
 *
 * Date: February 6 2016
 *
 * Notes: This function is an "instance" of a server which allows connections in
 *        and echos strings back. After a connection has been closed it will wait
 *        for annother connection
 */
func serverInstance(srvInfo serverInfo) {
	client := make(map[int]connectionInfo)
	epollFd, _ := syscall.EpollCreate1(0)
	events := make([]syscall.EpollEvent, epollQueueLen)
	addConnectionToEPoll(epollFd, srvInfo.listener)
	for {
		_, err := syscall.EpollWait(epollFd, events, -1)
		if err != nil {
			log.Println("err", err)
			return // block shouldn't be hit under normal conditions. If it does something is really wrong.
		}

		for _, ev := range events {
			if ev.Fd == int32(srvInfo.listener) { // new connection
				newClient, err := newConnection(srvInfo.listener)
				if err == nil {
					client[newClient.fileDescriptor] = newClient
					addConnectionToEPoll(epollFd, newClient.fileDescriptor)
				}
			} else { // data to read from connection
				conn := client[int(ev.Fd)]
				_, err := handleData(&conn)
				client[int(ev.Fd)] = conn
				if err != nil {
					endConnection(srvInfo, client[int(ev.Fd)])
				}
			}
		}
	}
}

func addConnectionToEPoll(epFd int, newFd int) {
	event := syscall.EpollEvent{Fd: int32(newFd), Events: syscall.EPOLLIN}
	syscall.EpollCtl(epFd, syscall.EPOLL_CTL_ADD, newFd, &event)
}

func endConnection(srvInfo serverInfo, conn connectionInfo) {
	srvInfo.connectInfo <- conn
	syscall.Close(conn.fileDescriptor)
}

/**/
func handleData(conn *connectionInfo) (int, error) {
	buf := make([]byte, 1024)
	var msg string

	for {
		n, err := syscall.Read(conn.fileDescriptor, buf[:])
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return len(msg), io.EOF
		}

		msg += string(buf[:n])

		if strings.ContainsRune(msg, '\n') {
			break
		}
		conn.ammountOfData += n

	}
	syscall.Write(conn.fileDescriptor, []byte(msg))
	conn.numberOfRequests++

	return len(msg), nil
}

/* Author: Marc Vouve
 *
 * Designer: Marc Vouve
 *
 * Date: February 7 2016
 *
 * Returns: connectionInfo information about the connection once the client has
 *          terminated the client.
 *
 * Notes: This was factored out of the main function.
 */
func observerLoop(srvInfo serverInfo, osSignals chan os.Signal) {
	currentConnections := 0
	connectionsMade := list.New()

	for {
		select {
		case <-srvInfo.serverConnection:
			currentConnections++
		case serverHost := <-srvInfo.connectInfo:
			serverHost.connectionsAtClose = currentConnections
			connectionsMade.PushBack(serverHost)
			currentConnections--
		case <-osSignals:
			generateReport(time.Now().String(), connectionsMade)
			fmt.Println("Total connections made:", connectionsMade.Len())
			os.Exit(1)
		}
	}
}

func newServerInfo() serverInfo {
	srvInfo := serverInfo{
		serverConnection: make(chan int, 10), connectInfo: make(chan connectionInfo)}
	fd, err := syscall.Socket(syscall.AF_INET, syscall.O_NONBLOCK|syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Println(err)
	}
	syscall.SetNonblock(fd, true)
	// TODO: make port vairable
	port, _ := strconv.Atoi(string(os.Args[1]))
	addr := syscall.SockaddrInet4{Port: port}
	copy(addr.Addr[:], net.ParseIP("0.0.0.0").To4())
	syscall.Bind(fd, &addr)
	syscall.Listen(fd, 1000)
	srvInfo.listener = fd

	return srvInfo
}

func (s serverInfo) Close() {
	syscall.Close(s.listener)
}

func main() {
	if len(os.Args) < 2 { // validate args
		fmt.Println("Missing args:", os.Args[0], " [PORT]")

		os.Exit(0)
	}

	srvInfo := newServerInfo()
	defer srvInfo.Close()

	// create servers
	for i := 0; i < 8; i++ {
		go serverInstance(srvInfo)
	}

	// when the server is killed it should print statistics need to catch the signal
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, os.Kill)

	observerLoop(srvInfo, osSignals)
}
