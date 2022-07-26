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
	encoding := flag.String("encoding", "", "Specifies if the image should be encoded as Grayscale ('Gray') or RGB image ('RGB')")
	output := flag.String("output", "", "Default is ASCII Raw Panel output. Use value 'JSON' for JSON output, 'C' for C output.")
	writePng := flag.Bool("png", false, "If set, the output is written back to a PNG file named 'ImageConverter_output.png' for validation purposes.")
	invert := flag.Bool("invert", false, "Inverts Monochrome images")
	hwcNumber := flag.Int("HWC", 99999, "HWC number to use in state message")
	flag.Parse()

	pureJsonOutput := *output == "JSON"

	arguments := flag.Args()
	if len(arguments) == 0 {
		fmt.Println("usage: ImageConverter [options] [image filename]")
		fmt.Println("help:  ImageConverter -h")
		fmt.Println("")
		return
	}

	// Welcome message!
	if !pureJsonOutput {
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

	// PNG:
	if *writePng {
		pngBytes, err := helpers.ConvertGfxStateToPngBytes(state)
		log.Must(err)
		err = os.WriteFile("ImageConverter_output.png", pngBytes, 0644)
		log.Must(err)
	}

	switch *output {
	case "JSON":
		fmt.Println(string(jsonBytes))
	case "C":
		fmt.Println("")
		fmt.Println("Image encoded for inclusion in C++ code:")
		fmt.Println("---------------------------------------------------------------------------------------")

		bytesPerLine := len(state.HWCGfx.ImageData) / int(state.HWCGfx.H)
		fmt.Println("TEST:", bytesPerLine, len(state.HWCGfx.ImageData)%int(state.HWCGfx.H))
		fmt.Printf("{ // Image %s, dimensions %dx%d", arguments[0], int(state.HWCGfx.W), int(state.HWCGfx.H))
		for i, byteValue := range state.HWCGfx.ImageData {
			if i%bytesPerLine == 0 {
				fmt.Print("\n\t")
			}

			switch state.HWCGfx.ImageType {
			case rwp.HWCGfx_MONO:
				if *invert {
					byteValue = byteValue ^ 0xff
				}
				fmt.Printf("0b%d%d%d%d%d%d%d%d, ", (byteValue>>7)&1, (byteValue>>6)&1, (byteValue>>5)&1, (byteValue>>4)&1, (byteValue>>3)&1, (byteValue>>2)&1, (byteValue>>1)&1, (byteValue>>0)&1)
			default:
				fmt.Printf("0x%x%x, ", byteValue>>4, byteValue&0xF)
			}
		}
		fmt.Println("\n---------------------------------------------------------------------------------------")
	default:
		fmt.Println("")
		fmt.Println("Image encoded as Raw Panel ASCII images over multiple messages:")
		fmt.Println("---------------------------------------------------------------------------------------")
		for _, line := range asciiLines {
			fmt.Println(line)
		}
		fmt.Println("---------------------------------------------------------------------------------------")
	}

}
