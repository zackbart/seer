package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

func renderImageASCII(img image.Image, width, height int) string {
	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return ""
	}

	outW := max(16, width-2)
	outH := max(8, height-3)

	if supportsTrueColor() {
		return renderImageTrueColor(img, outW, outH)
	}
	return renderImageGray(img, outW, outH)
}

func rgbValues(c color.Color) (int, int, int) {
	r, g, b, _ := c.RGBA()
	return int(r >> 8), int(g >> 8), int(b >> 8)
}

func renderImageTrueColor(img image.Image, outW, outH int) string {
	b := img.Bounds()
	scaledH := outH * 2

	var sb strings.Builder
	for row := 0; row < outH; row++ {
		upperY := b.Min.Y + ((row*2)*(b.Dy()-1))/max(1, scaledH-1)
		lowerY := b.Min.Y + ((row*2+1)*(b.Dy()-1))/max(1, scaledH-1)

		lastFgR, lastFgG, lastFgB := -1, -1, -1
		lastBgR, lastBgG, lastBgB := -1, -1, -1

		for x := 0; x < outW; x++ {
			sx := b.Min.X + (x*(b.Dx()-1))/max(1, outW-1)
			fgR, fgG, fgB := rgbValues(img.At(sx, upperY))
			bgR, bgG, bgB := rgbValues(img.At(sx, lowerY))

			if fgR != lastFgR || fgG != lastFgG || fgB != lastFgB || bgR != lastBgR || bgG != lastBgG || bgB != lastBgB {
				writeTrueColorANSI(&sb, fgR, fgG, fgB, bgR, bgG, bgB)
				lastFgR, lastFgG, lastFgB = fgR, fgG, fgB
				lastBgR, lastBgG, lastBgB = bgR, bgG, bgB
			}
			sb.WriteRune('▀')
		}

		sb.WriteString("\x1b[0m")
		if row < outH-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func renderImageGray(img image.Image, outW, outH int) string {
	b := img.Bounds()
	chars := []rune(" .:-=+*#%@")

	var sb strings.Builder
	for y := 0; y < outH; y++ {
		sy := b.Min.Y + (y*(b.Dy()-1))/max(1, outH-1)
		for x := 0; x < outW; x++ {
			sx := b.Min.X + (x*(b.Dx()-1))/max(1, outW-1)
			lum := luminance(img.At(sx, sy))
			idx := int(lum * float64(len(chars)-1) / 255.0)
			if idx < 0 {
				idx = 0
			}
			if idx >= len(chars) {
				idx = len(chars) - 1
			}
			sb.WriteRune(chars[idx])
		}
		if y < outH-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func writeTrueColorANSI(sb *strings.Builder, fgR, fgG, fgB, bgR, bgG, bgB int) {
	fmt.Fprintf(sb, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm", fgR, fgG, fgB, bgR, bgG, bgB)
}

func supportsTrueColor() bool {
	if _, noColor := os.LookupEnv("NO_COLOR"); noColor {
		return false
	}
	colorTerm := strings.ToLower(os.Getenv("COLORTERM"))
	if strings.Contains(colorTerm, "truecolor") || strings.Contains(colorTerm, "24bit") {
		return true
	}
	term := strings.ToLower(os.Getenv("TERM"))
	if strings.Contains(term, "kitty") || strings.Contains(term, "wezterm") {
		return true
	}
	return false
}

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	rf := float64(r>>8) * 0.299
	gf := float64(g>>8) * 0.587
	bf := float64(b>>8) * 0.114
	return rf + gf + bf
}
