// Package output provides terminal output formatting utilities for the CLI.
package output

import (
	"os"
)

// colorsEnabled caches the result of color support detection.
var colorsEnabled *bool

// IsColorSupported determines if color output should be enabled.
// It checks for NO_COLOR environment variable and terminal capability.
func IsColorSupported() bool {
	if colorsEnabled != nil {
		return *colorsEnabled
	}

	enabled := detectColorSupport()
	colorsEnabled = &enabled
	return enabled
}

// detectColorSupport checks environment variables and terminal capabilities.
func detectColorSupport() bool {
	// NO_COLOR takes precedence - if set to any value, disable colors
	// See https://no-color.org/
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		return false
	}

	// FORCE_COLOR forces color output regardless of terminal detection
	if _, exists := os.LookupEnv("FORCE_COLOR"); exists {
		return true
	}

	// Check if stdout is a terminal
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	// Check if it's a character device (terminal)
	if stat.Mode()&os.ModeCharDevice == 0 {
		return false
	}

	// Check TERM environment variable
	term := os.Getenv("TERM")
	if term == "" || term == "dumb" {
		return false
	}

	return true
}

// ResetColorDetection clears the cached color detection result.
// This is useful for testing or when environment variables change.
func ResetColorDetection() {
	colorsEnabled = nil
}

// Additional ANSI style codes not defined in the main output constants.
const (
	ColorBrightBlack = "\033[90m"
	ColorBrightWhite = "\033[97m"
	ColorUnderline   = "\033[4m"
	ColorBlink       = "\033[5m"
	ColorReverse     = "\033[7m"
)

// Background colors.
const (
	BgBlack   = "\033[40m"
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

// Styled helper functions that use the existing Color constants.
// These functions apply color codes directly and reset at the end.

// SuccessText formats text as a success message (green with checkmark).
func SuccessText(text string) string {
	return string(ColorGreen) + "✓ " + text + string(ColorReset)
}

// ErrorText formats text as an error message (red with X).
func ErrorText(text string) string {
	return string(ColorRed) + "✗ " + text + string(ColorReset)
}

// WarningText formats text as a warning message (yellow with warning symbol).
func WarningText(text string) string {
	return string(ColorYellow) + "⚠ " + text + string(ColorReset)
}

// InfoText formats text as an info message (blue with info symbol).
func InfoText(text string) string {
	return string(ColorBlue) + "ℹ " + text + string(ColorReset)
}

// HighlightText formats text as highlighted (cyan and bold).
func HighlightText(text string) string {
	return string(ColorBold) + string(ColorCyan) + text + string(ColorReset)
}

// MutedText formats text as muted (dim).
func MutedText(text string) string {
	return string(ColorDim) + text + string(ColorReset)
}

// BoldText formats text as bold.
func BoldText(text string) string {
	return string(ColorBold) + text + string(ColorReset)
}

// UnderlineText formats text with underline.
func UnderlineText(text string) string {
	return ColorUnderline + text + string(ColorReset)
}

// ColorIfEnabled applies color only if colors are supported.
func ColorIfEnabled(text string, color Color) string {
	if !IsColorSupported() {
		return text
	}
	return string(color) + text + string(ColorReset)
}

// SuccessTextIfEnabled formats as success only if colors are supported.
func SuccessTextIfEnabled(text string) string {
	if !IsColorSupported() {
		return "✓ " + text
	}
	return SuccessText(text)
}

// ErrorTextIfEnabled formats as error only if colors are supported.
func ErrorTextIfEnabled(text string) string {
	if !IsColorSupported() {
		return "✗ " + text
	}
	return ErrorText(text)
}

// WarningTextIfEnabled formats as warning only if colors are supported.
func WarningTextIfEnabled(text string) string {
	if !IsColorSupported() {
		return "⚠ " + text
	}
	return WarningText(text)
}

// InfoTextIfEnabled formats as info only if colors are supported.
func InfoTextIfEnabled(text string) string {
	if !IsColorSupported() {
		return "ℹ " + text
	}
	return InfoText(text)
}
