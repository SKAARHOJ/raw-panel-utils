package main

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	topology "github.com/SKAARHOJ/rawpanel-lib/topology"
)

func getTimeString() string {
	t := time.Now()
	return fmt.Sprintf("%02d:%02d:%02d.%d", t.Hour(), t.Minute(), t.Second(), t.Nanosecond()/1000/1000/100)
}

func prettyHexPrint(data []byte) string {
	output := ""
	for _, b := range data {
		output += hex.EncodeToString([]byte{b}) + " "
	}

	return strings.TrimSpace(output)
}

func getDisplay(hwc uint32) *topology.TopologyHWcTypeDef_Display {
	for _, HWcDef := range TopologyData.HWc {
		if HWcDef.Id == hwc {
			typeDef := TopologyData.GetTypeDefWithOverride(&HWcDef)
			return typeDef.Disp
		}
	}
	return nil
}
