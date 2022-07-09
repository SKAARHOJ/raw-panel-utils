package main

import (
	"embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"github.com/gorilla/websocket"
	log "github.com/s00500/env_logger"
	"google.golang.org/protobuf/proto"
)

type wsToClient struct {
	Title         string `json:",omitempty"`
	Model         string `json:",omitempty"`
	Serial        string `json:",omitempty"`
	SvgIcon       string `json:",omitempty"`
	TopologyTable string `json:",omitempty"`
	TopologyJSON  string `json:",omitempty"`
	Time          string `json:",omitempty"`
	ControlBlock  string `json:",omitempty"`

	PanelEvent *rwp.HWCEvent `json:",omitempty"`
	RWPState   *rwp.HWCState `json:",omitempty"`

	RWPASCIIToPanel    string `json:",omitempty"`
	RWPProtobufToPanel string `json:",omitempty"`
	RWPJSONToPanel     string `json:",omitempty"`
}

type wsFromClient struct {
	RWPState             *rwp.HWCState `json:",omitempty"`
	RWPStateAscii        string        `json:",omitempty"`
	RequestControlForHWC int           `json:",omitempty"`
}

type wsclient struct {
	msgToClient chan *wsToClient
	quit        chan bool
}

type threadSafeSlice struct {
	sync.Mutex
	wsclients []*wsclient
}

func (slice *threadSafeSlice) Push(w *wsclient) {
	wsslice.Lock()
	defer wsslice.Unlock()
	wsslice.wsclients = append(wsslice.wsclients, w)
}

func (slice *threadSafeSlice) Iter(routine func(*wsclient)) {
	wsslice.Lock()
	defer wsslice.Unlock()
	for _, wsclient := range wsslice.wsclients {
		routine(wsclient)
	}
}

var wsslice threadSafeSlice

func (w *wsclient) Start(ws *websocket.Conn) {
	w.msgToClient = make(chan *wsToClient, 10) // some buffer size to avoid blocking
	go func() {
		for {
			select {
			case msg := <-w.msgToClient:
				msgAsString, _ := json.Marshal(msg)
				//fmt.Println("msg to websocket: " + string(msgAsString))
				ws.WriteMessage(1, msgAsString)
			case <-w.quit:
				return
			}
		}
	}()
}

// We'll need to define an Upgrader
// this will require a Read and Write buffer size
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, string(ReadResourceFile("resources/index.html")))
}
func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// upgrade this connection to a WebSocket
	// connection
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	ww := &wsclient{}
	ww.Start(ws)
	wsslice.Push(ww)

	// listen indefinitely for new messages coming
	// through on our WebSocket connection
	reader(ws)
	ww.quit <- true
	fmt.Println("Exit")
}

// define a reader which will listen for
// new messages being sent to our WebSocket
// endpoint
func reader(conn *websocket.Conn) {
	for {
		// read in a message
		_, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}
		// print out that message for clarity
		switch string(p) {
		case "SendAll":
			lastStateMu.Lock()
			wsslice.Iter(func(w *wsclient) { w.msgToClient <- lastState })
			lastStateMu.Unlock()
		default:
			wsFromClient := &wsFromClient{}
			err := json.Unmarshal(p, wsFromClient)
			log.Should(err)
			//log.Println("Received from websocket: ", log.Indent(wsFromClient))

			if wsFromClient.RequestControlForHWC > 0 {
				wsToClient := &wsToClient{
					RWPState: &rwp.HWCState{},
				}
				wsslice.Iter(func(w *wsclient) { w.msgToClient <- wsToClient })
			}

			if wsFromClient.RWPState != nil {
				log.Println("Received State Change from Client: ", log.Indent(wsFromClient.RWPState))

				/*
					// If empty HWCMode structs are removed, we won't see triggers like "Off".
					if proto.Equal(wsFromClient.RWPState.HWCMode, &rwp.HWCMode{}) {
						wsFromClient.RWPState.HWCMode = nil
					} */
				if proto.Equal(wsFromClient.RWPState.HWCColor, &rwp.HWCColor{}) {
					wsFromClient.RWPState.HWCColor = nil
				}
				if proto.Equal(wsFromClient.RWPState.HWCExtended, &rwp.HWCExtended{}) {
					wsFromClient.RWPState.HWCExtended = nil
				}
				if proto.Equal(wsFromClient.RWPState.HWCText, &rwp.HWCText{}) {
					wsFromClient.RWPState.HWCText = nil
				}

				incomingMessages := []*rwp.InboundMessage{
					&rwp.InboundMessage{
						States: []*rwp.HWCState{
							wsFromClient.RWPState,
						},
					},
				}

				stateAsJsonString, _ := json.Marshal(wsFromClient.RWPState)

				pbdata, err := proto.Marshal(incomingMessages[0])
				log.Should(err)
				header := make([]byte, 4)                                  // Create a 4-bytes header
				binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
				pbdata = append(header, pbdata...)

				wsslice.Iter(func(w *wsclient) {
					w.msgToClient <- &wsToClient{
						RWPASCIIToPanel:    strings.Join(helpers.InboundMessagesToRawPanelASCIIstrings(incomingMessages), "\n"),
						RWPJSONToPanel:     string(stateAsJsonString),
						RWPProtobufToPanel: prettyHexPrint(pbdata),
					}
				})

				incoming <- incomingMessages
			}
		}
	}
}

func setupRoutes() {
	http.HandleFunc("/", homePage)
	http.HandleFunc("/ws", wsEndpoint)
}

//go:embed resources
var embeddedFS embed.FS

// Read contents from ordinary or embedded file
func ReadResourceFile(fileName string) []byte {
	fileName = strings.ReplaceAll(fileName, "\\", "/")
	byteValue, err := embeddedFS.ReadFile(fileName)
	log.Should(err)
	return byteValue
}
