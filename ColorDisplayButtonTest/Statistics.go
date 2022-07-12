package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"
	"golang.org/x/exp/slices"
)

// In this file we are collecting and writing out statistics data

// Shared:
var DataPointsMU sync.Mutex

// Profiling CPU data:
type CPUAnalogDataPoint struct {
	timestamp   uint32
	sampleWidth uint32
	CPUusageAvg uint32
	CPUtempAvg  float32
}
type CPUAnalogEventData struct {
	timestamp uint32
	CPUtemp   float32
	CPUusage  uint32
}

var CPURoundRobinSize = 5
var CPUDataPoints = make(map[int][]CPUAnalogDataPoint)
var CPUAnalogRoundRobin = make(map[int][]CPUAnalogEventData)

func procesSysStatValues(panelNum int, SysStat *rwp.SystemStat) {
	//log.Println(log.Indent(Event))

	DataPointsMU.Lock()
	defer DataPointsMU.Unlock()

	currentTimeMillis := uint32(time.Now().UnixNano() / int64(time.Millisecond))

	if _, exists := StartTimestamp[panelNum]; !exists {
		StartTimestamp[panelNum] = currentTimeMillis
	}

	// Check and create maps:
	if _, exists := CPUDataPoints[panelNum]; !exists {
		CPUDataPoints[panelNum] = make([]CPUAnalogDataPoint, 0)
	}
	if _, exists := CPUAnalogRoundRobin[panelNum]; !exists {
		CPUAnalogRoundRobin[panelNum] = make([]CPUAnalogEventData, 0)
	}

	// Add to round robin or elevate to datapoint:
	if len(CPUAnalogRoundRobin[panelNum]) < CPURoundRobinSize {
		CPUAnalogRoundRobin[panelNum] = append(CPUAnalogRoundRobin[panelNum], CPUAnalogEventData{
			CPUtemp:   SysStat.CPUTemp,
			CPUusage:  SysStat.CPUUsage,
			timestamp: currentTimeMillis,
		})
	} else {
		aDP := CPUAnalogDataPoint{sampleWidth: 0xFFFFFFFF}
		cnt := 0

		for _, aED := range CPUAnalogRoundRobin[panelNum] {
			aDP.CPUtempAvg += aED.CPUtemp
			aDP.CPUusageAvg += aED.CPUusage

			if aED.timestamp > aDP.timestamp {
				aDP.timestamp = aED.timestamp
			}
			if aED.timestamp < aDP.sampleWidth {
				aDP.sampleWidth = aED.timestamp // Collecting lowest value for later calculation
			}
			cnt++
		}
		aDP.CPUtempAvg = float32(math.Round(float64(aDP.CPUtempAvg)/float64(cnt)*10) / 10)
		aDP.CPUusageAvg = uint32(math.Round(float64(aDP.CPUusageAvg)/float64(cnt)*10) / 10)
		aDP.sampleWidth = aDP.timestamp - aDP.sampleWidth
		aDP.timestamp = aDP.timestamp - StartTimestamp[panelNum]

		CPUDataPoints[panelNum] = append(CPUDataPoints[panelNum], aDP)
		CPUAnalogRoundRobin[panelNum] = []CPUAnalogEventData{}

		if panelNum == 1 {
			writeSystemStatStatus()
		}
	}
}

var CPUCSVLines = []string{""}
var CPUCSVfieldsOrder = []string{}

func writeSystemStatStatus() {

	// Add any new panel/HWC pairs if found:
	for panelNum := range CPUDataPoints {
		panelHWC := fmt.Sprintf("_p%d", panelNum)
		if !slices.Contains(CPUCSVfieldsOrder, panelHWC) {
			CPUCSVfieldsOrder = append(CPUCSVfieldsOrder, panelHWC)
		}
	}

	thisLine := []string{}
	header := []string{}
	for _, panelHWC := range CPUCSVfieldsOrder {
		split := strings.Split(panelHWC[2:], ".")
		panelNum, _ := strconv.Atoi(split[0])
		lastDP := CPUAnalogDataPoint{}
		dpLen := len(CPUDataPoints[panelNum])
		if dpLen > 0 {
			lastDP = CPUDataPoints[panelNum][dpLen-1]
		}
		thisLine = append(thisLine, fmt.Sprintf("%d,%.1f,%d", lastDP.timestamp, lastDP.CPUtempAvg, lastDP.CPUusageAvg))
		header = append(header, fmt.Sprintf("%s time, %s temp, %s usage", panelHWC, panelHWC, panelHWC))
	}
	CPUCSVLines[0] = strings.Join(header, ",")
	lineAsString := strings.Join(thisLine, ",")
	CPUCSVLines = append(CPUCSVLines, lineAsString)

	os.WriteFile(filepath.Join(getOutputPath(), "allCPU.csv"), []byte(strings.Join(CPUCSVLines, "\n")), 0644)

	generateCPUHTML(strings.Join(CPUCSVLines, "\n"), "")
	for panelNum := range DataPoints {
		idString := fmt.Sprintf("_p%d", panelNum)
		generateCPUHTML("", idString)
	}
}

func generateCPUHTML(csv string, matchIdString string) {

	series := make(map[string]*ttDataSet)
	colorsOptions := []string{"rgb(238,65,37)", "rgb(0,0,0)", "rgb(206,171,55)", "rgb(21,50,245)", "rgb(238,87,247)", "rgb(52,127,248)", "rgb(148,117,120)", "rgb(109,248,253)", "rgb(108,246,138)", "rgb(169,173,249)", "rgb(159,66,246)", "rgb(240,131,49)", "rgb(191,191,191)", "rgb(220,249,80)"}
	colorsOptionsPointer := 0

	// Add any new panel/HWC pairs if found:
	for panelNum, panelDPs := range CPUDataPoints {
		idString := fmt.Sprintf("_p%d", panelNum)

		if matchIdString == "" || matchIdString == idString {
			// Set up new data series if found:
			borderWidth := 1.5
			series[idString+"T"] = &ttDataSet{
				Label:       idString + " Temp",
				BorderColor: colorsOptions[colorsOptionsPointer%len(colorsOptions)],
				BorderWidth: borderWidth,
			}
			series[idString+"L"] = &ttDataSet{
				Label:       idString + " Load",
				BorderColor: colorsOptions[colorsOptionsPointer%len(colorsOptions)],
				BorderWidth: borderWidth,
				BorderDash:  []int{6, 2},
				PointRadius: []int{0, 0},
			}

			for _, aDP := range panelDPs {
				series[idString+"T"].Data = append(series[idString+"T"].Data, &ttpoint{X: float64(int(aDP.timestamp / 1000)), Y: float64(aDP.CPUtempAvg)})
				series[idString+"L"].Data = append(series[idString+"L"].Data, &ttpoint{X: float64(int(aDP.timestamp / 1000)), Y: float64(int(aDP.CPUusageAvg))})
				series[idString+"L"].YaxisId = "yVal"
			}

			colorsOptionsPointer++
		}
	}

	ordered := []string{}
	for ttDSkey := range series {
		ordered = append(ordered, ttDSkey)
	}
	sort.Strings(ordered)

	outputSeries := []*ttDataSet{}
	for _, ttDS := range ordered {
		outputSeries = append(outputSeries, series[ttDS])
	}
	jsonBytes, _ := json.Marshal(outputSeries)

	PanelNames := []string{}
	for _, n := range PanelName {
		PanelNames = append(PanelNames, n)
	}
	sort.Strings(PanelNames)

	html := `
	<html>

	<head>
		<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
		<script>
	
			const config = {
				type: 'scatter',
				data: {
					datasets: JSON.parse('` + string(jsonBytes) + `')
				},
				options: {
					responsive: true,
					showLine: true,
					scales: {
						x: {
							display: true,
							title: {
								display: true,
								text: 'Time, s'
							}
						},
						y: {
							type: 'linear',
							display: true,
							title: {
								display: true,
								text: 'Temp'
							},
							position: 'left',
							min:40,
							max:90
						},
						yVal: {
							type: 'linear',
							display: true,
							position: 'right',
							title: {
								display: true,
								text: 'Load'
							},
							min:0,
							max:100
						}
					}
				}
			};
		</script>
		<style>
			body {
				font-family: Arial, Helvetica, sans-serif;
				font-size: 9px;
			}
	
			table {
				font-size: 9px;
			}
	
			table.details tr td {
				padding-left: 3px;
				padding-right: 3px;
				background-color: #cccccc;
			}
		</style>
	</head>
  <body>
		<h1>CPU Temperature and Load over time</h1>
		` + time.Now().Format(time.ANSIC) + `<br/> 
		
		<b>Panels:</b><br/>
		` + strings.Join(PanelNames, "<br/>") + `

		<div>
			<canvas id="myChart"></canvas>
		</div>

		<h2>CSV data:</h2>
		<pre>` + csv + `</pre>
		<script>
			const myChart = new Chart(
				document.getElementById('myChart'),
				config
			);
		</script>
  </body>
</html>
	`
	os.WriteFile(filepath.Join(getOutputPath(), "cpu"+matchIdString+".html"), []byte(html), 0644)
}

// Profiling analog data:
type AnalogDataPoint struct {
	timestamp   uint32
	sampleWidth uint32
	min         int
	max         int
	average     int
}
type AnalogEventData struct {
	timestamp uint32
	value     int
}

var StartTimestamp = make(map[int]uint32)
var RoundRobinSize = 100
var DataPoints = make(map[int]map[uint32][]AnalogDataPoint)
var AnalogRoundRobin = make(map[int]map[uint32][]AnalogEventData)

func procesRawAnalogValue(panelNum int, Event *rwp.HWCEvent) {
	DataPointsMU.Lock()
	defer DataPointsMU.Unlock()

	if _, exists := StartTimestamp[panelNum]; !exists {
		StartTimestamp[panelNum] = Event.Timestamp
	}

	// Check and create maps:
	if _, exists := DataPoints[panelNum]; !exists {
		DataPoints[panelNum] = make(map[uint32][]AnalogDataPoint)
	}
	if _, exists := AnalogRoundRobin[panelNum]; !exists {
		AnalogRoundRobin[panelNum] = make(map[uint32][]AnalogEventData)
	}

	// Check and create maps:
	if _, exists := DataPoints[panelNum][Event.HWCID]; !exists {
		DataPoints[panelNum][Event.HWCID] = []AnalogDataPoint{}
	}
	if _, exists := AnalogRoundRobin[panelNum][Event.HWCID]; !exists {
		AnalogRoundRobin[panelNum][Event.HWCID] = []AnalogEventData{}
	}

	// Add to round robin or elevate to datapoint:
	if len(AnalogRoundRobin[panelNum][Event.HWCID]) < RoundRobinSize {
		AnalogRoundRobin[panelNum][Event.HWCID] = append(AnalogRoundRobin[panelNum][Event.HWCID], AnalogEventData{
			value:     int(Event.RawAnalog.Value),
			timestamp: Event.Timestamp,
		})
	} else {
		aDP := AnalogDataPoint{min: 10000, max: -10000, sampleWidth: 0xFFFFFFFF}
		cnt := 0
		for _, aED := range AnalogRoundRobin[panelNum][Event.HWCID] {
			if aED.value < aDP.min {
				aDP.min = aED.value
			}
			if aED.value > aDP.max {
				aDP.max = aED.value
			}
			aDP.average += aED.value // Summing for later average calculation

			if aED.timestamp > aDP.timestamp {
				aDP.timestamp = aED.timestamp
			}
			if aED.timestamp < aDP.sampleWidth {
				aDP.sampleWidth = aED.timestamp // Collecting lowest value for later calculation
			}
			cnt++
		}
		aDP.average = int(math.Round(float64(aDP.average) / float64(cnt)))
		aDP.sampleWidth = aDP.timestamp - aDP.sampleWidth
		aDP.timestamp = aDP.timestamp - StartTimestamp[panelNum]
		//fmt.Println(aDP)

		DataPoints[panelNum][Event.HWCID] = append(DataPoints[panelNum][Event.HWCID], aDP)
		AnalogRoundRobin[panelNum][Event.HWCID] = []AnalogEventData{}
	}
}

var CSVLines = []string{""}
var CSVfieldsOrder = []string{}

func writeAnalogStatus() {

	// Add any new panel/HWC pairs if found:
	for panelNum, panelDPs := range DataPoints {
		for HWCID, _ := range panelDPs {
			panelHWC := fmt.Sprintf("_p%d.%d", panelNum, HWCID)
			if !slices.Contains(CSVfieldsOrder, panelHWC) {
				CSVfieldsOrder = append(CSVfieldsOrder, panelHWC)
			}
		}
	}

	thisLine := []string{}
	header := []string{}
	for _, panelHWC := range CSVfieldsOrder {
		split := strings.Split(panelHWC[2:]+".", ".")
		panelNum, _ := strconv.Atoi(split[0])
		HWCID, _ := strconv.Atoi(split[1])
		dpLen := len(DataPoints[panelNum][uint32(HWCID)])
		lastDP := DataPoints[panelNum][uint32(HWCID)][dpLen-1]
		thisLine = append(thisLine, fmt.Sprintf("%d,%d,%d", lastDP.timestamp, lastDP.average, lastDP.max-lastDP.min))
		header = append(header, fmt.Sprintf("%s time, %s avg value, %s delta", panelHWC, panelHWC, panelHWC))
	}
	CSVLines[0] = strings.Join(header, ",")
	lineAsString := strings.Join(thisLine, ",")
	CSVLines = append(CSVLines, lineAsString)

	os.WriteFile(filepath.Join(getOutputPath(), "allAnalog.csv"), []byte(strings.Join(CSVLines, "\n")), 0644)

	generateHTML(strings.Join(CSVLines, "\n"), "")
	for panelNum, panelDPs := range DataPoints {
		for HWCID, _ := range panelDPs {
			idString := fmt.Sprintf("_p%d.%d", panelNum, HWCID)
			generateHTML("", idString)
		}
	}
}

var OutputPath = ""

func getOutputPath() string {

	if OutputPath == "" {
		OutputPath = filepath.Join("Output", time.Now().Format(time.ANSIC))
		//OutputPath = filepath.Join("Output")
		err := os.MkdirAll(OutputPath, 0755)
		log.Should(err)
	}
	return OutputPath
}

// Write HTML

type ttpoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type ttDataSet struct {
	Label       string     `json:"label,omitempty"`
	Data        []*ttpoint `json:"data,omitempty"`
	PointRadius []int      `json:"pointRadius,omitempty"`
	BorderColor string     `json:"borderColor,omitempty"`
	BorderDash  []int      `json:"borderDash,omitempty"`
	BorderWidth float64    `json:"borderWidth,omitempty"`
	YaxisId     string     `json:"yAxisID,omitempty"`
}

func generateHTML(csv string, matchIdString string) {

	series := make(map[string]*ttDataSet)
	colorsOptions := []string{"rgb(238,65,37)", "rgb(0,0,0)", "rgb(206,171,55)", "rgb(21,50,245)", "rgb(238,87,247)", "rgb(52,127,248)", "rgb(148,117,120)", "rgb(109,248,253)", "rgb(108,246,138)", "rgb(169,173,249)", "rgb(159,66,246)", "rgb(240,131,49)", "rgb(191,191,191)", "rgb(220,249,80)"}
	colorsOptionsPointer := 0

	// Add any new panel/HWC pairs if found:
	for panelNum, panelDPs := range DataPoints {
		for HWCID, aDPs := range panelDPs {
			idString := fmt.Sprintf("_p%d.%d", panelNum, HWCID)

			if matchIdString == "" || matchIdString == idString {
				// Set up new data series if found:
				borderWidth := 1.5
				series[idString+"D"] = &ttDataSet{
					Label:       idString + " Diff",
					BorderColor: colorsOptions[colorsOptionsPointer%len(colorsOptions)],
					BorderWidth: borderWidth,
				}
				series[idString+"V"] = &ttDataSet{
					Label:       idString + " Value",
					BorderColor: colorsOptions[colorsOptionsPointer%len(colorsOptions)],
					BorderWidth: borderWidth,
					BorderDash:  []int{6, 2},
					PointRadius: []int{0, 0},
				}

				for _, aDP := range aDPs {
					series[idString+"D"].Data = append(series[idString+"D"].Data, &ttpoint{X: float64(int(aDP.timestamp / 1000)), Y: float64(int(aDP.max - aDP.min))})
					series[idString+"V"].Data = append(series[idString+"V"].Data, &ttpoint{X: float64(int(aDP.timestamp / 1000)), Y: float64(int(aDP.average))})
					series[idString+"V"].YaxisId = "yVal"
				}

				colorsOptionsPointer++
			}
		}
	}

	ordered := []string{}
	for ttDSkey := range series {
		ordered = append(ordered, ttDSkey)
	}
	sort.Strings(ordered)

	outputSeries := []*ttDataSet{}
	for _, ttDS := range ordered {
		outputSeries = append(outputSeries, series[ttDS])
	}
	jsonBytes, _ := json.Marshal(outputSeries)

	PanelNames := []string{}
	for _, n := range PanelName {
		PanelNames = append(PanelNames, n)
	}
	sort.Strings(PanelNames)

	html := `
	<html>

	<head>
		<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
		<script>
	
			const config = {
				type: 'scatter',
				data: {
					datasets: JSON.parse('` + string(jsonBytes) + `')
				},
				options: {
					responsive: true,
					showLine: true,
					scales: {
						x: {
							display: true,
							title: {
								display: true,
								text: 'Time, s'
							}
						},
						y: {
							type: 'linear',
							display: true,
							title: {
								display: true,
								text: 'Diff'
							},
							position: 'left',
							min: 0
						},
						yVal: {
							type: 'linear',
							display: true,
							position: 'right',
							title: {
								display: true,
								text: 'Value'
							},
							min: 0,
							max: 4196
						}
					}
				}
			};
		</script>
		<style>
			body {
				font-family: Arial, Helvetica, sans-serif;
				font-size: 9px;
			}
	
			table {
				font-size: 9px;
			}
	
			table.details tr td {
				padding-left: 3px;
				padding-right: 3px;
				background-color: #cccccc;
			}
		</style>
	</head>
  <body>
		<h1>Raw Analog Values over time</h1>
		` + time.Now().Format(time.ANSIC) + `<br/> 
		
		<b>Panels:</b><br/>
		` + strings.Join(PanelNames, "<br/>") + `

		<div>
			<canvas id="myChart"></canvas>
		</div>

		<h2>CSV data:</h2>
		<pre>` + csv + `</pre>
		<script>
			const myChart = new Chart(
				document.getElementById('myChart'),
				config
			);
		</script>
  </body>
</html>
	`
	os.WriteFile(filepath.Join(getOutputPath(), "analog"+matchIdString+".html"), []byte(html), 0644)
}
