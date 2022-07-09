package main

/*
	This is a convenience rendering in HTML of the topology in table form.
	The logic is useful as reference interpretation of how to deal with the various information in the topology
*/

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	su "github.com/SKAARHOJ/ibeam-lib-utils"
	topology "github.com/SKAARHOJ/rawpanel-lib/topology"
)

// Generates an HTML overview table
func GenerateTopologyOverviewTable(topologyJSON string, theMap map[uint32]uint32) string {

	// Reading JSON topology:
	var topology topology.Topology
	json.Unmarshal([]byte(topologyJSON), &topology)

	htmlOut := `<table border="0" cellpadding="1", cellspacing="1" id="TopologyTable">`

	htmlOut += `<tr class="header">
		<td>HWC id</td>
		<td>Text</td>
		<td>In</td>
		<td>Out</td>
		<td>Ext</td>
		<td>Display</td>
		<td>SubIdx</td>
		<td>SubEl#</td>
		<td>Rotate</td>
		<td>TypeOvr</td>
		<td>Map</td>
		<td>TypeIdx</td>
		<td>Descr.</td>
		<td>Events</td>
		</tr>`

	for _, HWcDef := range topology.HWc {
		typeDef := topology.GetTypeDefWithOverride(&HWcDef)

		htmlOut += fmt.Sprintf(`<tr id="HWc%d" class="%s">
		<td>%d</td>
		<td>%s</td>
		<td>%s</td>
		<td>%s</td>
		<td>%s</td>
		<td>%s</td>
		<td>%s</td>
		<td style="text-align: center">%s</td>
		<td>%s</td>
		<td>%s</td>
		<td>%s</td>
		<td style="text-align: center">%d</td>
		<td>%s</td>
		<td id="HWc%d_Events"></td>
		</tr>`,
			HWcDef.Id,
			su.Qstr(theMap[HWcDef.Id] == 0, "mappedOut", ""),
			HWcDef.Id,
			HWcDef.Txt,
			GenerateTopologyOverviewTableIn(&typeDef, typeDef.In),
			GenerateTopologyOverviewTableOut(&typeDef, typeDef.Out),
			GenerateTopologyOverviewTableExt(&typeDef, typeDef.Ext),
			GenerateTopologyOverviewTableDisplay(typeDef.Disp),
			GenerateTopologyOverviewTableSubIdx(&typeDef, typeDef.Subidx),
			su.Qstr(len(typeDef.Sub) > 0, strconv.Itoa(len(typeDef.Sub)), ""),
			su.Qstr(typeDef.Rotate != 0, fmt.Sprintf("%.2f", typeDef.Rotate), ""),
			GenerateTopologyOverviewTableTypeOverride(HWcDef.TypeOverride),
			su.Qstr(theMap[HWcDef.Id] == HWcDef.Id, "", su.Qstr(theMap[HWcDef.Id] == 0, "Disabled", fmt.Sprintf("%d", theMap[HWcDef.Id]))),
			HWcDef.Type,
			typeDef.Desc,
			HWcDef.Id,
		)
	}
	htmlOut += `</table>`

	return htmlOut
}

func GenerateTopologyOverviewTableIn(typeDef *topology.TopologyHWcTypeDef, in string) string {
	if in == "" || in == "-" {
		return "-"
	}

	output := ""

	parts := strings.Split(in+",", ",")

	switch strings.TrimSpace(parts[0]) {
	case "b4":
		output = "Button, Four-way"
	case "b2h":
		output = "Button, Horizontal Two-way"
	case "b4v":
		output = "Button, Vertical Two-way"
	case "b":
		output = "Button"
	case "gpi":
		output = "GPI input"
	case "p":
		output = "Encoder (pulsed input)"
	case "pb":
		output = "Encoder w/button"
	case "av":
		output = "Analog component, vertical"
	case "ah":
		output = "Analog component, horizontal"
	case "ar":
		output = "Analog component, rotational"
	case "a":
		output = "Analog component, generic"
	case "iv":
		output = "Intensity component, vertical"
	case "ih":
		output = "Intensity component, horizontal"
	case "ir":
		output = "Intensity component, rotational"
	case "i":
		output = "Intensity component, generic"
	default:
		output = "Unknown"
	}

	if parts[1] != "" {
		output += " + extra: " + parts[1]
	}

	return output + fmt.Sprintf(" (%s)", in)
}

func GenerateTopologyOverviewTableOut(typeDef *topology.TopologyHWcTypeDef, out string) string {
	if out == "" || out == "-" {
		return "-"
	}

	output := ""

	parts := strings.Split(out+",", ",")

	switch strings.TrimSpace(parts[0]) {
	case "gpo":
		output = "GPO output"
	case "mono":
		output = "mono LED"
	case "rg":
		output = "Red/Green LED"
	case "rgb":
		output = "RGB LED"
	default:
		output = "Unknown"
	}

	if parts[1] != "" {
		output += " + extra: " + parts[1]
	}

	return output + fmt.Sprintf(" (%s)", out)
}

func GenerateTopologyOverviewTableExt(typeDef *topology.TopologyHWcTypeDef, ext string) string {
	if ext == "" {
		return ""
	}

	output := ""

	switch strings.TrimSpace(ext) {
	case "pos":
		output = "Self Positioning"
	case "steps":
		min := 10000
		max := -10000
		for _, obj := range typeDef.Sub {
			if obj.Idx < min {
				min = obj.Idx
			}
			if obj.Idx > max {
				max = obj.Idx
			}
		}
		output = fmt.Sprintf("%d Steps", max-min+1)

	}
	return output + fmt.Sprintf(" (%s)", ext)
}
func GenerateTopologyOverviewTableSubIdx(typeDef *topology.TopologyHWcTypeDef, Subidx int) string {
	if Subidx < 0 {
		return "" // A Subidx below zero means it's "disabled"
	}

	if Subidx < len(typeDef.Sub) { // A Subidx that would reference an existing
		return fmt.Sprintf("Yes, index %d", Subidx)
	}

	if Subidx > 0 { // A Subidx larger than zero and not being an index would be invalid.
		return "Invalid"
	}
	return ""
}

func GenerateTopologyOverviewTableDisplay(Disp *topology.TopologyHWcTypeDef_Display) string {
	if Disp == nil {
		return ""
	}
	parts := []string{fmt.Sprintf("%dx%d", Disp.W, Disp.H)}

	switch Disp.Type {
	case "color":
		parts = append(parts, "RGB")
	case "gray":
		parts = append(parts, "Gray")
	case "text":
		parts = []string{"Text"}
		if Disp.H > 0 {
			parts = append(parts, fmt.Sprintf("%d lines", Disp.H))
		}
		if Disp.W > 0 {
			parts = append(parts, fmt.Sprintf("%d chars", Disp.W))
		}
	default:
		parts = append(parts, "Mono")
	}

	if Disp.Border > 0 {
		parts = append(parts, fmt.Sprintf("Text border=%d", Disp.Border))
	}

	switch Disp.Shrink & 3 {
	case 1:
		parts = append(parts, "Shrink Right")
	case 2:
		parts = append(parts, "Shrink Bottom")
	case 3:
		parts = append(parts, "Shrink Right+Bottom")
	}

	return strings.Join(parts, ", ")
}

func GenerateTopologyOverviewTableTypeOverride(TypeOverride *topology.TopologyHWcTypeDef) string {
	if TypeOverride == nil {
		return ""
	}

	parts := []string{}

	if TypeOverride.W > 0 {
		parts = append(parts, fmt.Sprintf("W:%d", TypeOverride.W))
	}
	if TypeOverride.H > 0 {
		parts = append(parts, fmt.Sprintf("H:%d", TypeOverride.H))
	}
	if TypeOverride.Rotate != 0 {
		parts = append(parts, fmt.Sprintf("Rotate:%.2f", TypeOverride.Rotate))
	}
	if TypeOverride.Out != "" {
		parts = append(parts, fmt.Sprintf("Out:%s", TypeOverride.Out))
	}
	if TypeOverride.In != "" {
		parts = append(parts, fmt.Sprintf("In:%s", TypeOverride.In))
	}
	if TypeOverride.Ext != "" {
		parts = append(parts, fmt.Sprintf("Ext:%s", TypeOverride.Ext))
	}
	if TypeOverride.Subidx > 0 {
		parts = append(parts, fmt.Sprintf("Subidx:%v", TypeOverride.Subidx))
	}
	if TypeOverride.Disp != nil {
		parts = append(parts, "[DISP]")
	}
	if len(TypeOverride.Sub) > 0 {
		parts = append(parts, "[SubElements]")
	}

	return strings.Join(parts, ", ")
}
