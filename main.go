package main

import (
	"github.com/pkg/term"
	"os"
	"strconv"
	"net"
	"io"
	"sync"
	"log"
	"fmt"
	"encoding/hex"
)

var clients map[net.Conn]bool
var writeLock *sync.Mutex

var DEBUG bool = false

func main() {
	log.SetOutput(os.Stdout)

	clients = make(map[net.Conn]bool)
	writeLock = new(sync.Mutex)

	if len(os.Args) < 4 {
		log.Printf("usage: %s <serial port> <baud> <tcp port> [debug]\n", os.Args[0])
		os.Exit(1)
	}
	if len(os.Args) == 5 && os.Args[4] == "debug" {
		DEBUG = true
	}

	serialPortName := os.Args[1]
	baud, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Println("Error parsing baud")
		os.Exit(1)
	}
	tcpPort, err := strconv.Atoi(os.Args[3])
	if err != nil {
		log.Println("Error parsing tcp port")
		os.Exit(1)
	}

	serialPort, err := term.Open(serialPortName)
	if err != nil {
		log.Printf("Unable to open Serial Port %+v\n", err)
		os.Exit(1)
	}
	serialPort.SetSpeed(baud)

	l, err := net.Listen("tcp", ":" + strconv.Itoa(tcpPort))
	if err != nil {
		log.Printf("Unable to open Listen on Port %d %+v\n", tcpPort, err)
		os.Exit(1)
	}
	defer l.Close()

	go serialPortReader(serialPort)

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Error accepting: ", err.Error())
			continue
		}
		go handleRequest(conn, serialPort)
	}
}

func serialPortReader(serialPort io.Reader) {
	buf := make([]byte, 1024)
	for {
		n, err := serialPort.Read(buf)
		if err != nil {
			log.Printf("Unable to read from serialport %+v\n", err)
			os.Exit(1)
		}
		log.Printf("Read %d bytes from serialport\n", n)
		if DEBUG {
			fmt.Println(hex.Dump(buf[:n]))
		}
		for k := range clients {
			_, err = k.Write(buf[:n])
			if err != nil {
				log.Printf("Unable to write to conn %s %+v\n", k.RemoteAddr(), err)
				continue
			}
		}
	}
}

func handleRequest(conn net.Conn, serialPort io.Writer) {
	defer conn.Close()
	defer delete(clients, conn)

	buf := make([]byte, 1024)
	clients[conn] = true

	log.Printf("%d Connect %s\n", len(clients), conn.RemoteAddr())
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("Unable to read from conn %s %+v\n", conn.RemoteAddr(), err)
			return
		}

		writeLock.Lock()
		n, err = serialPort.Write(buf[:n])
		if err != nil {
			log.Printf("Unable to read from conn %s %+v\n", conn.RemoteAddr(), err)
			writeLock.Unlock()
			return
		}
		writeLock.Unlock()
		log.Printf("Wrote %d bytes to serialport\n", n)
		if DEBUG {
			fmt.Println(hex.Dump(buf[:n]))
		}
	}
}