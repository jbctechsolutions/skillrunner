package converter

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jbctechsolutions/skillrunner/internal/types"
	"gopkg.in/yaml.v3"
)

// MarketplaceSkillMetadata represents the YAML frontmatter from SKILL.md/AGENT.md files
type MarketplaceSkillMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Tools       string `yaml:"tools"`
	Model       string `yaml:"model"`
	Version     string `yaml:"version"`
	Author      string `yaml:"author"`
	Source      string `yaml:"source"`
	LastUpdated string `yaml:"last_updated"`
}

// FromMarkdown converts a marketplace SKILL.md file to Skillrunner orchestrated format
// Accepts either a file path (SKILL.md or AGENT.md) or a directory path containing SKILL.md/AGENT.md
func FromMarkdown(path string) (*types.OrchestratedSkill, error) {
	// Check if path is a directory or file
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}

	var mdFile string
	if info.IsDir() {
		// It's a directory - find SKILL.md or AGENT.md within it
		skillMd := filepath.Join(path, "SKILL.md")
		agentMd := filepath.Join(path, "AGENT.md")

		if _, err := os.Stat(skillMd); err == nil {
			mdFile = skillMd
		} else if _, err := os.Stat(agentMd); err == nil {
			mdFile = agentMd
		} else {
			// Look for any .md file in the directory
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil, fmt.Errorf("read directory: %w", err)
			}

			for _, entry := range entries {
				if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
					mdFile = filepath.Join(path, entry.Name())
					break
				}
			}

			if mdFile == "" {
				return nil, fmt.Errorf("no SKILL.md, AGENT.md, or .md file found in directory: %s", path)
			}
		}
	} else {
		// It's a file - use it directly
		mdFile = path
	}

	// Parse the markdown file
	metadata, prompt, err := parseMarkdownSkill(mdFile)
	if err != nil {
		return nil, fmt.Errorf("parse marketplace skill: %w", err)
	}

	// Create orchestrated skill
	orchestrated := &types.OrchestratedSkill{
		Name:        metadata.Name,
		Version:     metadata.Version,
		Description: metadata.Description,
		Type:        types.SkillTypeSimple,
		Phases: []types.Phase{
			{
				ID:             "main",
				Name:           "Execute",
				TaskType:       types.TaskTypeGeneration,
				PromptTemplate: fmt.Sprintf("%s\n\nInput: {{request}}", prompt),
				OutputKey:      "result",
				Routing: &types.PhaseRouting{
					PreferredModels: []string{normalizeModelName(metadata.Model)},
				},
			},
		},
	}

	return orchestrated, nil
}

// parseMarkdownSkill parses a SKILL.md file with YAML frontmatter
func parseMarkdownSkill(path string) (*MarketplaceSkillMetadata, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var yamlLines []string
	var promptLines []string
	inFrontmatter := false
	frontmatterComplete := false

	for scanner.Scan() {
		line := scanner.Text()

		// Check for frontmatter delimiters
		if strings.TrimSpace(line) == "---" {
			if !inFrontmatter {
				inFrontmatter = true
				continue
			} else {
				frontmatterComplete = true
				continue
			}
		}

		// Collect frontmatter lines
		if inFrontmatter && !frontmatterComplete {
			yamlLines = append(yamlLines, line)
			continue
		}

		// Collect prompt lines (everything after frontmatter)
		if frontmatterComplete {
			promptLines = append(promptLines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("scan file: %w", err)
	}

	// Parse frontmatter
	var metadata MarketplaceSkillMetadata
	yamlContent := strings.Join(yamlLines, "\n")
	if err := yaml.Unmarshal([]byte(yamlContent), &metadata); err != nil {
		return nil, "", fmt.Errorf("parse frontmatter: %w", err)
	}

	// Join prompt lines
	prompt := strings.TrimSpace(strings.Join(promptLines, "\n"))

	return &metadata, prompt, nil
}

// normalizeModelName normalizes model names from marketplace format to Skillrunner format
func normalizeModelName(modelName string) string {
	// Map common marketplace model names to full provider/model format
	modelMap := map[string]string{
		"sonnet":  "anthropic/claude-3-5-sonnet-20241022",
		"opus":    "anthropic/claude-3-opus-20240229",
		"haiku":   "anthropic/claude-3-haiku-20240307",
		"gpt-4":   "openai/gpt-4",
		"gpt-4o":  "openai/gpt-4o",
		"gpt-3.5": "openai/gpt-3.5-turbo",
	}

	if fullName, ok := modelMap[strings.ToLower(modelName)]; ok {
		return fullName
	}

	// If already in provider/model format, return as-is
	if strings.Contains(modelName, "/") {
		return modelName
	}

	// Otherwise, assume it's an Ollama model
	return "ollama/" + modelName
}

// MarshalYAMLWithMultiLineStrings marshals a value to YAML with multi-line strings
// formatted as literal block scalars (|) for better readability
func MarshalYAMLWithMultiLineStrings(v interface{}) ([]byte, error) {
	// Marshal normally first
	data, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}

	// Post-process to convert multi-line escaped strings to literal blocks
	return formatMultiLineYAML(data), nil
}

// formatMultiLineYAML converts escaped newline strings to YAML literal block scalars
func formatMultiLineYAML(data []byte) []byte {
	lines := strings.Split(string(data), "\n")
	var result []string

	for _, line := range lines {
		// Check if this line starts a multi-line string (contains ": \" and has \n in the value)
		if strings.Contains(line, ": \"") && strings.Contains(line, "\\n") {
			// Extract key and check if value contains newlines
			parts := strings.SplitN(line, ": \"", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := parts[1]

				// Check if this is a multi-line string (has multiple \n or is very long)
				if strings.Count(value, "\\n") > 1 || (strings.Contains(value, "\\n") && len(value) > 100) {
					// Extract indentation
					indent := strings.Repeat(" ", len(line)-len(strings.TrimLeft(line, " ")))

					// Start literal block
					result = append(result, indent+key+": |")

					// Extract and format the content
					// Remove the closing quote if present
					value = strings.TrimSuffix(value, "\"")
					// Replace \n with actual newlines
					content := strings.ReplaceAll(value, "\\n", "\n")
					// Split into lines and add with proper indentation
					contentLines := strings.Split(content, "\n")
					for _, cl := range contentLines {
						result = append(result, indent+"  "+cl)
					}
					continue
				}
			}
		}

		result = append(result, line)
	}

	return []byte(strings.Join(result, "\n"))
}
