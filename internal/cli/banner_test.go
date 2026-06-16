package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestBannerNonEmpty(t *testing.T) {
	if monarchBanner == "" {
		t.Fatal("monarchBanner is empty")
	}
}

func TestBannerContainsPlusSigns(t *testing.T) {
	if !strings.Contains(monarchBanner, "+") {
		t.Fatal("monarchBanner does not contain '+' characters")
	}
}

func TestBannerLineCount(t *testing.T) {
	lines := strings.Split(monarchBanner, "\n")
	// The banner should have a reasonable number of lines (it's ASCII art).
	if len(lines) < 10 {
		t.Fatalf("monarchBanner has %d lines, want at least 10", len(lines))
	}
}

func TestBannerUsedInVersionOutput(t *testing.T) {
	// Verify that writeVersion includes the banner in plain text mode.
	var buf bytes.Buffer
	if err := writeVersion(&buf, "default", false, false, 0); err != nil {
		t.Fatalf("writeVersion() error = %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, monarchBanner) {
		t.Fatal("writeVersion() plain text output does not contain banner")
	}
}
