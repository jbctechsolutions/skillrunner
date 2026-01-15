package skill

import (
	"testing"
	"time"
)

func TestSourceType(t *testing.T) {
	t.Run("SourceType constants are defined", func(t *testing.T) {
		if SourceBuiltIn != "built-in" {
			t.Errorf("expected SourceBuiltIn to be 'built-in', got %q", SourceBuiltIn)
		}
		if SourceUser != "user" {
			t.Errorf("expected SourceUser to be 'user', got %q", SourceUser)
		}
		if SourceProject != "project" {
			t.Errorf("expected SourceProject to be 'project', got %q", SourceProject)
		}
	})

	t.Run("IsValid returns true for valid source types", func(t *testing.T) {
		validTypes := []SourceType{SourceBuiltIn, SourceUser, SourceProject}
		for _, st := range validTypes {
			if !st.IsValid() {
				t.Errorf("expected %q to be valid", st)
			}
		}
	})

	t.Run("IsValid returns false for invalid source types", func(t *testing.T) {
		invalid := SourceType("invalid")
		if invalid.IsValid() {
			t.Errorf("expected 'invalid' to be invalid")
		}
	})
}

func TestNewSkillSource(t *testing.T) {
	now := time.Now()

	t.Run("creates SkillSource with valid inputs", func(t *testing.T) {
		source, err := NewSkillSource("skill-1", "/path/to/skill.yaml", SourceUser, now)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if source == nil {
			t.Fatal("expected source to be non-nil")
		}
		if source.SkillID() != "skill-1" {
			t.Errorf("expected skillID 'skill-1', got %q", source.SkillID())
		}
		if source.FilePath() != "/path/to/skill.yaml" {
			t.Errorf("expected filePath '/path/to/skill.yaml', got %q", source.FilePath())
		}
		if source.Source() != SourceUser {
			t.Errorf("expected source SourceUser, got %q", source.Source())
		}
		if !source.LoadedAt().Equal(now) {
			t.Errorf("expected loadedAt %v, got %v", now, source.LoadedAt())
		}
	})

	t.Run("trims whitespace from skillID", func(t *testing.T) {
		source, err := NewSkillSource("  skill-1  ", "/path/to/skill.yaml", SourceUser, now)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if source.SkillID() != "skill-1" {
			t.Errorf("expected trimmed skillID 'skill-1', got %q", source.SkillID())
		}
	})

	t.Run("trims whitespace from filePath", func(t *testing.T) {
		source, err := NewSkillSource("skill-1", "  /path/to/skill.yaml  ", SourceUser, now)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if source.FilePath() != "/path/to/skill.yaml" {
			t.Errorf("expected trimmed filePath '/path/to/skill.yaml', got %q", source.FilePath())
		}
	})

	t.Run("returns error for empty skillID", func(t *testing.T) {
		_, err := NewSkillSource("", "/path/to/skill.yaml", SourceUser, now)
		if err == nil {
			t.Fatal("expected error for empty skillID")
		}
		if err != ErrSourceSkillIDRequired {
			t.Errorf("expected ErrSourceSkillIDRequired, got %v", err)
		}
	})

	t.Run("returns error for whitespace-only skillID", func(t *testing.T) {
		_, err := NewSkillSource("   ", "/path/to/skill.yaml", SourceUser, now)
		if err == nil {
			t.Fatal("expected error for whitespace-only skillID")
		}
	})

	t.Run("returns error for empty filePath", func(t *testing.T) {
		_, err := NewSkillSource("skill-1", "", SourceUser, now)
		if err == nil {
			t.Fatal("expected error for empty filePath")
		}
		if err != ErrSourceFilePathRequired {
			t.Errorf("expected ErrSourceFilePathRequired, got %v", err)
		}
	})

	t.Run("returns error for invalid source type", func(t *testing.T) {
		_, err := NewSkillSource("skill-1", "/path/to/skill.yaml", SourceType("invalid"), now)
		if err == nil {
			t.Fatal("expected error for invalid source type")
		}
		if err != ErrSourceTypeInvalid {
			t.Errorf("expected ErrSourceTypeInvalid, got %v", err)
		}
	})

	t.Run("returns error for zero time", func(t *testing.T) {
		_, err := NewSkillSource("skill-1", "/path/to/skill.yaml", SourceUser, time.Time{})
		if err == nil {
			t.Fatal("expected error for zero time")
		}
		if err != ErrSourceLoadedAtRequired {
			t.Errorf("expected ErrSourceLoadedAtRequired, got %v", err)
		}
	})
}

func TestSkillSourceImmutability(t *testing.T) {
	now := time.Now()
	source, _ := NewSkillSource("skill-1", "/path/to/skill.yaml", SourceUser, now)

	t.Run("SkillID returns same value", func(t *testing.T) {
		id1 := source.SkillID()
		id2 := source.SkillID()
		if id1 != id2 {
			t.Errorf("SkillID should return consistent value")
		}
	})

	t.Run("FilePath returns same value", func(t *testing.T) {
		path1 := source.FilePath()
		path2 := source.FilePath()
		if path1 != path2 {
			t.Errorf("FilePath should return consistent value")
		}
	})

	t.Run("Source returns same value", func(t *testing.T) {
		src1 := source.Source()
		src2 := source.Source()
		if src1 != src2 {
			t.Errorf("Source should return consistent value")
		}
	})

	t.Run("LoadedAt returns same value", func(t *testing.T) {
		time1 := source.LoadedAt()
		time2 := source.LoadedAt()
		if !time1.Equal(time2) {
			t.Errorf("LoadedAt should return consistent value")
		}
	})
}

func TestSkillSourcePriority(t *testing.T) {
	t.Run("Project has highest priority", func(t *testing.T) {
		if SourceProject.Priority() <= SourceUser.Priority() {
			t.Errorf("Project should have higher priority than User")
		}
		if SourceProject.Priority() <= SourceBuiltIn.Priority() {
			t.Errorf("Project should have higher priority than BuiltIn")
		}
	})

	t.Run("User has higher priority than BuiltIn", func(t *testing.T) {
		if SourceUser.Priority() <= SourceBuiltIn.Priority() {
			t.Errorf("User should have higher priority than BuiltIn")
		}
	})

	t.Run("BuiltIn has lowest priority", func(t *testing.T) {
		if SourceBuiltIn.Priority() >= SourceUser.Priority() {
			t.Errorf("BuiltIn should have lower priority than User")
		}
		if SourceBuiltIn.Priority() >= SourceProject.Priority() {
			t.Errorf("BuiltIn should have lower priority than Project")
		}
	})
}
