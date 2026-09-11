package media

import (
	"bytes"
	"encoding/binary"
	"image/gif"
	"image/jpeg"
	"image/png"
	"regexp"
	"strconv"
	"strings"
)

// DetectMIME sniffs the leading bytes and returns the supported MIME type.
// The second result is false for any type outside the whitelist.
func DetectMIME(data []byte) (string, bool) {
	switch {
	case len(data) >= 8 && bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n")):
		return MIMEPNG, true
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return MIMEJPEG, true
	case len(data) >= 6 && (bytes.Equal(data[:6], []byte("GIF87a")) || bytes.Equal(data[:6], []byte("GIF89a"))):
		return MIMEGIF, true
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return MIMEWEBP, true
	case looksLikeSVG(data):
		return MIMESVG, true
	default:
		return "", false
	}
}

// Extension returns the canonical file extension (with dot) for a MIME type.
func Extension(mimeType string) string {
	switch mimeType {
	case MIMEPNG:
		return ".png"
	case MIMEJPEG:
		return ".jpg"
	case MIMEGIF:
		return ".gif"
	case MIMEWEBP:
		return ".webp"
	case MIMESVG:
		return ".svg"
	default:
		return ""
	}
}

// Dimensions returns the media's intrinsic width and height, or zeros when the
// format or header cannot be parsed. Dimensions are cosmetic metadata, so a
// parse failure never rejects an otherwise valid upload.
func Dimensions(mimeType string, data []byte) (int, int) {
	switch mimeType {
	case MIMEPNG:
		if cfg, err := png.DecodeConfig(bytes.NewReader(data)); err == nil {
			return cfg.Width, cfg.Height
		}
	case MIMEJPEG:
		if cfg, err := jpeg.DecodeConfig(bytes.NewReader(data)); err == nil {
			return cfg.Width, cfg.Height
		}
	case MIMEGIF:
		if cfg, err := gif.DecodeConfig(bytes.NewReader(data)); err == nil {
			return cfg.Width, cfg.Height
		}
	case MIMEWEBP:
		return webpDimensions(data)
	case MIMESVG:
		return svgDimensions(data)
	}
	return 0, 0
}

// looksLikeSVG reports whether the document's root element is <svg>, tolerating
// a UTF-8 BOM, whitespace, an XML declaration, DOCTYPE, and comments.
func looksLikeSVG(data []byte) bool {
	sample := data
	if len(sample) > 2048 {
		sample = sample[:2048]
	}
	text := strings.ToLower(string(sample))
	text = strings.TrimPrefix(text, "\ufeff")
	for {
		text = strings.TrimLeft(text, " \t\r\n")
		switch {
		case strings.HasPrefix(text, "<?xml"):
			if end := strings.Index(text, "?>"); end >= 0 {
				text = text[end+2:]
				continue
			}
			return false
		case strings.HasPrefix(text, "<!--"):
			if end := strings.Index(text, "-->"); end >= 0 {
				text = text[end+3:]
				continue
			}
			return false
		case strings.HasPrefix(text, "<!doctype"):
			if end := strings.Index(text, ">"); end >= 0 {
				text = text[end+1:]
				continue
			}
			return false
		}
		break
	}
	return strings.HasPrefix(text, "<svg")
}

var svgNumberPattern = regexp.MustCompile(`[0-9]*\.?[0-9]+`)

// svgDimensions best-effort extracts width/height from an SVG root element,
// falling back to the viewBox. Unit suffixes (px, pt, %) and relative values
// are ignored; failures return zeros.
func svgDimensions(data []byte) (int, int) {
	sample := data
	if len(sample) > 8192 {
		sample = sample[:8192]
	}
	text := strings.ToLower(string(sample))
	start := strings.Index(text, "<svg")
	if start < 0 {
		return 0, 0
	}
	root := text[start:]
	if end := strings.Index(root, ">"); end >= 0 {
		root = root[:end]
	}
	if w, ok := attrDimension(root, "width"); ok {
		if h, ok := attrDimension(root, "height"); ok {
			return w, h
		}
	}
	if viewBox := attrValue(root, "viewbox"); viewBox != "" {
		fields := strings.FieldsFunc(viewBox, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' })
		if len(fields) == 4 {
			w := parseDimension(fields[2])
			h := parseDimension(fields[3])
			return w, h
		}
	}
	return 0, 0
}

func attrDimension(root, name string) (int, bool) {
	raw := attrValue(root, name)
	if raw == "" {
		return 0, false
	}
	value := parseDimension(raw)
	if value <= 0 {
		return 0, false
	}
	return value, true
}

// attrValue extracts a (case-insensitive) attribute value from an element tag.
func attrValue(tag, name string) string {
	lower := strings.ToLower(tag)
	idx := 0
	for {
		pos := strings.Index(lower[idx:], name)
		if pos < 0 {
			return ""
		}
		pos += idx
		// Require a tag-name boundary before the attribute name.
		if pos > 0 {
			prev := lower[pos-1]
			if prev != ' ' && prev != '\t' && prev != '\n' && prev != '\r' {
				idx = pos + len(name)
				continue
			}
		}
		after := lower[pos+len(name):]
		trimmed := strings.TrimLeft(after, " \t\r\n")
		if !strings.HasPrefix(trimmed, "=") {
			idx = pos + len(name)
			continue
		}
		rest := strings.TrimLeft(trimmed[1:], " \t\r\n")
		if len(rest) < 2 {
			return ""
		}
		quote := rest[0]
		if quote != '"' && quote != '\'' {
			return ""
		}
		end := strings.IndexByte(rest[1:], quote)
		if end < 0 {
			return ""
		}
		return rest[1 : 1+end]
	}
}

func parseDimension(raw string) int {
	match := svgNumberPattern.FindString(strings.TrimSpace(raw))
	if match == "" {
		return 0
	}
	value, err := strconv.ParseFloat(match, 64)
	if err != nil || value <= 0 {
		return 0
	}
	return int(value)
}

// webpDimensions parses the canvas size from a RIFF/WEBP container for the
// VP8 (lossy), VP8L (lossless), and VP8X (extended) chunk layouts.
func webpDimensions(data []byte) (int, int) {
	if len(data) < 20 {
		return 0, 0
	}
	switch string(data[12:16]) {
	case "VP8 ":
		// Frame tag (3 bytes) + start code (3 bytes) + 14-bit width/height.
		if len(data) < 30 || !bytes.Equal(data[23:26], []byte{0x9d, 0x01, 0x2a}) {
			return 0, 0
		}
		v := binary.LittleEndian.Uint16(data[26:28])
		return int(v & 0x3fff), int((v >> 14) & 0x3fff)
	case "VP8L":
		if len(data) < 25 || data[20] != 0x2f {
			return 0, 0
		}
		v := binary.LittleEndian.Uint32(data[21:25])
		return int(v&0x3fff) + 1, int((v>>14)&0x3fff) + 1
	case "VP8X":
		if len(data) < 30 {
			return 0, 0
		}
		width := readUint24LE(data[24:27]) + 1
		height := readUint24LE(data[27:30]) + 1
		return width, height
	default:
		return 0, 0
	}
}

func readUint24LE(b []byte) int {
	return int(b[0]) | int(b[1])<<8 | int(b[2])<<16
}
