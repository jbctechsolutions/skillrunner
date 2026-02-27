package compression_test

import (
	"strings"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/application/compression"
)

func TestResult_RatioRange(t *testing.T) {
	c := compression.New(compression.ModeConservative)
	r := c.Compress("hello  world\n\n\n\n\nfoo")
	if r.Ratio < 0 || r.Ratio > 1 {
		t.Errorf("ratio %.3f out of [0,1]", r.Ratio)
	}
}

func TestModeOff_NoChange(t *testing.T) {
	input := "hello   world\n\n\n\n\nduplicate\nduplicate\nduplicate"
	c := compression.New(compression.ModeOff)
	r := c.Compress(input)
	if r.Compressed != input {
		t.Error("ModeOff should not modify input")
	}
	if r.Ratio != 0 {
		t.Errorf("ModeOff ratio should be 0, got %.3f", r.Ratio)
	}
}

func TestEmptyInput(t *testing.T) {
	c := compression.New(compression.ModeAggressive)
	r := c.Compress("")
	if r.Compressed != "" {
		t.Error("empty input should remain empty")
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	input := "foo   bar\tbaz   qux"
	c := compression.New(compression.ModeConservative)
	r := c.Compress(input)
	if strings.Contains(r.Compressed, "   ") {
		t.Errorf("multiple spaces not collapsed: %q", r.Compressed)
	}
}

func TestDeduplicateBlankLines(t *testing.T) {
	input := "line1\n\n\n\n\nline2"
	c := compression.New(compression.ModeConservative)
	r := c.Compress(input)
	if strings.Contains(r.Compressed, "\n\n\n\n") {
		t.Errorf("4+ blank lines not collapsed: %q", r.Compressed)
	}
}

func TestDeduplicateRepeatedBlocks_Aggressive(t *testing.T) {
	input := "x\nx\nx\nx\nx" // 5 identical lines
	c := compression.New(compression.ModeAggressive)
	r := c.Compress(input)
	lineCount := len(strings.Split(strings.TrimSpace(r.Compressed), "\n"))
	if lineCount > 2 {
		t.Errorf("expected at most 2 repeated lines, got %d:\n%s", lineCount, r.Compressed)
	}
}

func TestDeduplicateRepeatedBlocks_Conservative(t *testing.T) {
	// Conservative should NOT deduplicate repeated lines
	input := "x\nx\nx"
	c := compression.New(compression.ModeConservative)
	r := c.Compress(input)
	lineCount := len(strings.Split(strings.TrimSpace(r.Compressed), "\n"))
	if lineCount < 3 {
		t.Errorf("conservative should preserve repeated lines, got %d lines", lineCount)
	}
}

func TestStripLineComments_Aggressive(t *testing.T) {
	input := "code line\n// this is a comment\n# another comment\nmore code"
	c := compression.New(compression.ModeAggressive)
	r := c.Compress(input)
	if strings.Contains(r.Compressed, "// this is a comment") {
		t.Error("aggressive mode should strip // comment lines")
	}
	if strings.Contains(r.Compressed, "# another comment") {
		t.Error("aggressive mode should strip # comment lines")
	}
	if !strings.Contains(r.Compressed, "code line") {
		t.Error("non-comment lines should be preserved")
	}
}

func TestStripLineComments_Conservative(t *testing.T) {
	input := "code\n// comment\nmore code"
	c := compression.New(compression.ModeConservative)
	r := c.Compress(input)
	if !strings.Contains(r.Compressed, "// comment") {
		t.Error("conservative mode should preserve comment lines")
	}
}

func TestNewFromProfile(t *testing.T) {
	cases := []struct {
		profile string
		wantAgg bool // whether aggressive strategies apply
	}{
		{"cheap", true},
		{"balanced", false},
		{"premium", false},
	}

	commentLine := "\n// comment line\n"
	for _, tc := range cases {
		c := compression.NewFromProfile(tc.profile)
		r := c.Compress("code" + commentLine + "more code")
		hasComment := strings.Contains(r.Compressed, "// comment line")
		if tc.wantAgg && hasComment {
			t.Errorf("profile %q should strip comments (aggressive)", tc.profile)
		}
		if !tc.wantAgg && !hasComment {
			t.Errorf("profile %q should preserve comments (conservative)", tc.profile)
		}
	}
}

func TestAggressive_ReducesMoreThanConservative(t *testing.T) {
	// A text with lots of comments and repeated blocks
	input := strings.Repeat("// repeated comment\n", 20) + "actual content\n"
	agg := compression.New(compression.ModeAggressive).Compress(input)
	con := compression.New(compression.ModeConservative).Compress(input)
	if agg.Ratio < con.Ratio {
		t.Errorf("aggressive (%.2f) should reduce more than conservative (%.2f)", agg.Ratio, con.Ratio)
	}
}
