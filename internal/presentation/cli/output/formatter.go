// Package output provides CLI output formatting utilities.
// It supports table, JSON, text, and colored output formats with thread-safe operations.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// Format represents the output format type.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatText  Format = "text"
)

// Color represents ANSI color codes for terminal output.
type Color string

const (
	ColorReset   Color = "\033[0m"
	ColorRed     Color = "\033[31m"
	ColorGreen   Color = "\033[32m"
	ColorYellow  Color = "\033[33m"
	ColorBlue    Color = "\033[34m"
	ColorMagenta Color = "\033[35m"
	ColorCyan    Color = "\033[36m"
	ColorWhite   Color = "\033[37m"
	ColorBold    Color = "\033[1m"
	ColorDim     Color = "\033[2m"
)

// Formatter handles output formatting with support for multiple formats and colors.
type Formatter struct {
	mu           sync.Mutex
	writer       io.Writer
	format       Format
	colorEnabled bool
	indent       string
}

// Option is a functional option for configuring a Formatter.
type Option func(*Formatter)

// NewFormatter creates a new Formatter with the given options.
func NewFormatter(opts ...Option) *Formatter {
	f := &Formatter{
		writer:       os.Stdout,
		format:       FormatText,
		colorEnabled: true,
		indent:       "  ",
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithWriter sets the output writer.
func WithWriter(w io.Writer) Option {
	return func(f *Formatter) {
		f.writer = w
	}
}

// WithFormat sets the output format.
func WithFormat(format Format) Option {
	return func(f *Formatter) {
		f.format = format
	}
}

// WithColor enables or disables colored output.
func WithColor(enabled bool) Option {
	return func(f *Formatter) {
		f.colorEnabled = enabled
	}
}

// WithIndent sets the indentation string for nested output.
func WithIndent(indent string) Option {
	return func(f *Formatter) {
		f.indent = indent
	}
}

// Format returns the current output format.
func (f *Formatter) Format() Format {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.format
}

// SetFormat changes the output format.
func (f *Formatter) SetFormat(format Format) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.format = format
}

// SetColor enables or disables colored output.
func (f *Formatter) SetColor(enabled bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.colorEnabled = enabled
}

// Write writes raw bytes to the output, implementing io.Writer.
func (f *Formatter) Write(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.writer.Write(p)
}

// Print writes formatted output without a newline.
func (f *Formatter) Print(format string, args ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, err := fmt.Fprintf(f.writer, format, args...)
	return err
}

// Println writes formatted output with a newline.
func (f *Formatter) Println(format string, args ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, err := fmt.Fprintf(f.writer, format+"\n", args...)
	return err
}

// Colorize wraps text with ANSI color codes if color is enabled.
func (f *Formatter) Colorize(text string, color Color) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.colorEnabled {
		return text
	}
	return string(color) + text + string(ColorReset)
}

// Success prints a success message in green.
func (f *Formatter) Success(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return f.Println("%s", f.Colorize("✓ "+msg, ColorGreen))
}

// Error prints an error message in red.
func (f *Formatter) Error(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return f.Println("%s", f.Colorize("✗ "+msg, ColorRed))
}

// Warning prints a warning message in yellow.
func (f *Formatter) Warning(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return f.Println("%s", f.Colorize("⚠ "+msg, ColorYellow))
}

// Info prints an info message in blue.
func (f *Formatter) Info(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	return f.Println("%s", f.Colorize("ℹ "+msg, ColorBlue))
}

// Bold prints text in bold.
func (f *Formatter) Bold(text string) string {
	return f.Colorize(text, ColorBold)
}

// Dim prints text in dim/muted style.
func (f *Formatter) Dim(text string) string {
	return f.Colorize(text, ColorDim)
}

// Header outputs a section header with underline.
func (f *Formatter) Header(msg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.colorEnabled {
		fmt.Fprintf(f.writer, "%s%s%s\n", ColorBold, msg, ColorReset)
	} else {
		fmt.Fprintln(f.writer, msg)
	}
	fmt.Fprintln(f.writer, strings.Repeat("─", len(msg)))
	return nil
}

// SubHeader outputs a sub-header.
func (f *Formatter) SubHeader(msg string) error {
	return f.Println("%s", f.Colorize(msg, ColorCyan))
}

// Item outputs a key-value pair for structured display.
func (f *Formatter) Item(key, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.colorEnabled {
		_, err := fmt.Fprintf(f.writer, "  %s%s%s: %s\n", ColorDim, key, ColorReset, value)
		return err
	}
	_, err := fmt.Fprintf(f.writer, "  %s: %s\n", key, value)
	return err
}

// BulletItem outputs a bulleted list item.
func (f *Formatter) BulletItem(msg string) error {
	return f.Println("  • %s", msg)
}

// TableColumn defines a column in a table.
type TableColumn struct {
	Header string
	Width  int
	Align  Alignment
}

// Alignment defines text alignment in table cells.
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// TableData represents data for table formatting.
type TableData struct {
	Columns []TableColumn
	Rows    [][]string
}

// Table writes data as a formatted table.
func (f *Formatter) Table(data TableData) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(data.Columns) == 0 {
		return nil
	}

	// Calculate column widths
	widths := make([]int, len(data.Columns))
	for i, col := range data.Columns {
		widths[i] = len(col.Header)
		if col.Width > widths[i] {
			widths[i] = col.Width
		}
	}

	// Check row widths
	for _, row := range data.Rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Build header
	var header strings.Builder
	var separator strings.Builder
	for i, col := range data.Columns {
		header.WriteString(f.padCell(col.Header, widths[i], col.Align))
		separator.WriteString(strings.Repeat("-", widths[i]))
		if i < len(data.Columns)-1 {
			header.WriteString("  ")
			separator.WriteString("  ")
		}
	}

	// Write header with color
	var err error
	if f.colorEnabled {
		_, err = fmt.Fprintf(f.writer, "%s%s%s\n", ColorBold, header.String(), ColorReset)
	} else {
		_, err = fmt.Fprintln(f.writer, header.String())
	}
	if err != nil {
		return err
	}

	if _, err = fmt.Fprintln(f.writer, separator.String()); err != nil {
		return err
	}

	// Write rows
	for _, row := range data.Rows {
		var rowStr strings.Builder
		for i, cell := range row {
			if i >= len(data.Columns) {
				break
			}
			rowStr.WriteString(f.padCell(cell, widths[i], data.Columns[i].Align))
			if i < len(data.Columns)-1 {
				rowStr.WriteString("  ")
			}
		}
		if _, err = fmt.Fprintln(f.writer, rowStr.String()); err != nil {
			return err
		}
	}

	return nil
}

// padCell pads a cell value to the specified width with the given alignment.
func (f *Formatter) padCell(text string, width int, align Alignment) string {
	if len(text) >= width {
		return text
	}

	padding := width - len(text)

	switch align {
	case AlignRight:
		return strings.Repeat(" ", padding) + text
	case AlignCenter:
		left := padding / 2
		right := padding - left
		return strings.Repeat(" ", left) + text + strings.Repeat(" ", right)
	default: // AlignLeft
		return text + strings.Repeat(" ", padding)
	}
}

// JSON writes data as formatted JSON.
func (f *Formatter) JSON(data any) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	encoder := json.NewEncoder(f.writer)
	encoder.SetIndent("", f.indent)
	return encoder.Encode(data)
}

// JSONCompact writes data as compact JSON without indentation.
func (f *Formatter) JSONCompact(data any) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	return json.NewEncoder(f.writer).Encode(data)
}

// FormatAuto formats data according to the current format setting.
func (f *Formatter) FormatAuto(data any, tableData *TableData) error {
	switch f.Format() {
	case FormatJSON:
		return f.JSON(data)
	case FormatTable:
		if tableData != nil {
			return f.Table(*tableData)
		}
		return f.JSON(data)
	default:
		if tableData != nil {
			return f.Table(*tableData)
		}
		return f.JSON(data)
	}
}

// Spinner provides a simple progress indicator for long-running operations.
type Spinner struct {
	mu       sync.Mutex
	frames   []string
	index    int
	message  string
	writer   io.Writer
	running  bool
	done     chan struct{}
	stopped  chan struct{} // signals that animate goroutine has exited
	interval time.Duration
	colored  bool
}

// SpinnerOption is a functional option for configuring a Spinner.
type SpinnerOption func(*Spinner)

// NewSpinner creates a new Spinner with the given options.
func NewSpinner(message string, opts ...SpinnerOption) *Spinner {
	s := &Spinner{
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		message:  message,
		writer:   os.Stdout,
		interval: 80 * time.Millisecond,
		colored:  true,
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// WithSpinnerWriter sets the output writer for the spinner.
func WithSpinnerWriter(w io.Writer) SpinnerOption {
	return func(s *Spinner) {
		s.writer = w
	}
}

// WithSpinnerFrames sets custom animation frames.
func WithSpinnerFrames(frames []string) SpinnerOption {
	return func(s *Spinner) {
		if len(frames) > 0 {
			s.frames = frames
		}
	}
}

// WithSpinnerInterval sets the animation interval.
func WithSpinnerInterval(d time.Duration) SpinnerOption {
	return func(s *Spinner) {
		s.interval = d
	}
}

// WithSpinnerColor enables or disables colored output for the spinner.
func WithSpinnerColor(enabled bool) SpinnerOption {
	return func(s *Spinner) {
		s.colored = enabled
	}
}

// Start begins the spinner animation.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.done = make(chan struct{})
	s.stopped = make(chan struct{})
	s.mu.Unlock()

	go s.animate()
}

// Stop stops the spinner animation and clears the line.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.done)
	stopped := s.stopped
	s.mu.Unlock()

	// Wait for animate goroutine to exit before writing to writer
	<-stopped

	// Clear the line - error intentionally ignored for terminal output
	_, _ = fmt.Fprintf(s.writer, "\r%s\r", strings.Repeat(" ", len(s.message)+4))
}

// StopWithSuccess stops the spinner and shows a success message.
func (s *Spinner) StopWithSuccess(message string) {
	s.Stop()
	// Error intentionally ignored for terminal output
	if s.colored {
		_, _ = fmt.Fprintf(s.writer, "%s✓ %s%s\n", ColorGreen, message, ColorReset)
	} else {
		_, _ = fmt.Fprintf(s.writer, "✓ %s\n", message)
	}
}

// StopWithError stops the spinner and shows an error message.
func (s *Spinner) StopWithError(message string) {
	s.Stop()
	// Error intentionally ignored for terminal output
	if s.colored {
		_, _ = fmt.Fprintf(s.writer, "%s✗ %s%s\n", ColorRed, message, ColorReset)
	} else {
		_, _ = fmt.Fprintf(s.writer, "✗ %s\n", message)
	}
}

// UpdateMessage updates the spinner message.
func (s *Spinner) UpdateMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.message = message
}

// animate runs the spinner animation loop.
func (s *Spinner) animate() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	defer close(s.stopped) // signal that we've exited

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			frame := s.frames[s.index]
			s.index = (s.index + 1) % len(s.frames)
			message := s.message
			s.mu.Unlock()

			// Error intentionally ignored for terminal output
			if s.colored {
				_, _ = fmt.Fprintf(s.writer, "\r%s%s%s %s", ColorCyan, frame, ColorReset, message)
			} else {
				_, _ = fmt.Fprintf(s.writer, "\r%s %s", frame, message)
			}
		}
	}
}

// ProgressBar represents a progress bar for tracking completion.
type ProgressBar struct {
	mu        sync.Mutex
	total     int
	current   int
	width     int
	message   string
	writer    io.Writer
	colored   bool
	fillChar  string
	emptyChar string
}

// ProgressBarOption is a functional option for configuring a ProgressBar.
type ProgressBarOption func(*ProgressBar)

// NewProgressBar creates a new ProgressBar with the given options.
func NewProgressBar(total int, message string, opts ...ProgressBarOption) *ProgressBar {
	p := &ProgressBar{
		total:     total,
		width:     40,
		message:   message,
		writer:    os.Stdout,
		colored:   true,
		fillChar:  "█",
		emptyChar: "░",
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// WithProgressBarWriter sets the output writer.
func WithProgressBarWriter(w io.Writer) ProgressBarOption {
	return func(p *ProgressBar) {
		p.writer = w
	}
}

// WithProgressBarWidth sets the bar width.
func WithProgressBarWidth(width int) ProgressBarOption {
	return func(p *ProgressBar) {
		if width > 0 {
			p.width = width
		}
	}
}

// WithProgressBarColor enables or disables colored output.
func WithProgressBarColor(enabled bool) ProgressBarOption {
	return func(p *ProgressBar) {
		p.colored = enabled
	}
}

// WithProgressBarChars sets the fill and empty characters.
func WithProgressBarChars(fill, empty string) ProgressBarOption {
	return func(p *ProgressBar) {
		p.fillChar = fill
		p.emptyChar = empty
	}
}

// Increment advances the progress bar by one.
func (p *ProgressBar) Increment() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.current < p.total {
		p.current++
	}
	p.render()
}

// Set sets the current progress value.
func (p *ProgressBar) Set(value int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if value < 0 {
		value = 0
	}
	if value > p.total {
		value = p.total
	}
	p.current = value
	p.render()
}

// SetMessage updates the progress message.
func (p *ProgressBar) SetMessage(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.message = message
	p.render()
}

// Complete marks the progress bar as complete.
func (p *ProgressBar) Complete() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = p.total
	p.render()
	// Error intentionally ignored for terminal output
	_, _ = fmt.Fprintln(p.writer)
}

// render draws the progress bar.
func (p *ProgressBar) render() {
	if p.total == 0 {
		return
	}

	percent := float64(p.current) / float64(p.total)
	filled := int(percent * float64(p.width))
	empty := p.width - filled

	bar := strings.Repeat(p.fillChar, filled) + strings.Repeat(p.emptyChar, empty)
	percentStr := fmt.Sprintf("%3.0f%%", percent*100)

	// Error intentionally ignored for terminal output
	if p.colored {
		_, _ = fmt.Fprintf(p.writer, "\r%s [%s%s%s] %s %s",
			p.message,
			ColorGreen, bar, ColorReset,
			percentStr,
			strings.Repeat(" ", 10)) // padding to clear previous longer messages
	} else {
		_, _ = fmt.Fprintf(p.writer, "\r%s [%s] %s%s",
			p.message,
			bar,
			percentStr,
			strings.Repeat(" ", 10))
	}
}

// StreamWriter provides a thread-safe writer for streaming output.
type StreamWriter struct {
	mu          sync.Mutex
	writer      io.Writer
	prefix      string
	colored     bool
	prefixColor Color
}

// StreamWriterOption is a functional option for configuring a StreamWriter.
type StreamWriterOption func(*StreamWriter)

// NewStreamWriter creates a new StreamWriter with the given options.
func NewStreamWriter(opts ...StreamWriterOption) *StreamWriter {
	sw := &StreamWriter{
		writer:      os.Stdout,
		colored:     true,
		prefixColor: ColorCyan,
	}

	for _, opt := range opts {
		opt(sw)
	}

	return sw
}

// WithStreamWriterWriter sets the output writer.
func WithStreamWriterWriter(w io.Writer) StreamWriterOption {
	return func(sw *StreamWriter) {
		sw.writer = w
	}
}

// WithStreamWriterPrefix sets the line prefix.
func WithStreamWriterPrefix(prefix string) StreamWriterOption {
	return func(sw *StreamWriter) {
		sw.prefix = prefix
	}
}

// WithStreamWriterColor enables or disables colored output.
func WithStreamWriterColor(enabled bool) StreamWriterOption {
	return func(sw *StreamWriter) {
		sw.colored = enabled
	}
}

// WithStreamWriterPrefixColor sets the prefix color.
func WithStreamWriterPrefixColor(color Color) StreamWriterOption {
	return func(sw *StreamWriter) {
		sw.prefixColor = color
	}
}

// Write implements io.Writer interface.
func (sw *StreamWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if sw.prefix == "" {
		return sw.writer.Write(p)
	}

	// Split by lines and add prefix
	lines := strings.Split(string(p), "\n")
	var output strings.Builder

	for i, line := range lines {
		if i == len(lines)-1 && line == "" {
			continue
		}
		if sw.colored {
			output.WriteString(string(sw.prefixColor))
			output.WriteString(sw.prefix)
			output.WriteString(string(ColorReset))
		} else {
			output.WriteString(sw.prefix)
		}
		output.WriteString(line)
		if i < len(lines)-1 {
			output.WriteString("\n")
		}
	}

	written, err := sw.writer.Write([]byte(output.String()))
	if err != nil {
		return written, err
	}

	return len(p), nil
}

// WriteLine writes a line with the configured prefix.
func (sw *StreamWriter) WriteLine(format string, args ...any) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	line := fmt.Sprintf(format, args...)

	if sw.prefix != "" {
		if sw.colored {
			_, err := fmt.Fprintf(sw.writer, "%s%s%s%s\n", sw.prefixColor, sw.prefix, ColorReset, line)
			return err
		}
		_, err := fmt.Fprintf(sw.writer, "%s%s\n", sw.prefix, line)
		return err
	}

	_, err := fmt.Fprintln(sw.writer, line)
	return err
}

// ParseFormat parses a string into a Format type.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "table":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "text", "":
		return FormatText, nil
	default:
		return FormatText, fmt.Errorf("unknown format: %s", s)
	}
}
