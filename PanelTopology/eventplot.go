package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	su "github.com/SKAARHOJ/ibeam-lib-utils"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"golang.org/x/exp/slices"
)

// Plots events in a little square window.

var PlotEvents = []*rwp.HWCEvent{}
var PlotEventTimes = []time.Time{}
var PlotHWCs = []int{}
var PlotTimeStart = time.Now()
var TriggerType string

func eventPlot(Event *rwp.HWCEvent) {

	// Renew:
	if !slices.Contains(PlotHWCs, int(Event.HWCID)) {
		triggerType := getTypeCode(Event.HWCID)
		if ((triggerType == "ih" && TriggerType == "iv") || (triggerType == "iv" && TriggerType == "ih")) && len(PlotHWCs) == 1 {
			PlotHWCs = append(PlotHWCs, int(Event.HWCID)) // Add a second dimension to joystick
		} else if len(PlotHWCs) != 2 || (triggerType != "ih" && triggerType != "iv" && triggerType != "ir") {
			PlotHWCs = []int{int(Event.HWCID)}
			PlotEvents = []*rwp.HWCEvent{}
			PlotTimeStart = time.Now()
			PlotEventTimes = []time.Time{}
			TriggerType = triggerType
		}
	}

	// Add to events:
	PlotEvents = append(PlotEvents, Event)
	PlotEventTimes = append(PlotEventTimes, time.Now())

	// X-scaling (for time)
	scaleX := int(PlotEventTimes[len(PlotEventTimes)-1].Sub(PlotTimeStart) / time.Millisecond)
	if scaleX < 1000 {
		scaleX = 1000
	}
	period := 0
	if len(PlotEventTimes) > 1 {
		period = int(PlotEventTimes[len(PlotEventTimes)-1].Sub(PlotEventTimes[len(PlotEventTimes)-2]) / time.Millisecond)
	}

	// Draw:
	innerSVG := ""
	x := 50
	y := 50
	if len(PlotHWCs) == 2 {
		x = 550
		y = 550
	}
	stepSize := 0

	// Draw time scale:
	for a := 0; a <= scaleX; a += 1000 {
		innerSVG += fmt.Sprintf(`<line x1="%d" y1="1050" x2="%d" y2="1080" stroke="#cccccc" stroke-width="10" />`, a*1000/scaleX+50, a*1000/scaleX+50)
	}

	// Draw event points:
	for i, event := range PlotEvents {
		newX := int(PlotEventTimes[i].Sub(PlotTimeStart)/time.Millisecond)*1000/scaleX + 50
		newY := 0

		if event.Speed != nil {
			if (TriggerType == "ih" || TriggerType == "iv") && len(PlotHWCs) == 2 {
				if slices.Contains(PlotHWCs, int(event.HWCID)) {
					if (TriggerType == "ih" && int(event.HWCID) == PlotHWCs[0]) || (TriggerType == "iv" && int(event.HWCID) == PlotHWCs[1]) {
						newX = int(event.Speed.Value) + 500 + 50
						stepSize = int(math.Abs(float64(newX) - float64(x)))
						newY = y
					} else {
						newX = x
						newY = int(-event.Speed.Value) + 500 + 50
						stepSize = int(math.Abs(float64(newY) - float64(y)))
					}
					innerSVG += fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#333333" stroke-width="1" />`, x, y, newX, newY)
					innerSVG += fmt.Sprintf(`<circle cx="%d" cy="%d" r="4" fill="#333333" />`, newX, newY)
					x = newX
					y = newY
				}
			} else {
				newY := int(event.Speed.Value) + 500 + 50
				stepSize = int(math.Abs(float64(newY) - float64(y)))
				innerSVG += fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#333333" stroke-width="1" />`, x, y, newX, newY)
				innerSVG += fmt.Sprintf(`<circle cx="%d" cy="%d" r="4" fill="#333333" />`, newX, newY)
				x = newX
				y = newY
			}
		}
		if event.Absolute != nil {
			newY = int(event.Absolute.Value) + 50
			stepSize = int(math.Abs(float64(newY) - float64(y)))
			innerSVG += fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#333333" stroke-width="1" />`, x, y, newX, newY)
			innerSVG += fmt.Sprintf(`<circle cx="%d" cy="%d" r="4" fill="#333333" />`, newX, newY)
			x = newX
			y = newY
		}
		if event.Binary != nil {
			newY = int(su.Qint(event.Binary.Pressed, 1000, 0)) + 50
			innerSVG += fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#333333" stroke-width="1" />`, x, y, newX, y)
			innerSVG += fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#333333" stroke-width="1" />`, newX, y, newX, newY)
			innerSVG += fmt.Sprintf(`<circle cx="%d" cy="%d" r="4" fill="#333333" />`, newX, newY)
			x = newX
			y = newY
		}
		if event.Pulsed != nil {
			newY = int(-event.Pulsed.Value*100) + 50 + 500
			innerSVG += fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="#333333" stroke-width="1" />`, newX, 550, newX, newY)
			innerSVG += fmt.Sprintf(`<circle cx="%d" cy="%d" r="4" fill="#333333" />`, newX, newY)
			x = newX
			y = newY
		}
	}

	if stepSize > 0 {
		innerSVG += fmt.Sprintf(`<text x="700" y="1120" text-anchor="Left" fill="#000" font-size="35" font-family="sans-serif">Stepsize: %d / Steps: %d</text>`, stepSize, 1000/stepSize)
	}
	if period > 0 {
		innerSVG += fmt.Sprintf(`<text x="40" y="1120" text-anchor="Left" fill="#000" font-size="35" font-family="sans-serif">Distance: %dms / Frequency: %.1fHz</text>`, period, 1000/float64(period))
	}

	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1100 1150" width="100%">
		<text x="550" y="30" text-anchor="Middle" fill="#000" font-size="40" font-weight="bold" font-family="sans-serif">Event Scope HWC#` + joinInt(PlotHWCs, ",") + `:</text>
		<rect x="50" y="50" width="1000" height="1000" stroke="#cccccc" stroke-width="10" fill="none" />
		` + innerSVG + `
	</svg>`

	eventMessage := &wsToClient{
		ControlBlock: svg,
	}
	wsslice.Iter(func(w *wsclient) { w.msgToClient <- eventMessage })
}

func getTypeCode(hwc uint32) string {
	for _, HWcDef := range TopologyData.HWc {
		if HWcDef.Id == hwc {
			typeDef := TopologyData.GetTypeDefWithOverride(&HWcDef)

			outSplit := strings.Split(typeDef.In, ",")
			return strings.TrimSpace(outSplit[0])
		}
	}
	return ""
}

func joinInt(integers []int, char string) string {
	parts := []string{}
	for _, integer := range integers {
		parts = append(parts, strconv.Itoa(integer))
	}
	return strings.Join(parts, char)
}
