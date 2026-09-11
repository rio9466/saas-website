package media

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

func testImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	return img
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, testImage()); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func jpegBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, testImage(), nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func gifBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := gif.Encode(&buf, testImage(), nil); err != nil {
		t.Fatalf("encode gif: %v", err)
	}
	return buf.Bytes()
}

func TestDetectMIMEAndDimensions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		data       []byte
		wantMIME   string
		wantWidth  int
		wantHeight int
	}{
		{"png", pngBytes(t), MIMEPNG, 3, 2},
		{"jpeg", jpegBytes(t), MIMEJPEG, 3, 2},
		{"gif", gifBytes(t), MIMEGIF, 3, 2},
		{"webp vp8x", webpVP8X(10, 5), MIMEWEBP, 10, 5},
		{"svg", []byte(`<?xml version="1.0"?><svg width="40" height="20" xmlns="http://www.w3.org/2000/svg"></svg>`), MIMESVG, 40, 20},
		{"svg viewBox", []byte(`<svg viewBox="0 0 64 32"></svg>`), MIMESVG, 64, 32},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gotMIME, ok := DetectMIME(tc.data)
			if !ok || gotMIME != tc.wantMIME {
				t.Fatalf("DetectMIME() = %q, %v; want %q, true", gotMIME, ok, tc.wantMIME)
			}
			w, h := Dimensions(gotMIME, tc.data)
			if w != tc.wantWidth || h != tc.wantHeight {
				t.Fatalf("Dimensions() = %d x %d; want %d x %d", w, h, tc.wantWidth, tc.wantHeight)
			}
		})
	}
}

func TestDetectMIMERejectsUnsupported(t *testing.T) {
	t.Parallel()
	for _, data := range [][]byte{
		nil,
		[]byte("hello world"),
		[]byte("<html><body>hi</body></html>"),
		[]byte("MZ\x90\x00"),
	} {
		if mimeType, ok := DetectMIME(data); ok {
			t.Fatalf("DetectMIME(%q) = %q, true; want false", data, mimeType)
		}
	}
}

func TestExtension(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		MIMEPNG:      ".png",
		MIMEJPEG:     ".jpg",
		MIMEGIF:      ".gif",
		MIMEWEBP:     ".webp",
		MIMESVG:      ".svg",
		"text/plain": "",
	}
	for mimeType, want := range cases {
		if got := Extension(mimeType); got != want {
			t.Fatalf("Extension(%q) = %q, want %q", mimeType, got, want)
		}
	}
}

// webpVP8X builds a minimal extended-format WEBP container header.
func webpVP8X(width, height int) []byte {
	buf := make([]byte, 30)
	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], uint32(len(buf)-8))
	copy(buf[8:12], "WEBP")
	copy(buf[12:16], "VP8X")
	binary.LittleEndian.PutUint32(buf[16:20], 10)
	// payload: 1 flags byte + 3 reserved bytes, then (width-1) and (height-1)
	// as 24-bit little-endian values.
	putUint24LE(buf[24:27], width-1)
	putUint24LE(buf[27:30], height-1)
	return buf
}

func putUint24LE(b []byte, v int) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
}
