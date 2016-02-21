# COMP8005.EPollScaleableServer
This project was written in Go and is intended to aid in comparing the preformance of epoll, select and a traditional multi-threaded network archetecture using go routines. This repository contains the default epoll scalable server. The other two repositories in this collection can be found here:
* [multithreaded](https://github.com/mvouve/COMP8005.ScalableServer)
* [select](https://github.com/mvouve/COMP8005.SelectScalableServer)

I have also written a [client](https://github.com/mvouve/COMP8005.ScalableServerClient) that works with all three servers. Full design documents, and a report based off findings from this experiment can be found there as well.

##Usage
This server can be envoked using the syntax of:
```bash
./COMP8005.EPollScaleableServer [Port]
```

When terminated, the process will exit and generate an XLSX report listing clients that had connected, the ammount of data that they transfered and the number of times they transfered data to the server as well as other useful information about the connections.

##Testing
This program has been tested to work on Fedora 22 and Manjaro 15 using a standered Go 1.5 compiler. It has been able to sustain over 40k concurrent connections.
