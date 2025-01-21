/*
This is a Raw Panel Connector application

It connects to a Panel in Raw Panel Server mode (eg. on 192.168.10.99:9923)
It also connects to a System (as a Raw Panel TCP server (eg. on 192.168.10.250:9923)
It takes care of ping handshakes
It facilitates that the panel sends "list" to the system server (something a UniSketch panel currently doesn't)
It also sends "map" over to the panel from the server since this is usually done by the panels initiative in client mode.

Other than that it just forwards messages between the panel and server, but translates into the intermediate Raw Panel protobuf format forth and back
The consequence of this translation is that any graphics over multiple lines from the server side will NOT survive the translation as this gets parsed as individual lines: Graphics won't come through!

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
	"strings"
	"time"

	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"

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
								pbdata, _ := proto.Marshal(msg)
								fmt.Println("System -> Panel: ", pbdata)
								header := make([]byte, 4)                                  // Create a 4-bytes header
								binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
								pbdata = append(header, pbdata...)                         // and concatenate it with the binary message
								c.Write(pbdata)
							}
						} else {
							lines := helpers.InboundMessagesToRawPanelASCIIstrings(incomingMessages)

							for _, line := range lines {
								fmt.Println(string("System -> Panel: " + strings.TrimSpace(string(line))))
								c.Write([]byte(line + "\n"))
							}
						}
					}
				}
			}()

			ticker := time.NewTicker(500 * time.Millisecond)
			done := make(chan bool)
			go func() {
				for {
					select {
					case <-done:
						return
					case t := <-ticker.C:
						if binaryPanel {
							msg := &rwp.InboundMessage{
								FlowMessage: 1,
							}
							pbdata, _ := proto.Marshal(msg)
							header := make([]byte, 4)                                  // Create a 4-bytes header
							binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
							pbdata = append(header, pbdata...)                         // and concatenate it with the binary message
							c.Write(pbdata)
						} else {
							c.Write([]byte("ping\n"))
						}

						_ = t
						//fmt.Println("Tick at", t)
						fmt.Println(".")
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
								outgoingMessage := &rwp.OutboundMessage{}
								proto.Unmarshal(payload, outgoingMessage)
								if outgoingMessage.FlowMessage != 2 { // ack
									outgoing <- []*rwp.OutboundMessage{outgoingMessage}
								}
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
					} else {
						netDataStr := strings.TrimSpace(netData)
						switch netDataStr {
						case "ack":
						default:
							outgoing <- helpers.RawPanelASCIIstringsToOutboundMessages([]string{netDataStr})
						}
					}
				}
			}

			done <- true
			quit <- true
			c.Close()
		}
	}
}

func connectToSystem(systemIPAndPort string, incoming chan []*rwp.InboundMessage, outgoing chan []*rwp.OutboundMessage, binarySystem bool) {
	var rwpASCIIreader helpers.ASCIIreader

	for {
		fmt.Println("Trying to connect to system on " + systemIPAndPort + "...")
		c, err := net.Dial("tcp", systemIPAndPort)

		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second * 3)
		} else {
			fmt.Println("Success - Connected to system")

			if binarySystem {
				msg := &rwp.OutboundMessage{
					FlowMessage: 100, // Hello
				}
				pbdata, _ := proto.Marshal(msg)
				header := make([]byte, 4)                                  // Create a 4-bytes header
				binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
				pbdata = append(header, pbdata...)                         // and concatenate it with the binary message
				c.Write(pbdata)
			} else {
				c.Write([]byte("list\n")) // Initialize with system
			}

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
								pbdata, _ := proto.Marshal(msg)
								fmt.Println("Panel -> System: ", pbdata)
								header := make([]byte, 4)                                  // Create a 4-bytes header
								binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
								pbdata = append(header, pbdata...)                         // and concatenate it with the binary message
								c.Write(pbdata)
							}
						} else {
							lines := helpers.OutboundMessagesToRawPanelASCIIstrings(outboundMessages)

							for _, line := range lines {
								fmt.Println(string("Panel -> System: " + strings.TrimSpace(string(line))))
								c.Write([]byte(line + "\n"))
							}
						}
					}
				}
			}()

			ticker := time.NewTicker(1000 * time.Millisecond)
			done := make(chan bool)
			go func() {
				for {
					select {
					case <-done:
						return
					case t := <-ticker.C:
						if binarySystem {
							msg := &rwp.OutboundMessage{
								FlowMessage: 1,
							}
							pbdata, _ := proto.Marshal(msg)
							header := make([]byte, 4)                                  // Create a 4-bytes header
							binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
							pbdata = append(header, pbdata...)                         // and concatenate it with the binary message
							c.Write(pbdata)
						} else {
							c.Write([]byte("ping\n"))
						}
						_ = t
						//fmt.Println("Tick at", t)
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
								inboundMessages := []*rwp.InboundMessage{incomingMessage}
								if incomingMessage.FlowMessage != 2 { // ack
									if incomingMessage.Command != nil && incomingMessage.Command.ActivatePanel {
										inboundMessages = append(inboundMessages, &rwp.InboundMessage{
											Command: &rwp.Command{
												ReportHWCavailability: true,
											},
										})
									}

									incoming <- inboundMessages
								}
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
						netDataStr := strings.TrimSpace(netData)

						inboundMessages := helpers.RawPanelASCIIstringsToInboundMessages([]string{netDataStr})

						switch netDataStr {
						case "ack":
						default:
							if netDataStr == "ActivePanel=1" {
								inboundMessages = append(inboundMessages, &rwp.InboundMessage{
									Command: &rwp.Command{
										ReportHWCavailability: true,
									},
								})
							}
							incoming <- inboundMessages

							asciiConvertedMessages := rwpASCIIreader.Parse(netDataStr)
							if asciiConvertedMessages != nil {
								incoming <- asciiConvertedMessages
							}
						}
					}
				}
			}

			done <- true
			quit <- true
			c.Close()
		}
	}
}

func main() {

	// Setting up and parsing command line parameters
	binPanel := flag.Bool("binPanel", false, "Works with the panel in binary mode")
	binSystem := flag.Bool("binSystem", false, "Works with the system in binary mode")
	flag.Parse()

	arguments := flag.Args()
	if len(arguments) == 0 {
		fmt.Println("usage: ServerPanel2ServerSystem [-binPanel -binSystem] [panelIP:port] [systemIP:port]")
		fmt.Println("help:  ServerPanel2ServerSystem -h")
		fmt.Println("")
		return
	}

	panelIPAndPort := string(arguments[0])
	systemIPAndPort := string(arguments[1])

	// Welcome message!
	fmt.Println("Welcome to Raw Panel - Server Panel to Server System! Made by Kasper Skaarhoj (c) 2020-2022")
	fmt.Println("Configuration:")
	fmt.Println("  binPanel:  ", *binPanel)
	fmt.Println("  binSystem: ", *binSystem)
	fmt.Println("Ready to facilitate communication between a panel and system, both in server mode. Starting to connect...\n")

	// Set up server:
	incoming := make(chan []*rwp.InboundMessage, 10)
	outgoing := make(chan []*rwp.OutboundMessage, 10)

	go connectToPanel(panelIPAndPort, incoming, outgoing, *binPanel)
	go connectToSystem(systemIPAndPort, incoming, outgoing, *binSystem)

	select {}
}
