package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"

	ipbase "iptools/ipbase"
)

func main() {

	// Setting up and parsing command line parameters
	hexArgPtr := flag.Bool("hex", false, "Parses input as hex like 'DE AD BE EF' or 'DEADBEEF' (and ignores line ending)")
	noNLArgPtr := flag.Bool("nolf", false, "Strips the newline character from the output")
	crlfArgPtr := flag.Bool("crlf", false, "Uses CR-LF as line ending in output")
	udpArgPtr := flag.Bool("udp", false, "Listens for UDP instead of TCP")
	flag.Parse()

	arguments := flag.Args()
	if len(arguments) == 0 {
		fmt.Println("usage: ipserver [-hex -udp] [port]")
		fmt.Println("help:  ipserver -h")
		fmt.Println("")
		return
	}

	portArg, err := strconv.Atoi(arguments[0])
	if err != nil {
		fmt.Println("Port was not an integer")
		fmt.Println("")
		return
	}

	// Welcome message!
	fmt.Println("Welcome to ipserver! Made by Kasper Skaarhoj 2020")
	fmt.Println("Configuration:")
	fmt.Println("  hex:  ", *hexArgPtr)
	fmt.Println("  nolf: ", *noNLArgPtr)
	fmt.Println("  crlf: ", *crlfArgPtr)
	fmt.Println("  udp:  ", *udpArgPtr)
	fmt.Println("  port: ", portArg)
	fmt.Println("Ready to accept "+ipbase.QStr(*udpArgPtr, "UDP", "TCP")+" connections on port", int(portArg), "and send values back...\n")

	// Set up server:
	PORT := ":" + arguments[0]

	if *udpArgPtr {

		s, err := net.ResolveUDPAddr("udp4", PORT)
		if err != nil {
			fmt.Println(err)
			return
		}

		c, err := net.ListenUDP("udp4", s)
		if err != nil {
			fmt.Println(err)
			return
		}

		defer c.Close()

		returnMessage := make(chan []byte, 2)

		// Looking for input from network:
		go func() {
			byteArray := make([]byte, 2000)
			for {
				byteCount, addr, err := c.ReadFromUDP(byteArray)
				if err != nil {
					fmt.Println(err)
					fmt.Println("")
					return
				}

				ipbase.PrintoutBytes(byteArray, byteCount, 16, "RECV: ")

				select {
				case msg := <-returnMessage:
					ipbase.PrintoutBytes(msg, len(msg), 16, "SENT: ")
					_, err = c.WriteToUDP(msg, addr)
					if err != nil {
						fmt.Println(err)
						return
					}
				default:
				}
			}
		}()

		// Duplicated from ipbase - a shame, but the one in ipbase writes to a Conn object, not a channel, so I'm not sure how to harmonize that.
		rConfig := ipbase.ReaderConfig{Hex: *hexArgPtr, Nolf: *noNLArgPtr, Crlf: *crlfArgPtr}
		go ipbase.LinereaderChannel(returnMessage, rConfig)
	} else {

		connections := ipbase.TCPconnections{}

		// Looking for text input to send:
		rConfig := ipbase.ReaderConfig{Hex: *hexArgPtr, Nolf: *noNLArgPtr, Crlf: *crlfArgPtr}
		go ipbase.LinereaderConnections(&connections, rConfig)

		l, err := net.Listen("tcp", PORT)
		if err != nil {
			fmt.Println(err)
			fmt.Println("")
			return
		}
		defer l.Close()

		for {
			c, err := l.Accept()
			if err != nil {
				fmt.Println(err)
				fmt.Println("")
				return
			}

			// Looking for input from network:
			connections.Add(&c)
			go func() {
				ipbase.Listener(c)
				connections.Remove(&c)
			}()
		}
	}

	// Eternal loop:
	for {
	}
}
