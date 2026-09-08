// Tests for email-service tracking package.
package tracking

import (
	"encoding/base64"
	"testing"
)

func TestPixelGif_Length(t *testing.T) {
	if len(PixelGif) != 43 {
		t.Errorf("expected 43-byte GIF, got %d", len(PixelGif))
	}
}

func TestPixelGif_Header(t *testing.T) {
	// GIF89a magic
	if string(PixelGif[0:6]) != "GIF89a" {
		t.Errorf("expected GIF89a header, got %s", string(PixelGif[0:6]))
	}
}

func TestPixelGif_ImageSize(t *testing.T) {
	// Width is at bytes 6-7 (little-endian), Height at 8-9
	width := int(PixelGif[6]) | int(PixelGif[7])<<8
	height := int(PixelGif[8]) | int(PixelGif[9])<<8
	if width != 1 {
		t.Errorf("expected width=1, got %d", width)
	}
	if height != 1 {
		t.Errorf("expected height=1, got %d", height)
	}
}

func TestPixelGif_Trailer(t *testing.T) {
	// GIF ends with 0x3B (semicolon)
	if PixelGif[len(PixelGif)-1] != 0x3B {
		t.Errorf("expected trailer 0x3B, got %x", PixelGif[len(PixelGif)-1])
	}
}

func TestPixelBase64_RoundTrip(t *testing.T) {
	decoded, err := base64.StdEncoding.DecodeString(PixelBase64)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(decoded) != len(PixelGif) {
		t.Errorf("decoded length: expected %d, got %d", len(PixelGif), len(decoded))
	}
	for i := range decoded {
		if decoded[i] != PixelGif[i] {
			t.Errorf("byte %d: expected %x, got %x", i, PixelGif[i], decoded[i])
			break
		}
	}
}
