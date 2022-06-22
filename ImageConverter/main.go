/*
	Image file converter for Raw Panel protocol
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"

	_ "image/gif"  // Allow gifs to be loaded
	_ "image/jpeg" // Allow jpegs to be loaded
	_ "image/png"  // Allow pngs to be loaded
)

func main() {

	// Setting up and parsing command line parameters
	wxh := flag.String("WxH", "", "The width x height of the encoded image. Default is to use the same dimensions as the input image.")
	scaling := flag.String("scaling", "", "Specifies how the input image will be fitted into the dimensions of WxH: 'Fit' will scale the image so everything is shown but with pillar/letter box, 'Fill' will scale the image so all pixels are filled but the image may be cropped, 'Stretch' will scale the image so all pixels as filled but image may be distorted in dimensions. Default is to show pixels 1:1")
	encoding := flag.String("encoding", "", "Specifies if the image should be encoder as Grayscale ('Gray') or RGB image ('RGB')")
	pureJsonOutput := flag.Bool("pureJSON", false, "If set, the output is pure JSON and not additional comments or explanations. (Useful for automated conversions).")
	hwcNumber := flag.Int("HWC", 99999, "HWC number to use in state message")
	flag.Parse()

	arguments := flag.Args()
	if len(arguments) == 0 {
		fmt.Println("usage: ImageConverter [options] [image filename]")
		fmt.Println("help:  ImageConverter -h")
		fmt.Println("")
		return
	}

	// Welcome message!
	if !*pureJsonOutput {
		fmt.Println("Image Converter to Raw Panel! Made by Kasper Skaarhoj (c) 2022")
	}

	file, err := os.ReadFile(arguments[0])
	log.Must(err)

	// Custom WxH:
	width := 0
	height := 0
	if *wxh != "" {
		parts := strings.Split((*wxh)+"x", "x")
		width, _ = strconv.Atoi(parts[0])
		height, _ = strconv.Atoi(parts[1])
		if width < 0 || width > 500 || height < 0 || height > 500 {
			width = 0
			height = 0
		}
	}

	// Specific scaling
	scalingValue := rwp.HWCGfxConverter_ScalingE(0)
	switch *scaling {
	case "Fit":
		scalingValue = rwp.HWCGfxConverter_FIT
	case "Fill":
		scalingValue = rwp.HWCGfxConverter_FILL
	case "Stretch":
		scalingValue = rwp.HWCGfxConverter_STRETCH
	}

	// Specific encoding
	encodingValue := rwp.HWCGfxConverter_ImageTypeE(0)
	switch *encoding {
	case "Gray":
		encodingValue = rwp.HWCGfxConverter_Gray4bit
	case "RGB":
		encodingValue = rwp.HWCGfxConverter_RGB16bit
	}

	state := &rwp.HWCState{
		HWCIDs: []uint32{uint32(*hwcNumber)},
		HWCGfxConverter: &rwp.HWCGfxConverter{
			W:         uint32(width),
			H:         uint32(height),
			ImageType: encodingValue,
			Scaling:   scalingValue,
			ImageData: file,
		},
	}

	// log.Println(log.Indent(state))
	helpers.StateConverter(state)

	// ASCII lines
	asciiLines := helpers.InboundMessagesToRawPanelASCIIstrings([]*rwp.InboundMessage{&rwp.InboundMessage{States: []*rwp.HWCState{state}}})

	// JSON encoded
	jsonBytes, _ := json.Marshal(state)

	if !*pureJsonOutput {
		fmt.Println("")
		fmt.Println("Image encoded as Raw Panel ASCII images over multiple messages:")
		fmt.Println("---------------------------------------------------------------------------------------")
		for _, line := range asciiLines {
			fmt.Println(line)
		}
		fmt.Println("---------------------------------------------------------------------------------------")
		fmt.Println("")

		fmt.Printf("Image encoded as a single ASCII transmissible JSON message (%d bytes):\n", len(jsonBytes))
		fmt.Println("---------------------------------------------------------------------------------------")
	}

	fmt.Println(string(jsonBytes))
}
