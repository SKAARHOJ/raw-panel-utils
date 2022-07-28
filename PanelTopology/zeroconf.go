package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	topology "github.com/SKAARHOJ/rawpanel-lib/topology"
	"github.com/grandcat/zeroconf"
	log "github.com/s00500/env_logger"
	"github.com/tatsushid/go-fastping"
	"go.uber.org/atomic"
)

type ZeroconfEntry struct {
	sync.Mutex

	IPaddr                 net.IP
	Model                  string
	Serial                 string
	Name                   string
	Protocol               string
	Port                   int
	SessionId              int
	IsNew                  bool
	AggressiveQueryStarted bool
	RawPanelDetails        *RawPanelDetails
	PingTime               int
	SkaarOS                string

	createdTime time.Time
}

type RawPanelDetails struct {
	FriendlyName     string
	Serial           string
	Model            string
	SerialModelError bool
	SoftwareVersion  string
	Platform         string
	BluePillReady    string
	MaxClients       uint32
	LockedToIPs      string
	TotalHWCs        int
	PanelTopologySVG string
	Connections      string
	BootsCount       int
	SessionUptime    string
	TotalUptime      string
	ScreenSaveOnTime string
	BinaryConnection bool
	ErrorMsg         string
	Msg              string
	PingTime         int

	DeltaTime int
}

var ZeroconfEntries []*ZeroconfEntry
var ZeroconfEntriesMu sync.Mutex
var UpdateWS atomic.Bool

func runZeroConfSearch() {

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for {
			<-ticker.C
			if UpdateWS.Load() {
				UpdateWS.Store(false)

				ZeroconfEntriesMu.Lock()
				wsslice.Iter(func(w *wsclient) {
					w.msgToClient <- &wsToClient{
						ZeroconfEntries: ZeroconfEntries,
						Time:            getTimeString(),
					}
				})
				ZeroconfEntriesMu.Unlock()
			}
		}
	}()

	sessionId := 0
	for {
		sessionId++
		zeroconfSearchSession(sessionId)
	}
}

func zeroconfSearchSession(sessionId int) {

	// Discover SKAARHOJ raw panel services on the network (_skaarhoj-rwp._tcp)
	resolverRwp, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	entriesRwp := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			addRWPEntry(entry, sessionId)
		}
	}(entriesRwp)

	ctxRwp, cancelRwp := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelRwp()
	err = resolverRwp.Browse(ctxRwp, "_skaarhoj-rwp._tcp", "local.", entriesRwp)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}

	// Discover all SKAARHOJ devices on the network (_skaarhoj._tcp)
	resolverAll, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	entriesAll := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			addGenericEntry(entry, sessionId)
		}
	}(entriesAll)

	ctxAll, cancelAll := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelAll()
	err = resolverAll.Browse(ctxAll, "_skaarhoj._tcp", "local.", entriesAll)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}

	// Wait for both to finish:
	<-ctxAll.Done()
	<-ctxRwp.Done()

	// Remove old entries
	ZeroconfEntriesMu.Lock()
	for a := len(ZeroconfEntries); a > 0; a-- {
		i := a - 1
		if ZeroconfEntries[i].SessionId+1 < sessionId {
			ZeroconfEntries = append(ZeroconfEntries[:i], ZeroconfEntries[i+1:]...)
			UpdateWS.Store(true)
		}
	}
	ZeroconfEntriesMu.Unlock()
}

func addRWPEntry(addThisEntry *zeroconf.ServiceEntry, sesId int) {
	ZeroconfEntriesMu.Lock()
	defer ZeroconfEntriesMu.Unlock()

	if len(addThisEntry.AddrIPv4) > 0 {
		// Derive some info here:
		parts := strings.Split(addThisEntry.HostName+"..", ".")
		devicename := ""
		protocol := ""
		for _, str := range addThisEntry.Text {
			dParts := strings.SplitN(str, "devicename=", 2)
			if len(dParts) == 2 {
				devicename = dParts[1]
			}
			dParts = strings.SplitN(str, "protocol=", 2)
			if len(dParts) == 2 {
				protocol = dParts[1]
			}
		}

		// Search for existing to update:
		for i, entry := range ZeroconfEntries {
			if entry.IPaddr.String() == addThisEntry.AddrIPv4[0].String() &&
				entry.Port == addThisEntry.Port {

				//fmt.Printf("Updating %v\n", zeroconfEntries[i])
				ZeroconfEntries[i].Lock()

				ZeroconfEntries[i].IPaddr = addThisEntry.AddrIPv4[0]
				ZeroconfEntries[i].Port = addThisEntry.Port
				ZeroconfEntries[i].Serial = parts[0]
				ZeroconfEntries[i].Model = parts[1]
				ZeroconfEntries[i].Name = devicename
				ZeroconfEntries[i].Protocol = protocol
				ZeroconfEntries[i].SessionId = sesId

				ZeroconfEntries[i].IsNew = time.Now().Before(ZeroconfEntries[i].createdTime.Add(time.Second * 5))

				if *AggressiveQuery && !ZeroconfEntries[i].AggressiveQueryStarted {
					go rawPanelInquery(ZeroconfEntries[i])
				}

				ZeroconfEntries[i].Unlock()
				ZeroconfEntries = sortEntries(ZeroconfEntries)
				UpdateWS.Store(true)

				// Pingtime:
				ipAddr := addThisEntry.AddrIPv4[0].String()
				theEntry := ZeroconfEntries[i]
				go func() {
					pingTime := getPingTimes(ipAddr)
					theEntry.Lock()
					theEntry.PingTime = pingTime
					theEntry.Unlock()
					UpdateWS.Store(true)
				}()

				return
			}
		}

		// We are here because the entry was not found, so we add it:
		newEntry := &ZeroconfEntry{
			IPaddr:      addThisEntry.AddrIPv4[0],
			Port:        addThisEntry.Port,
			Serial:      parts[0],
			Model:       parts[1],
			Name:        devicename,
			Protocol:    protocol,
			SessionId:   sesId,
			IsNew:       true,
			createdTime: time.Now(),
		}
		ZeroconfEntries = append([]*ZeroconfEntry{newEntry}, ZeroconfEntries...)
		ZeroconfEntries = sortEntries(ZeroconfEntries)

		if *AggressiveQuery {
			go rawPanelInquery(newEntry)
		}

		// Pingtime:
		go func() {
			pingTime := getPingTimes(addThisEntry.AddrIPv4[0].String())
			newEntry.Lock()
			newEntry.PingTime = pingTime
			newEntry.Unlock()
			UpdateWS.Store(true)
		}()

		UpdateWS.Store(true)
	}
}

func addGenericEntry(addThisEntry *zeroconf.ServiceEntry, sesId int) {
	ZeroconfEntriesMu.Lock()
	defer ZeroconfEntriesMu.Unlock()

	if len(addThisEntry.AddrIPv4) > 0 {

		// Derive some info here:
		parts := strings.Split(addThisEntry.HostName+"..", ".")
		skaarOS := ""
		devicename := ""
		for _, str := range addThisEntry.Text {
			dParts := strings.SplitN(str, "devicename=", 2)
			if len(dParts) == 2 {
				devicename = dParts[1]
			}
			dParts = strings.SplitN(str, "skaarOS=", 2)
			if len(dParts) == 2 {
				skaarOS = dParts[1]
			}
		}

		// Search for existing to update:
		foundIP := false
		foundOtherPort := false
		foundGeneric := false
		for i, entry := range ZeroconfEntries {
			if entry.IPaddr.String() == addThisEntry.AddrIPv4[0].String() {

				// For any port, update skaarOS:
				ZeroconfEntries[i].Lock()
				ZeroconfEntries[i].SkaarOS = skaarOS
				ZeroconfEntries[i].IsNew = time.Now().Before(ZeroconfEntries[i].createdTime.Add(time.Second * 5))
				ZeroconfEntries[i].Unlock()

				// Pingtime and session for true non-rwp devices:
				if entry.Port == -1 {
					ZeroconfEntries[i].Lock()
					ZeroconfEntries[i].SessionId = sesId
					ZeroconfEntries[i].Unlock()

					ipAddr := addThisEntry.AddrIPv4[0].String()
					theEntry := ZeroconfEntries[i]
					go func() {
						pingTime := getPingTimes(ipAddr)
						theEntry.Lock()
						theEntry.PingTime = pingTime
						theEntry.Unlock()
						UpdateWS.Store(true)
					}()
					foundGeneric = true
				} else {
					foundOtherPort = true
				}

				foundIP = true
				UpdateWS.Store(true)
			}
		}

		// Remove generic entry if other port was found:
		if foundOtherPort && foundGeneric {
			for i, entry := range ZeroconfEntries {
				if entry.IPaddr.String() == addThisEntry.AddrIPv4[0].String() && entry.Port == -1 {
					ZeroconfEntries = append(ZeroconfEntries[:i], ZeroconfEntries[i+1:]...)
					break
				}
			}
		} else if !foundIP { // Otherwise, add a new generic entry:
			// We are here because the entry was not found, so we add it:
			newEntry := &ZeroconfEntry{
				IPaddr:      addThisEntry.AddrIPv4[0],
				Port:        -1,
				Serial:      parts[0],
				Model:       parts[1],
				Name:        devicename,
				SessionId:   sesId,
				IsNew:       true,
				createdTime: time.Now(),
			}
			ZeroconfEntries = append([]*ZeroconfEntry{newEntry}, ZeroconfEntries...)
			ZeroconfEntries = sortEntries(ZeroconfEntries)

			// Pingtime:
			go func() {
				pingTime := getPingTimes(addThisEntry.AddrIPv4[0].String())
				newEntry.Lock()
				newEntry.PingTime = pingTime
				newEntry.Unlock()
				UpdateWS.Store(true)
			}()

			UpdateWS.Store(true)
		}
	}
}

func sortEntries(zEntries []*ZeroconfEntry) []*ZeroconfEntry {
	sort.SliceStable(zEntries, func(i, j int) bool {
		return zEntries[i].Model < zEntries[j].Model
	})

	return zEntries
}

// Connects to a panel, asks for information, then disconnects
func rawPanelInquery(newEntry *ZeroconfEntry) {

	// "Lock it" for aggressive search:
	newEntry.Lock()
	newEntry.AggressiveQueryStarted = true
	newEntry.Unlock()

	// Setting IP and port:
	panelIPAndPort := fmt.Sprintf("%s:%d", newEntry.IPaddr.String(), newEntry.Port)
	ownIPusedToConnect := ""
	wasConnected := false

	// Setting up channels etc.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	msgsToPanel := make(chan []*rwp.InboundMessage, 10)
	msgsFromPanel := make(chan []*rwp.OutboundMessage, 10)
	//defer close(msgsToPanel)		// TODO: I want to enable the wait groups and close these channels, but it won't work because of an unsolved by in ConnectToPanel where the ctx.Done() that closes the conn does NOT lead to the reading to cancel. This happens with the ASCII reader...
	//defer close(msgsFromPanel)	// TODO: I want to enable the wait groups and close these channels, but it won't work because of an unsolved by in ConnectToPanel where the ctx.Done() that closes the conn does NOT lead to the reading to cancel. This happens with the ASCII reader...
	var wg sync.WaitGroup

	// Set empty to make sure another session wont start another proces:
	rpDetails := &RawPanelDetails{}

	// On-connect function - asking for a bunch of things...:
	onconnect := func(errorMsg string, binary bool, c net.Conn) {
		fmt.Printf("Connected to %s\n", panelIPAndPort)
		wasConnected = true

		rpDetails.BinaryConnection = binary
		ownIPusedToConnect = strings.Split(c.LocalAddr().String(), ":")[0]

		// Set temporary:
		newEntry.Lock()
		newEntry.RawPanelDetails = &RawPanelDetails{BinaryConnection: binary, Msg: "Connected, fetching details..."}
		newEntry.Unlock()
		UpdateWS.Store(true)

		if errorMsg != "" {
			rpDetails.ErrorMsg = errorMsg
			cancel()
		} else {
			// Send query for stuff we want to know...:
			msgsToPanel <- []*rwp.InboundMessage{
				&rwp.InboundMessage{
					Command: &rwp.Command{
						ActivatePanel:     true,
						SendPanelInfo:     true,
						SendPanelTopology: true,
						GetConnections:    true,
						GetRunTimeStats:   true,
					},
				},
			}
		}
	}
	ondisconnect := func() {
		fmt.Printf("Disconnected from %s\n", panelIPAndPort)
	}

	// Init some vars:
	timeBeforeConnect := time.Now()
	var TotalUptimeGlobal uint32
	readParts := 0b1110001 // Pre-set first bit (and some more for the time being...)
	topologyJSON := ""
	topologySVG := ""

	// Connect to panel:
	go helpers.ConnectToPanel(panelIPAndPort, msgsToPanel, msgsFromPanel, ctx, &wg, onconnect, ondisconnect, nil)

	// Start a timer for timeout:
	ticker := time.NewTicker(time.Second)
	timer1 := time.NewTimer(time.Duration(rand.Intn(5)+5) * time.Second)
	timer2 := time.NewTimer(20 * time.Second) // This timeout of 20000 ms is also used in index.html to paint the time red, just beware of that.

readloop:
	for {
		select {
		case <-ctx.Done():
			break readloop
		case <-ticker.C:
			msgsToPanel <- []*rwp.InboundMessage{
				&rwp.InboundMessage{
					FlowMessage: 1,
				},
			}
		case <-timer1.C:
			log.Println("Resend...", panelIPAndPort)
			msgsToPanel <- []*rwp.InboundMessage{
				&rwp.InboundMessage{
					Command: &rwp.Command{
						SendPanelInfo:     true,
						SendPanelTopology: true,
						GetConnections:    true,
						GetRunTimeStats:   true,
					},
				},
			}
		case <-timer2.C:
			//fmt.Println("expire...", time.Now().Sub(timeBeforeConnect))
			cancel()
			fmt.Println("Timeout ", panelIPAndPort)
			break readloop
		case messagesFromPanel := <-msgsFromPanel:
			for _, msg := range messagesFromPanel {
				if msg.PanelInfo != nil {
					if msg.PanelInfo.Name != "" {
						rpDetails.FriendlyName = msg.PanelInfo.Name
					}
					if msg.PanelInfo.Model != "" {
						readParts |= 1 << 1
						rpDetails.Model = msg.PanelInfo.Model

						newEntry.Lock()
						if newEntry.Model != rpDetails.Model {
							rpDetails.SerialModelError = true
						}
						newEntry.Unlock()
					}
					if msg.PanelInfo.Serial != "" {
						readParts |= 1 << 2
						rpDetails.Serial = msg.PanelInfo.Serial

						newEntry.Lock()
						if newEntry.Serial != rpDetails.Serial {
							rpDetails.SerialModelError = true
						}
						newEntry.Unlock()
					}
					if msg.PanelInfo.SoftwareVersion != "" {
						rpDetails.SoftwareVersion = msg.PanelInfo.SoftwareVersion
					}
					if msg.PanelInfo.Platform != "" {
						rpDetails.Platform = msg.PanelInfo.Platform
					}
					if msg.PanelInfo.BluePillReady {
						rpDetails.BluePillReady = "Yes"
					}
					if msg.PanelInfo.MaxClients != 0 {
						rpDetails.MaxClients = msg.PanelInfo.MaxClients
					}
					if len(msg.PanelInfo.LockedToIPs) != 0 {
						rpDetails.LockedToIPs = strings.Join(msg.PanelInfo.LockedToIPs, ";")
					}
				}

				if msg.PanelTopology != nil {
					if msg.PanelTopology.Json != "" {
						var TopologyData = &topology.Topology{}
						err := json.Unmarshal([]byte(msg.PanelTopology.Json), TopologyData)
						if err != nil {
							log.Println("Topology JSON parsing Error: ", err)
						} else {
							rpDetails.TotalHWCs = len(TopologyData.HWc)
							topologyJSON = msg.PanelTopology.Json
						}
					}
					if msg.PanelTopology.Svgbase != "" {
						topologySVG = msg.PanelTopology.Svgbase
					}
					if topologyJSON != "" && topologySVG != "" {
						readParts |= 1 << 3
						rpDetails.PanelTopologySVG = topology.GenerateCompositeSVG(topologyJSON, topologySVG, nil)
					}
				}

				if msg.Connections != nil {
					readParts |= 1 << 4
					for i, connectedIP := range msg.Connections.Connection {
						if ownIPusedToConnect == connectedIP {
							msg.Connections.Connection = append(msg.Connections.Connection[:i], msg.Connections.Connection[i+1:]...)
							break // Only remote at most one IP address here since we want to know if we are - ourselves, but not this tool - connected.
						}
					}
					rpDetails.Connections = strings.Join(msg.Connections.Connection, ",")
				}

				if msg.RunTimeStats != nil {
					if msg.RunTimeStats.BootsCount > 0 {
						readParts |= 1 << 5
						rpDetails.BootsCount = int(msg.RunTimeStats.BootsCount)
					}
					if msg.RunTimeStats.TotalUptime > 0 {
						readParts |= 1 << 6
						TotalUptimeGlobal = msg.RunTimeStats.TotalUptime // Because we need the value below and these may not come in the same message (they DONT on ASCII version of RWP protocol...)
						rpDetails.TotalUptime = fmt.Sprintf("%dd %dh", msg.RunTimeStats.TotalUptime/60/24, (msg.RunTimeStats.TotalUptime/60)%24)
					}
					if msg.RunTimeStats.SessionUptime > 0 {
						rpDetails.SessionUptime = fmt.Sprintf("%dh %dm", msg.RunTimeStats.SessionUptime/60, msg.RunTimeStats.SessionUptime%60)
					}
					if msg.RunTimeStats.ScreenSaveOnTime > 0 {
						pct := -1
						if TotalUptimeGlobal > 0 {
							pct = 100 * int(msg.RunTimeStats.ScreenSaveOnTime) / int(TotalUptimeGlobal)
						}
						rpDetails.ScreenSaveOnTime = fmt.Sprintf("%dd %dh (%d%%)", msg.RunTimeStats.ScreenSaveOnTime/60/24, (msg.RunTimeStats.ScreenSaveOnTime/60)%24, pct)
					}
				}

				if readParts == 127 {
					cancel()
					fmt.Println("Cancel ", panelIPAndPort)
					break readloop
				}
			}
		}
	}
	timer1.Stop()
	timer2.Stop()
	ticker.Stop()

	fmt.Println("WG waiting ", panelIPAndPort)
	//wg.Wait()		// TODO: I want to enable the wait groups and close the channels above, but it won't work because of an unsolved by in ConnectToPanel where the ctx.Done() that closes the conn does NOT lead to the reading to cancel. This happens with the ASCII reader...
	fmt.Println("WG done ", panelIPAndPort)

	// Time spend:
	if wasConnected {
		rpDetails.DeltaTime = int(time.Now().Sub(timeBeforeConnect) / time.Millisecond)
		newEntry.Lock()
		newEntry.RawPanelDetails = rpDetails
		newEntry.Unlock()
	}

	// Signal to update frontend
	UpdateWS.Store(true)
}

// Sends a UDP based ping to the endpoint and returns the round trip time.
// It's a blocking function as it stands
func getPingTimes(ip string) int {

	p := fastping.NewPinger()
	p.Network("udp")
	ra, err := net.ResolveIPAddr("ip4:icmp", ip)
	if log.Should(err) {
		return -1
	}
	p.AddIPAddr(ra)

	pingTime := -1
	p.OnRecv = func(addr *net.IPAddr, rtt time.Duration) {
		pingTime = int(math.Ceil(float64(rtt) / float64(time.Millisecond)))
	}
	p.Run()
	return pingTime
}
