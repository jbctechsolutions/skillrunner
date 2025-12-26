// Package output provides CLI output formatting utilities.
package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// StreamingOutput provides real-time output display for streaming LLM responses.
type StreamingOutput struct {
	mu              sync.Mutex
	writer          io.Writer
	colored         bool
	currentPhase    string
	phaseName       string
	phaseIndex      int
	totalPhases     int
	inputTokens     int
	outputTokens    int
	startTime       time.Time
	phaseStartTime  time.Time
	contentBuffer   strings.Builder
	showTokenCounts bool
	showPhaseInfo   bool
}

// StreamingOutputOption is a functional option for configuring StreamingOutput.
type StreamingOutputOption func(*StreamingOutput)

// NewStreamingOutput creates a new StreamingOutput with the given options.
func NewStreamingOutput(opts ...StreamingOutputOption) *StreamingOutput {
	so := &StreamingOutput{
		writer:          os.Stdout,
		colored:         true,
		showTokenCounts: true,
		showPhaseInfo:   true,
		startTime:       time.Now(),
	}

	for _, opt := range opts {
		opt(so)
	}

	return so
}

// WithStreamingWriter sets the output writer.
func WithStreamingWriter(w io.Writer) StreamingOutputOption {
	return func(so *StreamingOutput) {
		so.writer = w
	}
}

// WithStreamingColor enables or disables colored output.
func WithStreamingColor(enabled bool) StreamingOutputOption {
	return func(so *StreamingOutput) {
		so.colored = enabled
	}
}

// WithShowTokenCounts enables or disables token count display.
func WithShowTokenCounts(enabled bool) StreamingOutputOption {
	return func(so *StreamingOutput) {
		so.showTokenCounts = enabled
	}
}

// WithShowPhaseInfo enables or disables phase information display.
func WithShowPhaseInfo(enabled bool) StreamingOutputOption {
	return func(so *StreamingOutput) {
		so.showPhaseInfo = enabled
	}
}

// StartWorkflow initializes the streaming output for a new workflow.
func (so *StreamingOutput) StartWorkflow(skillName, skillVersion string, totalPhases int) {
	so.mu.Lock()
	defer so.mu.Unlock()

	so.totalPhases = totalPhases
	so.startTime = time.Now()
	so.inputTokens = 0
	so.outputTokens = 0
	so.contentBuffer.Reset()

	// Print header
	if so.colored {
		fmt.Fprintf(so.writer, "%s%s v%s%s\n", ColorBold, skillName, skillVersion, ColorReset)
	} else {
		fmt.Fprintf(so.writer, "%s v%s\n", skillName, skillVersion)
	}
	fmt.Fprintf(so.writer, "%s\n\n", strings.Repeat("─", len(skillName)+len(skillVersion)+3))
}

// StartPhase indicates a new phase is beginning.
func (so *StreamingOutput) StartPhase(phaseID, phaseName string, phaseIndex int) {
	so.mu.Lock()
	defer so.mu.Unlock()

	so.currentPhase = phaseID
	so.phaseName = phaseName
	so.phaseIndex = phaseIndex
	so.phaseStartTime = time.Now()
	so.contentBuffer.Reset()

	if so.showPhaseInfo {
		if so.colored {
			fmt.Fprintf(so.writer, "%s[%d/%d] %s%s\n",
				ColorCyan, phaseIndex, so.totalPhases, phaseName, ColorReset)
		} else {
			fmt.Fprintf(so.writer, "[%d/%d] %s\n", phaseIndex, so.totalPhases, phaseName)
		}
	}
}

// WriteChunk writes a streaming chunk to the output.
func (so *StreamingOutput) WriteChunk(chunk string) {
	so.mu.Lock()
	defer so.mu.Unlock()

	so.contentBuffer.WriteString(chunk)
	_, _ = fmt.Fprint(so.writer, chunk)
}

// CompletePhase marks the current phase as complete.
func (so *StreamingOutput) CompletePhase(inputTokens, outputTokens int, modelUsed string) {
	so.mu.Lock()
	defer so.mu.Unlock()

	// Ensure we end with a newline
	if so.contentBuffer.Len() > 0 {
		content := so.contentBuffer.String()
		if !strings.HasSuffix(content, "\n") {
			fmt.Fprintln(so.writer)
		}
	}

	so.inputTokens += inputTokens
	so.outputTokens += outputTokens

	phaseDuration := time.Since(so.phaseStartTime)

	if so.showTokenCounts || so.showPhaseInfo {
		fmt.Fprintln(so.writer)
		if so.colored {
			fmt.Fprintf(so.writer, "%s✓ %s completed%s", ColorGreen, so.phaseName, ColorReset)
		} else {
			fmt.Fprintf(so.writer, "✓ %s completed", so.phaseName)
		}

		if so.showTokenCounts {
			if so.colored {
				fmt.Fprintf(so.writer, " %s(tokens: %d in, %d out | %s | %s)%s\n",
					ColorDim, inputTokens, outputTokens, modelUsed, formatStreamDuration(phaseDuration), ColorReset)
			} else {
				fmt.Fprintf(so.writer, " (tokens: %d in, %d out | %s | %s)\n",
					inputTokens, outputTokens, modelUsed, formatStreamDuration(phaseDuration))
			}
		} else {
			fmt.Fprintln(so.writer)
		}
		fmt.Fprintln(so.writer)
	}
}

// FailPhase marks the current phase as failed.
func (so *StreamingOutput) FailPhase(err error) {
	so.mu.Lock()
	defer so.mu.Unlock()

	fmt.Fprintln(so.writer)
	if so.colored {
		fmt.Fprintf(so.writer, "%s✗ %s failed: %v%s\n", ColorRed, so.phaseName, err, ColorReset)
	} else {
		fmt.Fprintf(so.writer, "✗ %s failed: %v\n", so.phaseName, err)
	}
}

// CompleteWorkflow marks the workflow as complete and shows summary.
func (so *StreamingOutput) CompleteWorkflow(success bool) {
	so.mu.Lock()
	defer so.mu.Unlock()

	totalDuration := time.Since(so.startTime)
	totalTokens := so.inputTokens + so.outputTokens

	fmt.Fprintln(so.writer, strings.Repeat("─", 40))
	if success {
		if so.colored {
			fmt.Fprintf(so.writer, "%s✓ Workflow completed successfully%s\n", ColorGreen, ColorReset)
		} else {
			fmt.Fprintf(so.writer, "✓ Workflow completed successfully\n")
		}
	} else {
		if so.colored {
			fmt.Fprintf(so.writer, "%s✗ Workflow failed%s\n", ColorRed, ColorReset)
		} else {
			fmt.Fprintf(so.writer, "✗ Workflow failed\n")
		}
	}

	if so.showTokenCounts {
		if so.colored {
			fmt.Fprintf(so.writer, "%sTotal: %d tokens (%d in, %d out) | Duration: %s%s\n",
				ColorDim, totalTokens, so.inputTokens, so.outputTokens, formatStreamDuration(totalDuration), ColorReset)
		} else {
			fmt.Fprintf(so.writer, "Total: %d tokens (%d in, %d out) | Duration: %s\n",
				totalTokens, so.inputTokens, so.outputTokens, formatStreamDuration(totalDuration))
		}
	}
}

// UpdateTokens updates the running token count display.
func (so *StreamingOutput) UpdateTokens(inputTokens, outputTokens int) {
	so.mu.Lock()
	defer so.mu.Unlock()

	so.inputTokens = inputTokens
	so.outputTokens = outputTokens
}

// GetTotalTokens returns the current total token counts.
func (so *StreamingOutput) GetTotalTokens() (input, output int) {
	so.mu.Lock()
	defer so.mu.Unlock()
	return so.inputTokens, so.outputTokens
}

// formatStreamDuration formats a duration for display.
func formatStreamDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}

// LiveTokenCounter provides a real-time token counting display.
type LiveTokenCounter struct {
	mu           sync.Mutex
	writer       io.Writer
	inputTokens  int
	outputTokens int
	lastUpdate   time.Time
	updateRate   time.Duration
	colored      bool
}

// NewLiveTokenCounter creates a new live token counter.
func NewLiveTokenCounter(writer io.Writer, colored bool) *LiveTokenCounter {
	return &LiveTokenCounter{
		writer:     writer,
		colored:    colored,
		updateRate: 100 * time.Millisecond,
	}
}

// Update updates the token counts and refreshes the display.
func (ltc *LiveTokenCounter) Update(inputTokens, outputTokens int) {
	ltc.mu.Lock()
	defer ltc.mu.Unlock()

	now := time.Now()
	if now.Sub(ltc.lastUpdate) < ltc.updateRate {
		// Rate limit updates
		ltc.inputTokens = inputTokens
		ltc.outputTokens = outputTokens
		return
	}

	ltc.inputTokens = inputTokens
	ltc.outputTokens = outputTokens
	ltc.lastUpdate = now

	// Clear and rewrite the line
	total := inputTokens + outputTokens
	if ltc.colored {
		fmt.Fprintf(ltc.writer, "\r%sTokens: %d (in: %d, out: %d)%s",
			ColorDim, total, inputTokens, outputTokens, ColorReset)
	} else {
		fmt.Fprintf(ltc.writer, "\rTokens: %d (in: %d, out: %d)",
			total, inputTokens, outputTokens)
	}
}

// Clear clears the token counter display.
func (ltc *LiveTokenCounter) Clear() {
	ltc.mu.Lock()
	defer ltc.mu.Unlock()

	// Clear the line
	fmt.Fprintf(ltc.writer, "\r%s\r", strings.Repeat(" ", 40))
}

// Final prints the final token count.
func (ltc *LiveTokenCounter) Final() {
	ltc.mu.Lock()
	defer ltc.mu.Unlock()

	total := ltc.inputTokens + ltc.outputTokens
	fmt.Fprintf(ltc.writer, "\rTokens: %d (in: %d, out: %d)\n",
		total, ltc.inputTokens, ltc.outputTokens)
}
