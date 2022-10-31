package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"

	ipbase "iptools/ipbase"
)

func main() {

	// Setting up and parsing command line parameters
	hexArgPtr := flag.Bool("hex", false, "Parses input as hex like 'DE AD BE EF' or 'DEADBEEF' (and ignores line ending)")
	noNLArgPtr := flag.Bool("nolf", false, "Strips the newline character from the output")
	crlfArgPtr := flag.Bool("crlf", false, "Uses CR-LF as line ending in output")
	crArgPtr := flag.Bool("cr", false, "Uses CR as line ending in output")
	udpArgPtr := flag.Bool("udp", false, "Sends UDP instead of TCP")
	flag.Parse()

	arguments := flag.Args()
	if len(arguments) == 0 {
		fmt.Println("usage: ipclient [-hex -udp] [host] [port]")
		fmt.Println("help:  ipclient -h")
		fmt.Println("")
		return
	}

	host := string(arguments[0])

	if len(arguments) < 2 {
		fmt.Println("No port was given")
		fmt.Println("")
		return
	}

	portArg, err := strconv.Atoi(arguments[1])
	if err != nil {
		fmt.Println("Port was not an integer")
		fmt.Println("")
		return
	}

	CONNECT := host + ":" + strconv.Itoa(portArg)

	// Welcome message!
	fmt.Println("Welcome to ipclient! Made by Kasper Skaarhoj 2020")
	fmt.Println("Configuration:")
	fmt.Println("  hex:  ", *hexArgPtr)
	fmt.Println("  nolf: ", *noNLArgPtr)
	fmt.Println("  crlf: ", *crlfArgPtr)
	fmt.Println("  cr: ", *crArgPtr)
	fmt.Println("  udp:  ", *udpArgPtr)
	fmt.Println("  host: ", host)
	fmt.Println("  port: ", portArg)
	fmt.Println("Ready to send " + ipbase.QStr(*udpArgPtr, "UDP", "TCP") + " messages to " + CONNECT + " and receive values back...\n")

	var c net.Conn
	if *udpArgPtr {
		s, err := net.ResolveUDPAddr("udp4", CONNECT)
		c, err = net.DialUDP("udp4", nil, s)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer c.Close()

		// Looking for input from network:
		go ipbase.ListenerUDP(c.(*net.UDPConn))
	} else {
		c, err = net.Dial("tcp", CONNECT)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Looking for input from network:
		go ipbase.Listener(c)
	}

	// Looking for text input to send:
	rConfig := ipbase.ReaderConfig{Hex: *hexArgPtr, Nolf: *noNLArgPtr, Crlf: *crlfArgPtr, Cr: *crArgPtr}
	go ipbase.Linereader(c, rConfig)

	// Eternal loop:
	for {
		time.Sleep(time.Millisecond * 20)

		//packet := []byte{0xF0, 0, 0}
		//packet := []byte{0xB0, 0, 0, 0, 0, 4}
		packet := []byte{0x30, 0, 0, 0, 0, 0, 0xFF}
		//packet := []byte{0x20, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00}

		checksum := byte(0)
		for _, v := range packet {
			checksum += v
		}
		checksum += byte(len(packet)) + 1

		packet = append(packet, checksum)
		packet = append([]byte{byte(len(packet))}, packet...)
		//ipbase.PrintoutBytes(packet, len(packet), 16, "packet: ")
		c.Write(packet)
	}
}
