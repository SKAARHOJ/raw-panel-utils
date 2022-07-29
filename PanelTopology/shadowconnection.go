package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	topology "github.com/SKAARHOJ/rawpanel-lib/topology"
	"go.uber.org/atomic"
	"google.golang.org/protobuf/proto"

	log "github.com/s00500/env_logger"
)

var shadowPanelIncoming chan []*rwp.InboundMessage
var shadowPanelListening atomic.Bool
var shadowPanelTopologyData *topology.Topology

// Panel centric view:
// Inbound TCP commands - from external system to SKAARHOJ panel
// Outbound TCP commands - from panel to external system
func connectToShadowPanel(panelIPAndPort string, shadowIncoming chan []*rwp.InboundMessage) {

	binaryPanel := true

	for {
		log.Infoln("Trying to connect to shadow panel on " + panelIPAndPort)
		c, err := net.Dial("tcp", panelIPAndPort)

		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second * 3)
		} else {
			log.Infoln("(SHADOW PANEL) TCP Connection established...")

			// Is panel ASCII or Binary? Try by sending a binary ping to the panel.
			// Background: Since it's possible that a panel auto detects binary or ascii protocol mode itself, it's better to probe with a Binary package since otherwise a binary capable panel/system pair in auto mode would negotiate to use ASCII which is not efficient.
			pingMessage := &rwp.InboundMessage{
				FlowMessage: rwp.InboundMessage_PING,
			}
			pbdata, err := proto.Marshal(pingMessage)
			log.Should(err)
			header := make([]byte, 4)                                  // Create a 4-bytes header
			binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
			pbdata = append(header, pbdata...)                         // and concatenate it with the binary message
			log.Infoln("(SHADOW PANEL) Autodetecting binary / ascii mode of panel", panelIPAndPort, "by sending binary ping:", pbdata)

			_, err = c.Write(pbdata) // Send "ping" and wait one second for a reply:
			log.Should(err)
			byteArray := make([]byte, 1000)
			err = c.SetReadDeadline(time.Now().Add(2000 * time.Millisecond))
			log.Should(err)

			byteCount, err := c.Read(byteArray) // Should timeout after 2 seconds if ascii panel, otherwise respond promptly with an ACK message
			assumeASCII := false
			if err == nil {
				if byteCount > 4 {
					responsePayloadLength := binary.LittleEndian.Uint32(byteArray[0:4])
					if responsePayloadLength+4 == uint32(byteCount) {
						reply := &rwp.OutboundMessage{}
						proto.Unmarshal(byteArray[4:byteCount], reply)
						if reply.FlowMessage == rwp.OutboundMessage_ACK {
							log.Println("(SHADOW PANEL) Received ACK successfully: ", byteArray[0:byteCount])
							log.Infoln("(SHADOW PANEL) Using Binary Protocol Mode for panel ", panelIPAndPort)
						} else {
							log.Infoln("(SHADOW PANEL) Received something else than an ack response, staying with Binary Protocol Mode for panel ", panelIPAndPort)
						}
					} else {
						log.Infoln("(SHADOW PANEL) Bytecount didn't match header")
						assumeASCII = true
					}
				} else {
					log.Infoln("(SHADOW PANEL) Unexpected reply length")
					assumeASCII = true
				}
			} else {
				log.WithError(err).Debug("(SHADOW PANEL) Tried to connected in binary mode failed.")
				assumeASCII = true
			}
			err = c.SetReadDeadline(time.Time{}) // Reset - necessary for ASCII line reading.

			if assumeASCII {
				log.Printf("(SHADOW PANEL) Reply from panel was: %s\n", strings.ReplaceAll(string(byteArray[:byteCount]), "\n", "\\n"))
				log.Infoln("(SHADOW PANEL) Using ASCII Protocol Mode for panel", panelIPAndPort)
				_, err = c.Write([]byte("\n")) // Clearing an ASCII panels buffer with a newline since we sent it binary stuff
				binaryPanel = false
			}

			// Send query for a lot of stuff we want to know...:
			shadowIncoming <- []*rwp.InboundMessage{
				&rwp.InboundMessage{
					Command: &rwp.Command{
						ActivatePanel:     true,
						SendPanelInfo:     true,
						SendPanelTopology: true,
					},
				},
			}

			outgoing := make(chan []*rwp.OutboundMessage, 10)

			quit := make(chan bool)
			go func() {
				//a := 0
				shadowPanelListening.Store(true)
				for {
					select {
					case <-quit:
						shadowPanelListening.Store(false)
						return
					case incomingMessages := <-shadowIncoming:
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
								//fmt.Println(string("System -> Panel: " + strings.TrimSpace(string(line))))
								c.Write([]byte(line + "\n"))
							}
						}

					case outboundMessages := <-outgoing:
						for _, msg := range outboundMessages {
							if msg.PanelTopology != nil {
								if msg.PanelTopology.Json != "" {
									shadowPanelTopologyData := &topology.Topology{}
									err := json.Unmarshal([]byte(msg.PanelTopology.Json), shadowPanelTopologyData)
									if err != nil {
										log.Println("(SHADOW PANEL) Topology JSON parsing Error: ", err)
									}

									s, _ := json.MarshalIndent(shadowPanelTopologyData, "", "  ")

									if *writeTopologiesToFiles {
										f, _ := os.Create("_topology(Shadow).json")
										defer f.Close()
										f.WriteString(string(s))
										f.Sync()
									}
								}
							}
							if msg.FlowMessage == 1 { // Ping:
								shadowIncoming <- []*rwp.InboundMessage{
									&rwp.InboundMessage{
										FlowMessage: 2,
									},
								}
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
						log.Println("(SHADOW PANEL) Binary: ", err)
						break
					} else {
						currentPayloadLength := binary.LittleEndian.Uint32(headerArray[0:4])
						if currentPayloadLength < 500000 {
							payload := make([]byte, currentPayloadLength)
							c.SetReadDeadline(time.Now().Add(2 * time.Second)) // Set a deadline that we want all data within at most 2 seconds. This helps a run-away scenario where not all data arrives or we read the wront (and too big) header
							_, err := io.ReadFull(c, payload)
							if err != nil {
								log.Println(err)
								break
							} else {
								outcomingMessage := &rwp.OutboundMessage{}
								proto.Unmarshal(payload, outcomingMessage)
								outgoing <- []*rwp.OutboundMessage{outcomingMessage}
							}
						} else {
							log.Println("(SHADOW PANEL) Error: Payload", currentPayloadLength, "exceed limit")
							break
						}
					}
				}
			} else {
				//log.Println("Reading ASCII lines...")
				connectionReader := bufio.NewReader(c) // Define OUTSIDE the for loop
				for {
					netData, err := connectionReader.ReadString('\n')
					if err != nil {
						if err == io.EOF {
							log.Println("(SHADOW PANEL) Panel: " + c.RemoteAddr().String() + " disconnected")
							time.Sleep(time.Second)
						} else {
							log.Println(err)
						}
						break
					} else {
						outgoing <- helpers.RawPanelASCIIstringsToOutboundMessages([]string{strings.TrimSpace(netData)})
					}
				}
			}

			log.Println("(SHADOW PANEL) Network connection closed or failed")
			close(quit)
			c.Close()
			return
		}
	}
}
