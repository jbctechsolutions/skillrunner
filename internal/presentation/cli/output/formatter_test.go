package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewFormatter(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		f := NewFormatter()
		if f.format != FormatText {
			t.Errorf("expected format %v, got %v", FormatText, f.format)
		}
		if !f.colorEnabled {
			t.Error("expected color to be enabled by default")
		}
	})

	t.Run("with custom options", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewFormatter(
			WithWriter(&buf),
			WithFormat(FormatJSON),
			WithColor(false),
			WithIndent("    "),
		)

		if f.format != FormatJSON {
			t.Errorf("expected format %v, got %v", FormatJSON, f.format)
		}
		if f.colorEnabled {
			t.Error("expected color to be disabled")
		}
		if f.indent != "    " {
			t.Errorf("expected indent '    ', got %q", f.indent)
		}
	})
}

func TestFormatter_Print(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(WithWriter(&buf))

	err := f.Print("hello %s", "world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := buf.String(); got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
}

func TestFormatter_Println(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(WithWriter(&buf))

	err := f.Println("hello %s", "world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := buf.String(); got != "hello world\n" {
		t.Errorf("expected 'hello world\\n', got %q", got)
	}
}

func TestFormatter_Colorize(t *testing.T) {
	t.Run("with color enabled", func(t *testing.T) {
		f := NewFormatter(WithColor(true))
		result := f.Colorize("test", ColorRed)

		if !strings.Contains(result, string(ColorRed)) {
			t.Error("expected result to contain red color code")
		}
		if !strings.Contains(result, string(ColorReset)) {
			t.Error("expected result to contain reset code")
		}
		if !strings.Contains(result, "test") {
			t.Error("expected result to contain 'test'")
		}
	})

	t.Run("with color disabled", func(t *testing.T) {
		f := NewFormatter(WithColor(false))
		result := f.Colorize("test", ColorRed)

		if result != "test" {
			t.Errorf("expected 'test', got %q", result)
		}
	})
}

func TestFormatter_MessageTypes(t *testing.T) {
	tests := []struct {
		name   string
		method func(*Formatter, string, ...any) error
		prefix string
	}{
		{"Success", (*Formatter).Success, "✓"},
		{"Error", (*Formatter).Error, "✗"},
		{"Warning", (*Formatter).Warning, "⚠"},
		{"Info", (*Formatter).Info, "ℹ"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			f := NewFormatter(WithWriter(&buf), WithColor(false))

			err := tc.method(f, "test message")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, tc.prefix) {
				t.Errorf("expected output to contain %q, got %q", tc.prefix, output)
			}
			if !strings.Contains(output, "test message") {
				t.Errorf("expected output to contain 'test message', got %q", output)
			}
		})
	}
}

func TestFormatter_Table(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(WithWriter(&buf), WithColor(false))

	data := TableData{
		Columns: []TableColumn{
			{Header: "Name", Width: 10, Align: AlignLeft},
			{Header: "Status", Width: 8, Align: AlignCenter},
			{Header: "Count", Width: 6, Align: AlignRight},
		},
		Rows: [][]string{
			{"skill1", "ready", "5"},
			{"skill2", "pending", "10"},
		},
	}

	err := f.Table(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Check header
	if !strings.Contains(output, "Name") {
		t.Error("expected output to contain 'Name'")
	}
	if !strings.Contains(output, "Status") {
		t.Error("expected output to contain 'Status'")
	}
	if !strings.Contains(output, "Count") {
		t.Error("expected output to contain 'Count'")
	}

	// Check rows
	if !strings.Contains(output, "skill1") {
		t.Error("expected output to contain 'skill1'")
	}
	if !strings.Contains(output, "skill2") {
		t.Error("expected output to contain 'skill2'")
	}

	// Check separator
	if !strings.Contains(output, "---") {
		t.Error("expected output to contain separator")
	}
}

func TestFormatter_Table_EmptyColumns(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(WithWriter(&buf))

	data := TableData{
		Columns: []TableColumn{},
		Rows:    [][]string{{"a", "b"}},
	}

	err := f.Table(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if buf.Len() != 0 {
		t.Errorf("expected no output for empty columns, got %q", buf.String())
	}
}

func TestFormatter_JSON(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(WithWriter(&buf), WithIndent("  "))

	data := map[string]any{
		"name":   "test",
		"status": "ok",
	}

	err := f.JSON(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify it's valid JSON
	var decoded map[string]any
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Verify values
	if decoded["name"] != "test" {
		t.Errorf("expected name 'test', got %v", decoded["name"])
	}
	if decoded["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", decoded["status"])
	}
}

func TestFormatter_JSONCompact(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(WithWriter(&buf))

	data := map[string]string{"key": "value"}

	err := f.JSONCompact(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Compact JSON should not have indentation
	output := buf.String()
	if strings.Contains(output, "\n  ") {
		t.Error("expected compact JSON without indentation")
	}
}

func TestFormatter_FormatAuto(t *testing.T) {
	t.Run("JSON format", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewFormatter(WithWriter(&buf), WithFormat(FormatJSON))

		data := map[string]string{"key": "value"}
		err := f.FormatAuto(data, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(buf.String(), "key") {
			t.Error("expected JSON output")
		}
	})

	t.Run("Table format with data", func(t *testing.T) {
		var buf bytes.Buffer
		f := NewFormatter(WithWriter(&buf), WithFormat(FormatTable), WithColor(false))

		tableData := &TableData{
			Columns: []TableColumn{{Header: "Col", Width: 5}},
			Rows:    [][]string{{"val"}},
		}

		err := f.FormatAuto(nil, tableData)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(buf.String(), "Col") {
			t.Error("expected table output")
		}
	})
}

func TestFormatter_SetFormat(t *testing.T) {
	f := NewFormatter()

	f.SetFormat(FormatJSON)
	if f.Format() != FormatJSON {
		t.Errorf("expected FormatJSON, got %v", f.Format())
	}

	f.SetFormat(FormatTable)
	if f.Format() != FormatTable {
		t.Errorf("expected FormatTable, got %v", f.Format())
	}
}

func TestFormatter_SetColor(t *testing.T) {
	f := NewFormatter()

	f.SetColor(false)
	result := f.Colorize("test", ColorRed)
	if result != "test" {
		t.Errorf("expected no color, got %q", result)
	}

	f.SetColor(true)
	result = f.Colorize("test", ColorRed)
	if !strings.Contains(result, string(ColorRed)) {
		t.Error("expected color to be applied")
	}
}

func TestFormatter_padCell(t *testing.T) {
	f := NewFormatter()

	tests := []struct {
		text     string
		width    int
		align    Alignment
		expected string
	}{
		{"abc", 6, AlignLeft, "abc   "},
		{"abc", 6, AlignRight, "   abc"},
		{"abc", 6, AlignCenter, " abc  "},
		{"abc", 3, AlignLeft, "abc"},
		{"abc", 2, AlignLeft, "abc"}, // text longer than width
	}

	for _, tc := range tests {
		result := f.padCell(tc.text, tc.width, tc.align)
		if result != tc.expected {
			t.Errorf("padCell(%q, %d, %v) = %q, expected %q",
				tc.text, tc.width, tc.align, result, tc.expected)
		}
	}
}

func TestFormatter_Write(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(WithWriter(&buf))

	n, err := f.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 5 {
		t.Errorf("expected 5 bytes written, got %d", n)
	}
	if buf.String() != "hello" {
		t.Errorf("expected 'hello', got %q", buf.String())
	}
}

func TestSpinner(t *testing.T) {
	t.Run("basic lifecycle", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner("Loading...",
			WithSpinnerWriter(&buf),
			WithSpinnerInterval(10*time.Millisecond),
			WithSpinnerColor(false),
		)

		s.Start()
		time.Sleep(30 * time.Millisecond)
		s.Stop()

		// Verify something was written
		if buf.Len() == 0 {
			t.Error("expected spinner to write output")
		}
	})

	t.Run("double start", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner("Test",
			WithSpinnerWriter(&buf),
			WithSpinnerInterval(10*time.Millisecond),
		)

		s.Start()
		s.Start() // Should be idempotent
		s.Stop()
	})

	t.Run("stop without start", func(t *testing.T) {
		s := NewSpinner("Test")
		s.Stop() // Should not panic
	})

	t.Run("stop with success", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner("Loading",
			WithSpinnerWriter(&buf),
			WithSpinnerInterval(10*time.Millisecond),
			WithSpinnerColor(false),
		)

		s.Start()
		time.Sleep(20 * time.Millisecond)
		s.StopWithSuccess("Done!")

		if !strings.Contains(buf.String(), "Done!") {
			t.Error("expected success message in output")
		}
	})

	t.Run("stop with error", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner("Loading",
			WithSpinnerWriter(&buf),
			WithSpinnerInterval(10*time.Millisecond),
			WithSpinnerColor(false),
		)

		s.Start()
		time.Sleep(20 * time.Millisecond)
		s.StopWithError("Failed!")

		if !strings.Contains(buf.String(), "Failed!") {
			t.Error("expected error message in output")
		}
	})

	t.Run("update message", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner("Initial",
			WithSpinnerWriter(&buf),
			WithSpinnerInterval(10*time.Millisecond),
		)

		s.Start()
		s.UpdateMessage("Updated")
		time.Sleep(30 * time.Millisecond)
		s.Stop()

		if !strings.Contains(buf.String(), "Updated") {
			t.Error("expected updated message in output")
		}
	})

	t.Run("custom frames", func(t *testing.T) {
		var buf bytes.Buffer
		s := NewSpinner("Test",
			WithSpinnerWriter(&buf),
			WithSpinnerFrames([]string{"A", "B", "C"}),
			WithSpinnerInterval(10*time.Millisecond),
			WithSpinnerColor(false),
		)

		s.Start()
		time.Sleep(50 * time.Millisecond)
		s.Stop()

		output := buf.String()
		if !strings.Contains(output, "A") && !strings.Contains(output, "B") && !strings.Contains(output, "C") {
			t.Error("expected custom frames in output")
		}
	})
}

func TestProgressBar(t *testing.T) {
	t.Run("basic progress", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewProgressBar(10, "Progress",
			WithProgressBarWriter(&buf),
			WithProgressBarColor(false),
		)

		p.Set(5)

		output := buf.String()
		if !strings.Contains(output, "50%") {
			t.Errorf("expected 50%% in output, got %q", output)
		}
	})

	t.Run("increment", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewProgressBar(10, "Progress",
			WithProgressBarWriter(&buf),
			WithProgressBarWidth(20),
			WithProgressBarColor(false),
		)

		p.Increment()
		p.Increment()

		output := buf.String()
		if !strings.Contains(output, "20%") {
			t.Errorf("expected 20%% in output, got %q", output)
		}
	})

	t.Run("complete", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewProgressBar(10, "Progress",
			WithProgressBarWriter(&buf),
			WithProgressBarColor(false),
		)

		p.Complete()

		output := buf.String()
		if !strings.Contains(output, "100%") {
			t.Errorf("expected 100%% in output, got %q", output)
		}
	})

	t.Run("custom chars", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewProgressBar(10, "Progress",
			WithProgressBarWriter(&buf),
			WithProgressBarChars("#", "-"),
			WithProgressBarColor(false),
		)

		p.Set(5)

		output := buf.String()
		if !strings.Contains(output, "#") {
			t.Error("expected custom fill char in output")
		}
		if !strings.Contains(output, "-") {
			t.Error("expected custom empty char in output")
		}
	})

	t.Run("set bounds", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewProgressBar(10, "Progress",
			WithProgressBarWriter(&buf),
			WithProgressBarColor(false),
		)

		p.Set(-5)
		if p.current != 0 {
			t.Errorf("expected current to be 0 for negative value, got %d", p.current)
		}

		p.Set(100)
		if p.current != 10 {
			t.Errorf("expected current to be capped at total, got %d", p.current)
		}
	})

	t.Run("update message", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewProgressBar(10, "Initial",
			WithProgressBarWriter(&buf),
			WithProgressBarColor(false),
		)

		p.SetMessage("Updated")
		p.Set(5)

		if !strings.Contains(buf.String(), "Updated") {
			t.Error("expected updated message in output")
		}
	})

	t.Run("zero total", func(t *testing.T) {
		var buf bytes.Buffer
		p := NewProgressBar(0, "Progress",
			WithProgressBarWriter(&buf),
		)

		p.Set(5) // Should not panic
		if buf.Len() != 0 {
			t.Error("expected no output for zero total")
		}
	})
}

func TestStreamWriter(t *testing.T) {
	t.Run("write without prefix", func(t *testing.T) {
		var buf bytes.Buffer
		sw := NewStreamWriter(WithStreamWriterWriter(&buf))

		n, err := sw.Write([]byte("hello"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != 5 {
			t.Errorf("expected 5 bytes, got %d", n)
		}
		if buf.String() != "hello" {
			t.Errorf("expected 'hello', got %q", buf.String())
		}
	})

	t.Run("write with prefix", func(t *testing.T) {
		var buf bytes.Buffer
		sw := NewStreamWriter(
			WithStreamWriterWriter(&buf),
			WithStreamWriterPrefix("[LOG] "),
			WithStreamWriterColor(false),
		)

		sw.Write([]byte("line1\nline2\n"))

		output := buf.String()
		if !strings.Contains(output, "[LOG] line1") {
			t.Errorf("expected prefixed line1, got %q", output)
		}
		if !strings.Contains(output, "[LOG] line2") {
			t.Errorf("expected prefixed line2, got %q", output)
		}
	})

	t.Run("write line", func(t *testing.T) {
		var buf bytes.Buffer
		sw := NewStreamWriter(
			WithStreamWriterWriter(&buf),
			WithStreamWriterPrefix("[LOG] "),
			WithStreamWriterColor(false),
		)

		err := sw.WriteLine("message %d", 42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "[LOG] message 42") {
			t.Errorf("expected '[LOG] message 42', got %q", output)
		}
	})

	t.Run("write line without prefix", func(t *testing.T) {
		var buf bytes.Buffer
		sw := NewStreamWriter(WithStreamWriterWriter(&buf))

		err := sw.WriteLine("test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if buf.String() != "test\n" {
			t.Errorf("expected 'test\\n', got %q", buf.String())
		}
	})

	t.Run("with color", func(t *testing.T) {
		var buf bytes.Buffer
		sw := NewStreamWriter(
			WithStreamWriterWriter(&buf),
			WithStreamWriterPrefix("[LOG] "),
			WithStreamWriterColor(true),
			WithStreamWriterPrefixColor(ColorGreen),
		)

		sw.Write([]byte("test\n"))

		output := buf.String()
		if !strings.Contains(output, string(ColorGreen)) {
			t.Error("expected color code in output")
		}
	})
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected Format
		hasError bool
	}{
		{"table", FormatTable, false},
		{"TABLE", FormatTable, false},
		{"  table  ", FormatTable, false},
		{"json", FormatJSON, false},
		{"JSON", FormatJSON, false},
		{"text", FormatText, false},
		{"", FormatText, false},
		{"unknown", FormatText, true},
		{"xml", FormatText, true},
	}

	for _, tc := range tests {
		result, err := ParseFormat(tc.input)

		if tc.hasError {
			if err == nil {
				t.Errorf("ParseFormat(%q): expected error, got nil", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseFormat(%q): unexpected error: %v", tc.input, err)
			}
			if result != tc.expected {
				t.Errorf("ParseFormat(%q) = %v, expected %v", tc.input, result, tc.expected)
			}
		}
	}
}

func TestFormatter_ThreadSafety(t *testing.T) {
	var buf bytes.Buffer
	f := NewFormatter(WithWriter(&buf))

	done := make(chan bool, 10)

	// Run concurrent writes
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				f.Println("goroutine %d iteration %d", n, j)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify output contains expected content
	output := buf.String()
	if len(output) == 0 {
		t.Error("expected output from concurrent writes")
	}
}

func TestSpinner_ThreadSafety(t *testing.T) {
	var buf bytes.Buffer
	s := NewSpinner("Test",
		WithSpinnerWriter(&buf),
		WithSpinnerInterval(5*time.Millisecond),
	)

	done := make(chan bool, 5)

	s.Start()

	// Run concurrent message updates
	for i := 0; i < 5; i++ {
		go func(n int) {
			for j := 0; j < 20; j++ {
				s.UpdateMessage(strings.Repeat("x", n+1))
				time.Sleep(time.Millisecond)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	s.Stop()
}

func TestProgressBar_ThreadSafety(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgressBar(100, "Progress",
		WithProgressBarWriter(&buf),
	)

	done := make(chan bool, 10)

	// Run concurrent increments
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				p.Increment()
				time.Sleep(time.Millisecond)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if p.current != 100 {
		t.Errorf("expected progress to be 100, got %d", p.current)
	}
}

func TestStreamWriter_ThreadSafety(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(
		WithStreamWriterWriter(&buf),
		WithStreamWriterPrefix("[LOG] "),
	)

	done := make(chan bool, 10)

	// Run concurrent writes
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 50; j++ {
				sw.WriteLine("goroutine %d iteration %d", n, j)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("expected output from concurrent writes")
	}
}
