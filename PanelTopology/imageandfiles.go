package main

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

	log "github.com/s00500/env_logger"

	"github.com/fogleman/gg"

	"github.com/golang/freetype/truetype"

	"golang.org/x/image/font"
)

//go:embed resources
var embeddedFS embed.FS

// Read contents from ordinary or embedded file
func ReadResourceFile(fileName string) []byte {
	fileName = strings.ReplaceAll(fileName, "\\", "/")
	byteValue, err := embeddedFS.ReadFile(fileName)
	log.Should(err)
	return byteValue
}

func createTestImage(W int, H int, imagetype string) ([]byte, error) {

	canvas := image.NewRGBA(image.Rect(0, 0, W, H))
	dc := gg.NewContextForRGBA(canvas)

	grad := gg.NewLinearGradient(0, float64(H), float64(W), 0)
	if imagetype == "color" {
		grad.AddColorStop(0, color.RGBA{0, 255, 0, 255})
		grad.AddColorStop(1, color.RGBA{0, 0, 255, 255})
		grad.AddColorStop(0.5, color.RGBA{255, 0, 0, 255})
		dc.SetFillStyle(grad)
	} else if imagetype == "gray" {
		grad.AddColorStop(0, color.RGBA{0, 0, 0, 255})
		grad.AddColorStop(1, color.RGBA{255, 255, 255, 255})
		dc.SetFillStyle(grad)
	} else {
		dc.SetColor(color.Black)
	}

	dc.MoveTo(0, 0)
	dc.LineTo(float64(W), 0)
	dc.LineTo(float64(W), float64(H))
	dc.LineTo(0, float64(H))
	dc.ClosePath()
	dc.Fill()

	dc.SetColor(color.White)
	dc.DrawRectangle(0, 0, float64(W), float64(H))
	dc.Stroke()

	face, err := LoadFontFace("resources/NotoSans-Regular.ttf", 20)
	if !log.Should(err) {
		dc.SetFontFace(face)
		dc.DrawStringAnchored(fmt.Sprintf("%dx%d", W, H), float64(W)/2, float64(H)/2, 0.5, 0.5)
	}

	buf := new(bytes.Buffer)
	err = png.Encode(buf, canvas)
	return buf.Bytes(), err
}

// LoadFontFace is a helper function to load the specified font file with
// the specified point size. Note that the returned `font.Face` objects
// are not thread safe and cannot be used in parallel across goroutines.
// You can usually just use the Context.LoadFontFace function instead of
// this package-level function.
func LoadFontFace(path string, points float64) (font.Face, error) {
	fontBytes, err := embeddedFS.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		return nil, err
	}
	face := truetype.NewFace(f, &truetype.Options{
		Size: points,
		// Hinting: font.HintingFull,
	})
	return face, nil
}
