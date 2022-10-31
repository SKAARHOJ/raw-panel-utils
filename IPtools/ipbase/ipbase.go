package ipbase

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
)

type ReaderConfig struct {
	Hex  bool
	Nolf bool
	Crlf bool
	Cr   bool
}
type TCPconnections struct {
	sync.RWMutex

	connections []*net.Conn
}

func (tcpconnections *TCPconnections) Add(c *net.Conn) {
	tcpconnections.Lock()
	defer tcpconnections.Unlock()

	tcpconnections.connections = append(tcpconnections.connections, c)
}

func (tcpconnections *TCPconnections) Remove(c *net.Conn) {
	tcpconnections.Lock()
	defer tcpconnections.Unlock()

	for s, cf := range tcpconnections.connections {
		if cf == c {
			fmt.Printf("Remove entry %d\n", s)
			tcpconnections.connections = append(tcpconnections.connections[:s], tcpconnections.connections[s+1:]...)
			break
		}
	}
}

func Listener(connection net.Conn) {
	byteArray := make([]byte, 2000)
	for {
		byteCount, err := connection.Read(byteArray)
		if err != nil {
			fmt.Println("err != nil")
			fmt.Println(err)
			fmt.Println("")
			return
		}

		PrintoutBytes(byteArray, byteCount, 16, "RECV: ")
	}
}

func ListenerUDP(connection *net.UDPConn) {
	byteArray := make([]byte, 2000)
	for {
		byteCount, _, err := connection.ReadFromUDP(byteArray)
		if err != nil {
			fmt.Println(err)
			fmt.Println("")
			return
		}

		PrintoutBytes(byteArray, byteCount, 16, "RECV: ")
	}
}

// PrintoutBytes Public... upper case!
func PrintoutBytes(byteArray []byte, byteCount int, setSize int, prefix string) {

	for ptr := 0; ptr < byteCount; ptr += setSize {

		fmt.Print(prefix)

		for j := 0; j < setSize; j++ {
			if j+ptr < byteCount {
				fmt.Printf("%02X ", byteArray[j+ptr])
			} else {
				fmt.Printf("   ")
			}
		}

		substr := string(byteArray)[ptr:QInt(ptr+setSize < byteCount, ptr+setSize, byteCount)]

		fmt.Println(" " + strings.ReplaceAll(substr, "\n", " "))
	}
	fmt.Println()
}

func Linereader(connection net.Conn, rConfig ReaderConfig) {
	for {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')

		bytes := parseInput(text, rConfig)

		PrintoutBytes(bytes, len(bytes), 16, "SENT: ")
		connection.Write(bytes)
	}
}

func LinereaderConnections(connections *TCPconnections, rConfig ReaderConfig) {
	for {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')

		bytes := parseInput(text, rConfig)

		PrintoutBytes(bytes, len(bytes), 16, "SENT: ")

		for s, c := range connections.connections {
			fmt.Printf("Print to connection %d\n", s)
			(*c).Write(bytes)
		}
	}
}

func LinereaderChannel(returnMessage chan []byte, rConfig ReaderConfig) {
	for {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')

		bytes := parseInput(text, rConfig)

		PrintoutBytes(bytes, len(bytes), 16, "KEYB: ")
		returnMessage <- bytes
	}
}

func parseInput(text string, rConfig ReaderConfig) []byte {
	var bytes []byte

	if rConfig.Hex {
		_bytes, err := hex.DecodeString(strings.ReplaceAll(text[0:len(text)-1], " ", ""))
		if err != nil {
			fmt.Println(err)
		}
		bytes = _bytes
	} else if rConfig.Nolf {
		bytes = []byte(text[0 : len(text)-1])
	} else if rConfig.Crlf {
		bytes = []byte(text[0:len(text)-1] + "\r\n")
	} else if rConfig.Cr {
		bytes = []byte(text[0:len(text)-1] + "\r")
	} else {
		bytes = []byte(text)
	}

	return bytes
}

// internal...
func QInt(condition bool, ifTrue int, ifFalse int) int {
	if condition {
		return ifTrue
	} else {
		return ifFalse
	}
}

// internal...
func QStr(condition bool, ifTrue string, ifFalse string) string {
	if condition {
		return ifTrue
	} else {
		return ifFalse
	}
}
