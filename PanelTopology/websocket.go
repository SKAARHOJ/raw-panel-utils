package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"github.com/gorilla/websocket"
	log "github.com/s00500/env_logger"
)

type wsMessage struct {
	Title         string `json:",omitempty"`
	Model         string `json:",omitempty"`
	Serial        string `json:",omitempty"`
	SvgIcon       string `json:",omitempty"`
	TopologyTable string `json:",omitempty"`
	TopologyJSON  string `json:",omitempty"`
	Time          string `json:",omitempty"`

	PanelEvent *rwp.HWCEvent `json:",omitempty"`
}

type wsclient struct {
	msgToClient chan *wsMessage
	quit        chan bool
}

type threadSafeSlice struct {
	sync.Mutex
	wsclients []*wsclient
}

func (slice *threadSafeSlice) Push(w *wsclient) {
	slice.Lock()
	defer slice.Unlock()
	slice.wsclients = append(slice.wsclients, w)
}

func (slice *threadSafeSlice) Iter(routine func(*wsclient)) {
	slice.Lock()
	defer slice.Unlock()
	for _, wsclient := range slice.wsclients {
		routine(wsclient)
	}
}

var slice threadSafeSlice

func (w *wsclient) Start(ws *websocket.Conn) {
	w.msgToClient = make(chan *wsMessage, 10) // some buffer size to avoid blocking
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
	slice.Push(ww)

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
			slice.Iter(func(w *wsclient) { w.msgToClient <- lastState })
			lastStateMu.Unlock()
		default:
			log.Println("Received from websocket: ", string(p))
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
