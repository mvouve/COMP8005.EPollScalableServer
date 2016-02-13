package main

import (
	"container/list"
	"fmt"
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

func main() {
	if len(os.Args) < 2 { // validate args
		fmt.Println("Missing args:", os.Args[0], " [PORT]")
		return
	}
	// Create structure to handle server info.
	srvInfo := newServerInfo()
	defer srvInfo.Close()
	// create servers
	for i := 0; i < 8; i++ {
		go listen(srvInfo) // spawn listen routines
	}
	// when the server is killed it should print statistics need to catch the signal
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, os.Kill)
	// Setup complete
	manageConnections(srvInfo, osSignals)
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
func manageConnections(srvInfo serverInfo, osSignals chan os.Signal) {
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
	srvInfo := serverInfo{serverConnection: make(chan int, 100), connectInfo: make(chan connectionInfo, 100)}
	fd, err := syscall.Socket(syscall.AF_INET, syscall.O_NONBLOCK|syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Println(err)
	}
	syscall.SetNonblock(fd, true)
	// TODO: make port vairable
	addr := getAddr()
	syscall.Bind(fd, &addr)
	syscall.Listen(fd, 1000)
	srvInfo.listener = fd

	return srvInfo
}

func getAddr() syscall.SockaddrInet4 {
	portStr := strings.SplitAfter(os.Args[1], ":")
	port, err := strconv.Atoi(portStr[0])
	if err != nil {
		log.Fatal("Port must be declaired as :[PORT]")
	}
	addr := syscall.SockaddrInet4{Port: port}
	copy(addr.Addr[:], net.ParseIP("0.0.0.0").To4())

	return addr
}

func (s serverInfo) Close() {
	syscall.Close(s.listener)
}
