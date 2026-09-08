package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"regexp"
	"strings"
)

var backgroundPattern = regexp.MustCompile(`background-image:\s*url\('data:image/png;base64,[^']+'\);`)

func main() {
	if len(os.Args) != 4 && len(os.Args) != 5 {
		fail("usage: go run scripts/build-cftp-template.go <cfta-template> <rendered-cftp-png> <output> [preview-output]")
	}

	cftaTemplate, err := os.ReadFile(os.Args[1])
	if err != nil {
		fail("read CFTA template: %v", err)
	}

	backgroundFile, err := os.Open(os.Args[2])
	if err != nil {
		fail("open CFtP background: %v", err)
	}
	defer backgroundFile.Close()

	decoded, err := png.Decode(backgroundFile)
	if err != nil {
		fail("decode CFtP background: %v", err)
	}

	background := image.NewNRGBA(decoded.Bounds())
	draw.Draw(background, background.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	blankDynamicText(background)

	var encodedBackground bytes.Buffer
	if err := png.Encode(&encodedBackground, background); err != nil {
		fail("encode CFtP background: %v", err)
	}

	html := string(cftaTemplate)
	replacement := "background-image: url('data:image/png;base64," + base64.StdEncoding.EncodeToString(encodedBackground.Bytes()) + "');"
	html = backgroundPattern.ReplaceAllString(html, replacement)
	if !strings.Contains(html, replacement) {
		fail("CFTA template background declaration was not found")
	}
	html = strings.Replace(html, "Certificate of Completion — Global Fintech Institute", "CFtP Charter - Global Fintech Institute", 1)
	html = strings.Replace(html, "aria-label=\"Certificate of Completion\"", "aria-label=\"Chartered Fintech Professional certificate\"", 1)
	parsed, err := template.New("cftp-certificate").Option("missingkey=error").Parse(html)
	if err != nil {
		fail("parse generated Go template: %v", err)
	}
	var preview bytes.Buffer
	if err := parsed.Execute(&preview, map[string]string{
		"Name":        "Andrzej Gwizdalski",
		"CandidateNo": "1000111J",
	}); err != nil {
		fail("render generated Go template: %v", err)
	}

	if err := os.WriteFile(os.Args[3], []byte(html), 0o644); err != nil {
		fail("write CFtP template: %v", err)
	}
	if len(os.Args) == 5 {
		if err := os.WriteFile(os.Args[4], preview.Bytes(), 0o644); err != nil {
			fail("write CFtP preview: %v", err)
		}
	}

	fmt.Printf("generated %s\n", os.Args[3])
}

func blankDynamicText(img *image.NRGBA) {
	bounds := img.Bounds()
	blankWithVerticalInterpolation(img, proportionalRect(bounds, 0.35, 0.428, 0.65, 0.489))
	blankWithVerticalInterpolation(img, proportionalRect(bounds, 0.51, 0.502, 0.62, 0.542))
}

func proportionalRect(bounds image.Rectangle, left, top, right, bottom float64) image.Rectangle {
	width := float64(bounds.Dx())
	height := float64(bounds.Dy())
	return image.Rect(
		bounds.Min.X+int(width*left),
		bounds.Min.Y+int(height*top),
		bounds.Min.X+int(width*right),
		bounds.Min.Y+int(height*bottom),
	).Intersect(bounds)
}

func blankWithVerticalInterpolation(img *image.NRGBA, area image.Rectangle) {
	topY := max(area.Min.Y-2, img.Bounds().Min.Y)
	bottomY := min(area.Max.Y+2, img.Bounds().Max.Y-1)
	span := float64(area.Dy() + 1)

	for x := area.Min.X; x < area.Max.X; x++ {
		top := color.NRGBAModel.Convert(img.At(x, topY)).(color.NRGBA)
		bottom := color.NRGBAModel.Convert(img.At(x, bottomY)).(color.NRGBA)
		for y := area.Min.Y; y < area.Max.Y; y++ {
			ratio := float64(y-area.Min.Y+1) / span
			img.SetNRGBA(x, y, color.NRGBA{
				R: blend(top.R, bottom.R, ratio),
				G: blend(top.G, bottom.G, ratio),
				B: blend(top.B, bottom.B, ratio),
				A: blend(top.A, bottom.A, ratio),
			})
		}
	}
}

func blend(start, end uint8, ratio float64) uint8 {
	return uint8(float64(start)*(1-ratio) + float64(end)*ratio)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
