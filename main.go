/*------------------------------------------------------------------------------
-- DATE:	       February, 2016
--
-- Source File:	 main.go
--
-- REVISIONS: 	(Date and Description)
--
-- DESIGNER:	   Marc Vouve
--
-- PROGRAMMER:	 Marc Vouve
--
--
-- INTERFACE:
--  func main()
--	func manageConnections(srvInfo serverInfo, osSignals chan os.Signal)
--	func newServerInfo() serverInfo
--  func getAddr() syscall.SockaddrInet4
--  func (s serverInfo) Close()
--
-- NOTES: This is the main file for the EPoll Scalable server
------------------------------------------------------------------------------*/

package main

import (
	"container/list"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

type connectionInfo struct {
	FileDescriptor     int    // connections file descriptor
	HostName           string // the remote host name
	AmmountOfData      int    // the ammount of data transfered to/from the host
	NumberOfRequests   int    // the total requests sent to the server from this client
	ConnectionsAtClose int    // the total number of connections being sustained when the connection was closed.
}

type serverInfo struct {
	serverConnection chan int            // channel used to inform main loop of new connections
	connectInfo      chan connectionInfo // channel to connection info of closing connections to main loop
	listener         int                 // listening socket
}

const newConnectionConst = 1
const finishedConnectionConst = -1
const epollQueueLen = 1000000
const backlog = 10000
const numThreads = 30

/*-----------------------------------------------------------------------------
-- FUNCTION:    main
--
-- DATE:        February 6, 2016
--
-- REVISIONS:	  February 11, 2016 Modified for EPoll
--
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:   func main()
--
-- RETURNS:     void
--
-- NOTES:			The main entry point for the scalable server.
------------------------------------------------------------------------------*/
func main() {
	if len(os.Args) < 2 { // validate args
		fmt.Println("Missing args:", os.Args[0], " [PORT]")
		return
	}
	srvInfo := newServerInfo()
	defer srvInfo.Close()
	for i := 0; i < numThreads; i++ {
		go listen(srvInfo) // spawn listen routines
	}
	osSignals := make(chan os.Signal)
	signal.Notify(osSignals, os.Interrupt, os.Kill)
	manageConnections(srvInfo, osSignals)
}

/*-----------------------------------------------------------------------------
-- FUNCTION:    manageConnections
--
-- DATE:        February 7, 2016
--
-- REVISIONS:	  February 11, 2016 Modified for EPoll
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:   func manageConnections(srvInfo serverInfo, osSignals chan os.Signal)
--	 srvInfo:   information about the server (IPC and listening port)
-- osSignals:	  listens for signals from the OS that should stop the server from running
--
-- RETURNS:     void
--
-- NOTES:			This server will loop until inturupted by an OS Signal on soSignals.
------------------------------------------------------------------------------*/
func manageConnections(srvInfo serverInfo, osSignals chan os.Signal) {
	currentConnections := 0
	connectionsMade := list.New()

	for {
		select {
		case <-srvInfo.serverConnection:
			currentConnections++
		case serverHost := <-srvInfo.connectInfo:
			serverHost.ConnectionsAtClose = currentConnections
			connectionsMade.PushBack(serverHost)
			currentConnections--
		case <-osSignals:
			generateReport(time.Now().String(), connectionsMade)
			fmt.Println("Total connections made:", connectionsMade.Len())
			os.Exit(1)
		}
	}
}

/*-----------------------------------------------------------------------------
-- FUNCTION:    manageConnections
--
-- DATE:        February 7, 2016
--
-- REVISIONS:	  February 11, 2016 Modified for EPoll
--
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:   func manageConnections(srvInfo serverInfo, osSignals chan os.Signal)
--	 srvInfo:   information about the server (IPC and listening port)
-- osSignals:	  listens for signals from the OS that should stop the server from running
--
-- RETURNS:     void
--
-- NOTES:			This server will loop until inturupted by an OS Signal on soSignals.
------------------------------------------------------------------------------*/
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
	syscall.Listen(fd, backlog)
	srvInfo.listener = fd

	return srvInfo
}

/*-----------------------------------------------------------------------------
-- FUNCTION:    getAddr
--
-- DATE:        February 11, 2016
--
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:   getAddr() syscall.SockaddrInet4
--
-- RETURNS:     syscall.SockaddrInet4 a sockaddr of the port specified by the user.
--
-- NOTES:			  this funciton parses a port out of user input
------------------------------------------------------------------------------*/
func getAddr() syscall.SockaddrInet4 {
	port, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal("Port must be declaired as [PORT]")
	}
	addr := syscall.SockaddrInet4{Port: port}
	copy(addr.Addr[:], net.ParseIP("0.0.0.0").To4())

	return addr
}

/*-----------------------------------------------------------------------------
-- FUNCTION:    close
--
-- DATE:        February 7, 2016
--
-- REVISIONS:	  February 11, 2016 Modified for EPoll
--
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:   func (s serverInfo) Close()
--
-- RETURNS:     void
--
-- NOTES:			  This function closes everything that must be closed in serverInfo
--							upon program termination. currently just the listener
------------------------------------------------------------------------------*/
func (s serverInfo) Close() {
	syscall.Close(s.listener)
}
