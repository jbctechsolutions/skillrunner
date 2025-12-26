package output

import (
	"os"
	"testing"
)

func TestIsColorSupported(t *testing.T) {
	// Save original env and restore after test
	origNoColor := os.Getenv("NO_COLOR")
	origForceColor := os.Getenv("FORCE_COLOR")
	origTerm := os.Getenv("TERM")
	defer func() {
		if origNoColor != "" {
			os.Setenv("NO_COLOR", origNoColor)
		} else {
			os.Unsetenv("NO_COLOR")
		}
		if origForceColor != "" {
			os.Setenv("FORCE_COLOR", origForceColor)
		} else {
			os.Unsetenv("FORCE_COLOR")
		}
		os.Setenv("TERM", origTerm)
		ResetColorDetection()
	}()

	tests := []struct {
		name       string
		noColor    string
		forceColor string
		term       string
		want       bool
	}{
		{
			name:    "NO_COLOR set",
			noColor: "1",
			term:    "xterm-256color",
			want:    false,
		},
		{
			name:       "FORCE_COLOR overrides",
			forceColor: "1",
			term:       "",
			want:       true,
		},
		{
			name: "TERM dumb",
			term: "dumb",
			want: false,
		},
		{
			name: "TERM empty",
			term: "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset detection before each test
			ResetColorDetection()

			// Set up environment
			os.Unsetenv("NO_COLOR")
			os.Unsetenv("FORCE_COLOR")
			os.Unsetenv("TERM")

			if tt.noColor != "" {
				os.Setenv("NO_COLOR", tt.noColor)
			}
			if tt.forceColor != "" {
				os.Setenv("FORCE_COLOR", tt.forceColor)
			}
			os.Setenv("TERM", tt.term)

			got := IsColorSupported()
			if got != tt.want {
				t.Errorf("IsColorSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResetColorDetection(t *testing.T) {
	// Set up a known state
	os.Setenv("FORCE_COLOR", "1")
	defer os.Unsetenv("FORCE_COLOR")

	ResetColorDetection()

	// Check that color is supported after reset
	if !IsColorSupported() {
		t.Error("IsColorSupported() = false, want true after FORCE_COLOR=1")
	}

	// Now change environment and verify cache needs reset
	os.Unsetenv("FORCE_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	// Should still return cached value
	if !IsColorSupported() {
		t.Log("Cache was invalidated unexpectedly")
	}

	// Reset and verify new state
	ResetColorDetection()
	if IsColorSupported() {
		t.Error("IsColorSupported() = true, want false after NO_COLOR=1 and reset")
	}
}

func TestSuccessText(t *testing.T) {
	text := "Operation completed"
	result := SuccessText(text)
	expected := string(ColorGreen) + "✓ " + text + string(ColorReset)
	if result != expected {
		t.Errorf("SuccessText(%q) = %q, want %q", text, result, expected)
	}
}

func TestErrorText(t *testing.T) {
	text := "Something went wrong"
	result := ErrorText(text)
	expected := string(ColorRed) + "✗ " + text + string(ColorReset)
	if result != expected {
		t.Errorf("ErrorText(%q) = %q, want %q", text, result, expected)
	}
}

func TestWarningText(t *testing.T) {
	text := "Be careful"
	result := WarningText(text)
	expected := string(ColorYellow) + "⚠ " + text + string(ColorReset)
	if result != expected {
		t.Errorf("WarningText(%q) = %q, want %q", text, result, expected)
	}
}

func TestInfoText(t *testing.T) {
	text := "FYI"
	result := InfoText(text)
	expected := string(ColorBlue) + "ℹ " + text + string(ColorReset)
	if result != expected {
		t.Errorf("InfoText(%q) = %q, want %q", text, result, expected)
	}
}

func TestHighlightText(t *testing.T) {
	text := "Important"
	result := HighlightText(text)
	expected := string(ColorBold) + string(ColorCyan) + text + string(ColorReset)
	if result != expected {
		t.Errorf("HighlightText(%q) = %q, want %q", text, result, expected)
	}
}

func TestMutedText(t *testing.T) {
	text := "Less important"
	result := MutedText(text)
	expected := string(ColorDim) + text + string(ColorReset)
	if result != expected {
		t.Errorf("MutedText(%q) = %q, want %q", text, result, expected)
	}
}

func TestBoldText(t *testing.T) {
	text := "Bold statement"
	result := BoldText(text)
	expected := string(ColorBold) + text + string(ColorReset)
	if result != expected {
		t.Errorf("BoldText(%q) = %q, want %q", text, result, expected)
	}
}

func TestUnderlineText(t *testing.T) {
	text := "Underlined"
	result := UnderlineText(text)
	expected := ColorUnderline + text + string(ColorReset)
	if result != expected {
		t.Errorf("UnderlineText(%q) = %q, want %q", text, result, expected)
	}
}

func TestColorIfEnabled(t *testing.T) {
	// Save and restore
	origNoColor := os.Getenv("NO_COLOR")
	origForceColor := os.Getenv("FORCE_COLOR")
	defer func() {
		if origNoColor != "" {
			os.Setenv("NO_COLOR", origNoColor)
		} else {
			os.Unsetenv("NO_COLOR")
		}
		if origForceColor != "" {
			os.Setenv("FORCE_COLOR", origForceColor)
		} else {
			os.Unsetenv("FORCE_COLOR")
		}
		ResetColorDetection()
	}()

	text := "colored text"

	// Test with colors forced on
	os.Unsetenv("NO_COLOR")
	os.Setenv("FORCE_COLOR", "1")
	ResetColorDetection()

	result := ColorIfEnabled(text, ColorRed)
	expected := string(ColorRed) + text + string(ColorReset)
	if result != expected {
		t.Errorf("ColorIfEnabled with color enabled = %q, want %q", result, expected)
	}

	// Test with colors disabled
	os.Unsetenv("FORCE_COLOR")
	os.Setenv("NO_COLOR", "1")
	ResetColorDetection()

	result = ColorIfEnabled(text, ColorRed)
	if result != text {
		t.Errorf("ColorIfEnabled with color disabled = %q, want %q", result, text)
	}
}

func TestConditionalTextFunctions(t *testing.T) {
	// Save and restore
	origNoColor := os.Getenv("NO_COLOR")
	origForceColor := os.Getenv("FORCE_COLOR")
	defer func() {
		if origNoColor != "" {
			os.Setenv("NO_COLOR", origNoColor)
		} else {
			os.Unsetenv("NO_COLOR")
		}
		if origForceColor != "" {
			os.Setenv("FORCE_COLOR", origForceColor)
		} else {
			os.Unsetenv("FORCE_COLOR")
		}
		ResetColorDetection()
	}()

	text := "test message"

	// Test with colors disabled
	os.Unsetenv("FORCE_COLOR")
	os.Setenv("NO_COLOR", "1")
	ResetColorDetection()

	tests := []struct {
		name string
		fn   func(string) string
		want string
	}{
		{"SuccessTextIfEnabled", SuccessTextIfEnabled, "✓ " + text},
		{"ErrorTextIfEnabled", ErrorTextIfEnabled, "✗ " + text},
		{"WarningTextIfEnabled", WarningTextIfEnabled, "⚠ " + text},
		{"InfoTextIfEnabled", InfoTextIfEnabled, "ℹ " + text},
	}

	for _, tt := range tests {
		t.Run(tt.name+" (disabled)", func(t *testing.T) {
			result := tt.fn(text)
			if result != tt.want {
				t.Errorf("%s(%q) = %q, want %q", tt.name, text, result, tt.want)
			}
		})
	}
}

func TestBackgroundColors(t *testing.T) {
	// Just verify the constants are defined correctly
	bgColors := []string{BgBlack, BgRed, BgGreen, BgYellow, BgBlue, BgMagenta, BgCyan, BgWhite}
	expected := []string{
		"\033[40m", "\033[41m", "\033[42m", "\033[43m",
		"\033[44m", "\033[45m", "\033[46m", "\033[47m",
	}

	for i, bg := range bgColors {
		if bg != expected[i] {
			t.Errorf("Background color at index %d = %q, want %q", i, bg, expected[i])
		}
	}
}

func TestAdditionalStyleCodes(t *testing.T) {
	// Verify additional style constants
	if ColorBrightBlack != "\033[90m" {
		t.Errorf("ColorBrightBlack = %q, want %q", ColorBrightBlack, "\033[90m")
	}
	if ColorBrightWhite != "\033[97m" {
		t.Errorf("ColorBrightWhite = %q, want %q", ColorBrightWhite, "\033[97m")
	}
	if ColorUnderline != "\033[4m" {
		t.Errorf("ColorUnderline = %q, want %q", ColorUnderline, "\033[4m")
	}
	if ColorBlink != "\033[5m" {
		t.Errorf("ColorBlink = %q, want %q", ColorBlink, "\033[5m")
	}
	if ColorReverse != "\033[7m" {
		t.Errorf("ColorReverse = %q, want %q", ColorReverse, "\033[7m")
	}
}
