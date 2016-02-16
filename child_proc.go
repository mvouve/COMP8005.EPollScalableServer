/*------------------------------------------------------------------------------
-- DATE:	       February, 2016
--
-- Source File:	 child_proc.go
--
-- REVISIONS: 	(Date and Description)
--
-- DESIGNER:	   Marc Vouve
--
-- PROGRAMMER:	 Marc Vouve
--
--
-- INTERFACE:
--	func newConnection(listenFd int) (connectionInfo, error)
--	func hostString(socketAddr syscall.Sockaddr) string
--  func listen(srvInfo serverInfo)
--  func addConnectionToEPoll(epFd int, newFd int)
--  func endConnection(srvInfo serverInfo, conn connectionInfo)
--  func handleData(conn *connectionInfo) (int, error)
--  func read(fd int) (string, error)
--
--
--
-- NOTES: This file is for functions that are part of child go routines which
--        handle data for the EPoll version of the scalable server.
------------------------------------------------------------------------------*/
package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"syscall"
)

const bufferSize = 1024

/*-----------------------------------------------------------------------------
-- FUNCTION:    newConnection
--
-- DATE:        February 6, 2016
--
-- REVISIONS:	  February 10, 2016 - modified for EPoll
--              February 11, 2016 - fixed blocking issues
--              February 12, 2016 - factored out hostString()
--
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:   func newConnection(listenFd int) (connectionInfo, error)
--  listenFd:   the listener file descriptor with the new connection
--
-- RETURNS:
-- connectionInfo: A structor to store data over the life of the connection
--          error: If an error occurs
--
-- NOTES:			this function sets the port into non blocking mode.
------------------------------------------------------------------------------*/
func newConnection(listenFd int) (connectionInfo, error) {
	newFileDescriptor, socketAddr, err := syscall.Accept(int(listenFd))
	if err != nil {
		return connectionInfo{}, err
	}
	syscall.SetNonblock(newFileDescriptor, true)
	hostName := hostString(socketAddr)

	return connectionInfo{FileDescriptor: newFileDescriptor, HostName: hostName}, nil
}

/*-----------------------------------------------------------------------------
-- FUNCTION:    hostString
--
-- DATE:        February 6, 2016
--
-- REVISIONS:	  February 12, 2016 - Factored out of new connection
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:   func hostString(socketAddr syscall.Sockaddr) string
--  socketAddr: the socket address to extract a string of the address from
--
-- RETURNS:     the ip as a string or "Unknown" if there is an error
--
-- NOTES:			  This function will log if an unknown connection type is found.
------------------------------------------------------------------------------*/
func hostString(socketAddr syscall.Sockaddr) string {
	var hostName string
	switch socketAddr := socketAddr.(type) {
	default:
		log.Printf("%T is not supported, will appear as Uknown in logs", socketAddr)
		return "Unknown"
	case *syscall.SockaddrInet4:
		hostName = net.IPv4(socketAddr.Addr[0], socketAddr.Addr[1], socketAddr.Addr[2], socketAddr.Addr[3]).String()
		hostName += ":" + strconv.FormatInt(int64(socketAddr.Port), 10)
	case *syscall.SockaddrInet6:
		hostName = net.IP(socketAddr.Addr[0:16]).String() + ":" + string(socketAddr.Port)
	}

	return hostName
}

/*-----------------------------------------------------------------------------
-- FUNCTION:    listen
--
-- DATE:        February 6, 2016
--
-- REVISIONS:	  February 10, 2016 - Modified for EPoll
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:   func listen(srvInfo serverInfo)
--   srvInfo:   information about the overall application, IPC and networking
--
-- RETURNS:     void doesn't actually return.
--
-- NOTES:			This function handles Epoll in a foreverloop. This function never
--            exits
------------------------------------------------------------------------------*/
func listen(srvInfo serverInfo) {
	client := make(map[int]connectionInfo)
	epollFd, _ := syscall.EpollCreate1(0)
	events := make([]syscall.EpollEvent, epollQueueLen)
	addConnectionToEPoll(epollFd, srvInfo.listener)

	for {
		n, _ := syscall.EpollWait(epollFd, events[:], -1)
		for i := 0; i < n; i++ {
			if events[i].Events&(syscall.EPOLLHUP|syscall.EPOLLERR) != 0 {
				fmt.Println("Error on epoll")
				endConnection(srvInfo, client[int(events[i].Fd)])
			}
			if events[i].Fd == int32(srvInfo.listener) { // new connection
				newClient, err := newConnection(srvInfo.listener)
				if err == nil {
					srvInfo.serverConnection <- newConnectionConst
					client[newClient.FileDescriptor] = newClient
					addConnectionToEPoll(epollFd, newClient.FileDescriptor)
				}
			} else { // data to read from connection
				//  client[int(ev.Fd)] can not be used as an argument in handleData
				conn := client[int(events[i].Fd)]
				err := handleData(&conn)
				client[int(events[i].Fd)] = conn
				if err != nil {
					endConnection(srvInfo, client[int(events[i].Fd)])
				}
			}
		}
	}
}

/*-----------------------------------------------------------------------------
-- FUNCTION:    addConnectionToEPoll
--
-- DATE:        February 10, 2016
--
-- REVISIONS:	  February 12, 2016 - Factored out of listen
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:   func addConnectionToEPoll(epFd int, newFd int)
--      epFd:   the EPoll File Descriptor to add a newFd to.
--     newFd:   the new file descriptor to add to the epoll file descriptor
--
-- RETURNS:     void
--
-- NOTES:			this function was created to make listen less verbose. Simply adds
              newFd to epollFd waiting on the event EPOLLIN
------------------------------------------------------------------------------*/
func addConnectionToEPoll(epFd int, newFd int) {
	event := syscall.EpollEvent{Fd: int32(newFd), Events: (syscall.EPOLLIN | syscall.EPOLLERR | syscall.EPOLLHUP)}
	syscall.EpollCtl(epFd, syscall.EPOLL_CTL_ADD, newFd, &event)
}

/*-----------------------------------------------------------------------------
-- FUNCTION:    endConnection
--
-- DATE:        February 10, 2016
--
-- REVISIONS:	  February 12, 2016 - Factored out of listen
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:   func addConnectionToEPoll(epFd int, newFd int)
--   srvInfo:   The server info structure assoiated with the running program
--      conn:   The connection info for the current host
--
-- RETURNS:     void
--
-- NOTES:			this function closes the socket and notifies any function listening
--            on the connectInfo channel.
------------------------------------------------------------------------------*/
func endConnection(srvInfo serverInfo, conn connectionInfo) {
	srvInfo.connectInfo <- conn
	syscall.Close(conn.FileDescriptor)
}

/*-----------------------------------------------------------------------------
-- FUNCTION:    handleData
--
-- DATE:        February 7, 2016
--
-- REVISIONS:	  February 12, 2016 - made read function its own function.
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:   func handleData(conn *connectionInfo) (int, error)
--      conn:   structure for the current connection.
--
-- RETURNS:     error: io.EOF if the client closed the connection. or system error.
--              from internal call.
--
-- NOTES:			handles an incoming request from a client.
------------------------------------------------------------------------------*/
func handleData(conn *connectionInfo) error {
	msg, err := read(conn.FileDescriptor)
	if err != nil {
		return err
	}
	syscall.Write(conn.FileDescriptor, []byte(msg))
	conn.NumberOfRequests++
	conn.AmmountOfData += len(msg)

	return nil
}

/*-----------------------------------------------------------------------------
-- FUNCTION:    read
--
-- DATE:        February 7, 2016
--
-- REVISIONS:	  February 12, 2016 - refactored out of handleData function.
--
-- DESIGNER:		Marc Vouve
--
-- PROGRAMMER:	Marc Vouve
--
-- INTERFACE:   func read(fd int) (string, error)
--      fd:     the file descriptor being read from.
--
-- RETURNS:     string: the data read from the client.
--
-- NOTES:			reads data from a file descriptor, returns EOF on remote closing
--            the connection.
------------------------------------------------------------------------------*/
func read(fd int) (string, error) {
	fmt.Println(fd)
	buf := make([]byte, bufferSize)
	var msg string
	for {
		n, err := syscall.Read(fd, buf[:])

		if err != nil {
			return "", err
		}
		if n == 0 {
			return "", io.EOF
		}
		msg += string(buf[:n])
		if err == syscall.EAGAIN {
			fmt.Println(msg)
			return msg, nil
		}
		if strings.ContainsRune(msg, '\n') {
			break
		}
	}

	return msg, nil
}
