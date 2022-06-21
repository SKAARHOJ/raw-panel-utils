/*
	Raw Panel ASCII / Binary converter
	Facilitates a connection from a Raw Panel (Server Mode) Panel to a (TCP Client) System.
	Uses protobuf format internally

	NOTICE:
	- Currently doesn't support graphics in multiline incoming ASCII format!

	Distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY;
	without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
	PARTICULAR PURPOSE. MIT License
*/
package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"

	"google.golang.org/protobuf/proto"
)

// Panel centric view:
// Inbound TCP commands - from external system to SKAARHOJ panel
// Outbound TCP commands - from panel to external system
func connectToPanel(panelIPAndPort string, incoming chan []*rwp.InboundMessage, outgoing chan []*rwp.OutboundMessage, binaryPanel bool) {

	for {
		fmt.Println("Trying to connect to panel on " + panelIPAndPort + "...")
		c, err := net.Dial("tcp", panelIPAndPort)

		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second * 3)
		} else {
			fmt.Println("Success - Connected to panel")
			quit := make(chan bool)
			go func() {
				for {
					select {
					case <-quit:
						close(quit)
						return
					case incomingMessages := <-incoming:

						//su.Debug(outboundMessages)
						if binaryPanel {
							for _, msg := range incomingMessages {
								pbdata, err := proto.Marshal(msg)
								log.Should(err)
								header := make([]byte, 4)                                  // Create a 4-bytes header
								binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
								pbdata = append(header, pbdata...)                         // and concatenate it with the binary message
								//log.Println("System -> Panel: ", pbdata)
								_, err = c.Write(pbdata)
								log.Should(err)
							}
						} else {
							lines := helpers.InboundMessagesToRawPanelASCIIstrings(incomingMessages)

							for _, line := range lines {
								publicMessage := string("System -> Panel: " + strings.TrimSpace(string(line)))
								fmt.Println(publicMessage)
								c.Write([]byte(line + "\n"))
							}
						}
					}
				}
			}()

			if binaryPanel {
				for {
					c.SetReadDeadline(time.Time{}) // Reset deadline, waiting for header
					headerArray := make([]byte, 4)
					_, err := io.ReadFull(c, headerArray) // Read 4 header bytes
					if err != nil {
						if err == io.EOF {
							fmt.Println("Panel: " + c.RemoteAddr().String() + " disconnected")
							time.Sleep(time.Second)
						} else {
							fmt.Println("Binary: ", err)
						}
						break
					} else {
						currentPayloadLength := binary.LittleEndian.Uint32(headerArray[0:4])
						if currentPayloadLength < 500000 {
							payload := make([]byte, currentPayloadLength)
							c.SetReadDeadline(time.Now().Add(2 * time.Second)) // Set a deadline that we want all data within at most 2 seconds. This helps a run-away scenario where not all data arrives or we read the wront (and too big) header
							_, err := io.ReadFull(c, payload)
							if err != nil {
								fmt.Println(err)
								break
							} else {
								outcomingMessage := &rwp.OutboundMessage{}
								proto.Unmarshal(payload, outcomingMessage)
								outgoing <- []*rwp.OutboundMessage{outcomingMessage}
							}
						} else {
							fmt.Println("Error: Payload", currentPayloadLength, "exceed limit")
							break
						}
					}
				}
			} else {
				connectionReader := bufio.NewReader(c) // Define OUTSIDE the for loop
				for {
					netData, err := connectionReader.ReadString('\n')
					if err != nil {
						if err == io.EOF {
							fmt.Println("Panel: " + c.RemoteAddr().String() + " disconnected")
							time.Sleep(time.Second)
						} else {
							fmt.Println(err)
						}
						break
					} else {
						outgoing <- helpers.RawPanelASCIIstringsToOutboundMessages([]string{strings.TrimSpace(netData)})
					}
				}
			}

			quit <- true
			c.Close()
		}
	}
}

func connectToSystem(c net.Conn, incoming chan []*rwp.InboundMessage, outgoing chan []*rwp.OutboundMessage, binarySystem bool) {

	fmt.Println("Success - TCP Connection from a system at " + c.RemoteAddr().String() + "...")
	var rwpASCIIreader helpers.ASCIIreader

	quit := make(chan bool)
	go func() {
		for {
			select {
			case <-quit:
				close(quit)
				return
			case outboundMessages := <-outgoing:

				//su.Debug(outboundMessages)
				if binarySystem {
					for _, msg := range outboundMessages {
						pbdata, err := proto.Marshal(msg)
						log.Should(err)
						header := make([]byte, 4)                                  // Create a 4-bytes header
						binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
						pbdata = append(header, pbdata...)                         // and concatenate it with the binary message
						fmt.Println("Panel -> System: ", pbdata)
						_, err = c.Write(pbdata)
						log.Should(err)
					}
				} else {
					lines := helpers.OutboundMessagesToRawPanelASCIIstrings(outboundMessages)

					for _, line := range lines {
						publicMessage := string("Panel -> System: " + strings.TrimSpace(string(line)))
						fmt.Println(publicMessage)
						c.Write([]byte(line + "\n"))
					}
				}
			}
		}
	}()

	if binarySystem {
		for {
			c.SetReadDeadline(time.Time{}) // Reset deadline, waiting for header
			headerArray := make([]byte, 4)
			_, err := io.ReadFull(c, headerArray) // Read 4 header bytes
			if err != nil {
				if err == io.EOF {
					fmt.Println("System: " + c.RemoteAddr().String() + " disconnected")
					time.Sleep(time.Second)
				} else {
					fmt.Println("Binary: ", err)
				}
				break
			} else {
				currentPayloadLength := binary.LittleEndian.Uint32(headerArray[0:4])
				if currentPayloadLength < 500000 {
					payload := make([]byte, currentPayloadLength)
					c.SetReadDeadline(time.Now().Add(2 * time.Second)) // Set a deadline that we want all data within at most 2 seconds. This helps a run-away scenario where not all data arrives or we read the wront (and too big) header
					_, err := io.ReadFull(c, payload)
					if err != nil {
						fmt.Println(err)
						break
					} else {
						incomingMessage := &rwp.InboundMessage{}
						proto.Unmarshal(payload, incomingMessage)
						incoming <- []*rwp.InboundMessage{incomingMessage}
					}
				} else {
					fmt.Println("Error: Payload", currentPayloadLength, "exceed limit")
					break
				}
			}
		}
	} else {
		connectionReader := bufio.NewReader(c) // Define OUTSIDE the for loop
		for {
			netData, err := connectionReader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					fmt.Println("System: " + c.RemoteAddr().String() + " disconnected")
				} else {
					fmt.Println(err)
				}
				break
			} else {
				inputString := strings.TrimSpace(netData)
				//log.Debugln("System -> Panel: ", inputString)

				asciiConvertedMessages := rwpASCIIreader.Parse(inputString)
				if asciiConvertedMessages != nil {
					incoming <- asciiConvertedMessages
				}
			}
		}
	}
	quit <- true
	c.Close()
}

func main() {

	// Setting up and parsing command line parameters
	binPanel := flag.Bool("binPanel", false, "Works with the panel in binary mode")
	binSystem := flag.Bool("binSystem", false, "Works with the system in binary mode")
	flag.Parse()

	arguments := flag.Args()
	if len(arguments) == 0 {
		fmt.Println("usage: ServerPanel2ClientSystem [-binPanel -binSystem] [panelIP:port] [port for system connections]")
		fmt.Println("help:  ServerPanel2ClientSystem -h")
		fmt.Println("")
		return
	}

	portArg, err := strconv.Atoi(arguments[1])
	if err != nil {
		fmt.Println("Port was not an integer")
		fmt.Println("")
		return
	}

	panelIPAndPort := string(arguments[0])

	// Welcome message!
	fmt.Println("Welcome to Raw Panel - Server Panel 2 Client System! Made by Kasper Skaarhoj (c) 2020-2022")
	fmt.Println("Configuration:")
	fmt.Println("  binPanel:  ", *binPanel)
	fmt.Println("  binSystem: ", *binSystem)
	fmt.Println("  system port: ", portArg)
	fmt.Println("Ready to accept TCP connections on port", int(portArg), "and facilitate communication to panel on "+panelIPAndPort+"...\n")

	// Set up server:
	PORT := ":" + arguments[1]

	l, err := net.Listen("tcp", PORT)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	incoming := make(chan []*rwp.InboundMessage, 10)
	outgoing := make(chan []*rwp.OutboundMessage, 10)

	go connectToPanel(panelIPAndPort, incoming, outgoing, *binPanel)

	// Looks for a single incoming connection from the system:
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		connectToSystem(c, incoming, outgoing, *binSystem)
	}

	select {}
}
