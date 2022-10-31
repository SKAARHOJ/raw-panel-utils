package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	ipbase "./ipbase"
	"github.com/google/uuid"
)

var globalConnectionCounter = 0
var pauseConsoleOutput = false

type HWcomponent struct {
	Id   uint32 `json:"id"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
	Txt  string `json:"txt"`
	Type uint32 `json:"type"`
}

type HWcTypeDef struct {
	W      int               `json:"w"`
	H      int               `json:"h"`
	Out    string            `json:"out"`
	In     string            `json:"in"`
	Desc   string            `json:"desc"`
	Subidx int               `json:"subidx,omitempty"`
	Disp   HWcTypeDefDisplay `json:"disp,omitempty"`
	Sub    []interface{}     `json:"sub,omitempty"`
}

type HWcTypeDefDisplay struct {
	W      int `json:"w,omitempty"`
	H      int `json:"h,omitempty"`
	Subidx int `json:"subidx,omitempty"`
}

/*
type HWcTypes struct {
	Idx0  HWcTypeDef `json:"1"`
	Idx1  HWcTypeDef `json:"2"`
	Idx2  HWcTypeDef `json:"3"`
	Idx3  HWcTypeDef `json:"4"`
	Idx18 HWcTypeDef `json:"18"`
	Idx23 HWcTypeDef `json:"23"`
}*/

type Panel struct {
	HWc       []HWcomponent
	TypeIndex map[uint32]HWcTypeDef `json:"typeIndex"`
}

func handleConnection(id string, c net.Conn, connMap *sync.Map) {
	// Make sure we close the connection and remote the ID from the connection map in case we exit:
	defer func() {
		c.Close()
		connMap.Delete(id)
	}()

	fmt.Println("System: New Connection from: " + c.RemoteAddr().String() + " (connections: " + strconv.Itoa(globalConnectionCounter) + ")")

	var busy = false
	var panelInitialized = false
	HWCtracker := make(map[int]int)

	connectionReader := bufio.NewReader(c) // Define OUTSIDE the for loop - otherwise it can skip content! (https://stackoverflow.com/questions/46309810/does-readstring-discard-bytes-following-newline). I saw that with the JSON part of Topology Data.

	for {
		netData, err := connectionReader.ReadString('\n')
		if err != nil {
			globalConnectionCounter--
			if err == io.EOF {
				fmt.Println("System: " + c.RemoteAddr().String() + " disconnected (connections: " + strconv.Itoa(globalConnectionCounter) + ")")
			} else {
				fmt.Println(err)
			}
			return
		}

		inputFromClient := strings.TrimSpace(string(netData))
		if !pauseConsoleOutput {
			fmt.Printf("%-21v", c.RemoteAddr().String())
			fmt.Println("> " + inputFromClient)
		}

		outputToClient := []string{}

		switch inputFromClient {
		case "list":
			outputToClient = []string{"", "ActivePanel=1", "list"}
			if !panelInitialized {
				outputToClient = append(outputToClient, "PanelTopology?")
			}
			panelInitialized = true
			if panelInitialized {
				// Dummy
			}
		case "ack":
		case "BSY":
			busy = true
		case "RDY":
			busy = false
		case "ping":
			outputToClient = []string{"ack"}
		default:
			// Looking for special patterns, like "map", whether the input is "map":
			r, _ := regexp.Compile("^map=([0-9]+):([0-9]+)$")
			//svg, _ := regexp.Compile("^_panelTopology_svgbase=(.*)$")
			jsonRegex, _ := regexp.Compile("^_panelTopology_HWC=(.*)$")
			if r.MatchString(inputFromClient) {
				HWcServer, _ := strconv.Atoi(r.FindStringSubmatch(inputFromClient)[2]) // Extract the HWc number of the keypress from the match
				HWcClient, _ := strconv.Atoi(r.FindStringSubmatch(inputFromClient)[1]) // Extract the HWc number of the keypress from the match
				HWCtracker[HWcClient] = HWcServer
			} else if jsonRegex.MatchString(inputFromClient) {
				jsonString := jsonRegex.FindStringSubmatch(inputFromClient)[1] // JSON content

				// Parse if into a struct (mostly, except the typeIndex, which is a map and requires some special care)
				var panelInformation Panel
				json.Unmarshal([]byte(jsonString), &panelInformation)
				fmt.Println(panelInformation)
				/*		for typeIndexKey, typeIndexDefinition := range panelInformation.TypeIndex.(map[string]interface{}) {
							var typeIndexDefinitionAsStruct HWcTypeDef
							mapstructure.Decode(typeIndexDefinition, &typeIndexDefinitionAsStruct)
							panelInformation.TypeIndex.(map[string]interface{})[typeIndexKey] = typeIndexDefinitionAsStruct
						}
				*/
				//Writes back the JSON:
				//				bolB, _ := json.MarshalIndent(panelInformation, "", "  ")
				//				fmt.Println(string(bolB))

				// Using this information to write to the display tiles what resolution they have:
				myTypeIndex := panelInformation.TypeIndex
				for _, HWc := range panelInformation.HWc {
					displayCfg := myTypeIndex[HWc.Type].Disp
					if displayCfg.H > 0 && displayCfg.W > 0 {
						outputToClient = append(outputToClient, fmt.Sprintf("HWCt#%d=|||HWC#%d|1|%dx%d", HWc.Id, HWc.Id, displayCfg.H, displayCfg.W))
					}
				}

			} else {
				if busy {
				}
			}
		}

		// Sending output lines:
		if len(outputToClient) > 0 {
			for i, _ := range outputToClient {
				if !pauseConsoleOutput {
					fmt.Printf("%-21v", c.RemoteAddr().String())
					fmt.Println("< " + outputToClient[i])
				}

				c.Write([]byte(outputToClient[i] + "\n"))
			}
		}
	}
}

func main() {

	// Welcome message!
	fmt.Println("Welcome to Raw Panel Server! Made by Kasper Skaarhoj 2020")
	fmt.Println("Raw Panels will connect")
	fmt.Println("Ready to accept TCP connections from a SKAARHOJ panel on port 9923\n")

	// Listening for new connections:
	l, err := net.Listen("tcp4", ":9923")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	// Using sync.Map to not deal with concurrency slice/map issues	(https://golangforall.com/en/post/golang-tcp-server-chat.html)
	var connMap = &sync.Map{}

	// Keyboard input listener:
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)

			if len(text) == 0 { // Empty lines enables/disables console output:
				pauseConsoleOutput = !pauseConsoleOutput
				fmt.Print(ipbase.QStr(!pauseConsoleOutput, "Console output enabled", "Console output disabled, ready for input:\n[All clients] < "))
			} else {
				// Enable console output again.
				pauseConsoleOutput = false
				fmt.Println("Console output enabled\n")

				// Traverse over connections and send the keyboard input out.
				connMap.Range(func(key, value interface{}) bool {
					if conn, ok := value.(net.Conn); ok {
						conn.Write([]byte(text + "\n"))

						fmt.Printf("%-21v", conn.RemoteAddr().String())
						fmt.Println("< " + text)
					}
					return true
				})
			}
		}
	}()

	for {
		// Whenever something tries to connect, ask a new goroutine to handle it:
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		// Register a new connection here:
		id := uuid.New().String()
		connMap.Store(id, c)

		// Start handler for new connection:
		go handleConnection(id, c, connMap)
		globalConnectionCounter++
	}
}
