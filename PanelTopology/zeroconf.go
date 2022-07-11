package main

import (
	"context"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/grandcat/zeroconf"
	log "github.com/s00500/env_logger"
)

type ZeroconfEntry struct {
	IPaddr    net.IP
	Model     string
	Serial    string
	Name      string
	Protocol  string
	Port      int
	SessionId int
	Updated   bool
}

var ZeroconfEntries []*ZeroconfEntry
var ZeroconfEntriesMu sync.Mutex

func runZeroConfSearch() {
	sessionId := 0
	for {
		sessionId++
		zeroconfSearchSession(sessionId)
	}
}

func zeroconfSearchSession(sessionId int) {
	// Discover all SKAARHOJ raw panel services on the network (_skaarhoj-rwp._tcp)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			addEntry(entry, sessionId)
			//log.Println(log.Indent(entry), entry.AddrIPv4)
		}
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err = resolver.Browse(ctx, "_skaarhoj-rwp._tcp", "local.", entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}
	<-ctx.Done()

	// Remove old entries
	ZeroconfEntriesMu.Lock()
	removedSome := false
	for a := len(ZeroconfEntries); a > 0; a-- {
		i := a - 1
		if ZeroconfEntries[i].SessionId < sessionId {
			ZeroconfEntries = append(ZeroconfEntries[:i], ZeroconfEntries[i+1:]...)
			removedSome = true
		}
	}
	if removedSome {
		//fmt.Println("Removed some entries, now the list looks like this:")
		//log.Println(log.Indent(ZeroconfEntries))
		wsslice.Iter(func(w *wsclient) {
			w.msgToClient <- &wsToClient{
				ZeroconfEntries: ZeroconfEntries,
				Time:            getTimeString(),
			}
		})
	}
	ZeroconfEntriesMu.Unlock()
}

func addEntry(addThisEntry *zeroconf.ServiceEntry, sesId int) {
	ZeroconfEntriesMu.Lock()
	defer ZeroconfEntriesMu.Unlock()

	if len(addThisEntry.AddrIPv4) > 0 {
		for i, entry := range ZeroconfEntries {
			if entry.IPaddr.String() == addThisEntry.AddrIPv4[0].String() &&
				entry.Port == addThisEntry.Port {

				//fmt.Printf("Updating %v\n", zeroconfEntries[i])
				ZeroconfEntries[i].SessionId = sesId
				ZeroconfEntries[i].Updated = true
				ZeroconfEntries = sortEntries(ZeroconfEntries)
				return
			}
		}

		// We are here because the entry was not found, so we add it:
		// fmt.Printf("New %v\n", addThisEntry.AddrIPv4)
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

		ZeroconfEntries = append([]*ZeroconfEntry{&ZeroconfEntry{
			IPaddr:    addThisEntry.AddrIPv4[0],
			Port:      addThisEntry.Port,
			Serial:    parts[0],
			Model:     parts[1],
			Name:      devicename,
			Protocol:  protocol,
			SessionId: sesId,
		}}, ZeroconfEntries...)
		ZeroconfEntries = sortEntries(ZeroconfEntries)

		//fmt.Println("Added an entry, now the list looks like this:")
		//log.Println(log.Indent(ZeroconfEntries))

		wsslice.Iter(func(w *wsclient) {
			w.msgToClient <- &wsToClient{
				ZeroconfEntries: ZeroconfEntries,
				Time:            getTimeString(),
			}
		})
	}
}

func sortEntries(zEntries []*ZeroconfEntry) []*ZeroconfEntry {
	sort.SliceStable(zEntries, func(i, j int) bool {
		return zEntries[i].Model < zEntries[j].Model
	})

	return zEntries
}
