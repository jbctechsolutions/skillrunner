package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/types"
)

// HuggingFaceClient provides access to HuggingFace marketplace via MCP
type HuggingFaceClient struct {
	mcpEndpoint string
	httpClient  *http.Client
	cache       *SkillCache
	useMock     bool // Use mock data when MCP server unavailable
}

// NewHuggingFaceClient creates a new HuggingFace marketplace client
func NewHuggingFaceClient(mcpEndpoint string) *HuggingFaceClient {
	if mcpEndpoint == "" {
		mcpEndpoint = "http://localhost:3000" // Default MCP endpoint
	}

	return &HuggingFaceClient{
		mcpEndpoint: mcpEndpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:   NewSkillCache(),
		useMock: false, // Will be set to true on connection failure
	}
}

// SearchSkills searches for skills in the HuggingFace marketplace
func (c *HuggingFaceClient) SearchSkills(ctx context.Context, query string) ([]*types.MarketplaceSkill, error) {
	// Check cache first
	if cached := c.cache.Search(query); len(cached) > 0 {
		return cached, nil
	}

	// Try MCP server first
	if !c.useMock {
		req := MCPRequest{
			Tool: "space_search",
			Params: map[string]interface{}{
				"query": query,
				"limit": 20,
			},
		}

		resp, err := c.callMCP(ctx, req)
		if err != nil {
			// Connection failed, switch to mock mode
			fmt.Printf("Note: MCP server unavailable, using mock data\n")
			c.useMock = true
		} else {
			skills, err := c.parseSkillResults(resp)
			if err != nil {
				return nil, err
			}

			// Cache results
			c.cache.AddSearchResults(query, skills)
			return skills, nil
		}
	}

	// Use mock data
	skills := c.getMockSearchResults(query)
	c.cache.AddSearchResults(query, skills)
	return skills, nil
}

// GetSkill retrieves detailed information about a specific skill
func (c *HuggingFaceClient) GetSkill(ctx context.Context, skillID string) (*types.MarketplaceSkill, error) {
	// Check cache first
	if cached := c.cache.Get(skillID); cached != nil {
		return cached, nil
	}

	// Use HuggingFace MCP to get skill details
	req := MCPRequest{
		Tool: "get_space",
		Params: map[string]interface{}{
			"space_id": skillID,
		},
	}

	resp, err := c.callMCP(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get skill %s: %w", skillID, err)
	}

	skill, err := c.parseSkillDetail(resp)
	if err != nil {
		return nil, err
	}

	// Cache the skill
	c.cache.Add(skill)

	return skill, nil
}

// ExecuteSkill executes a marketplace skill using HuggingFace MCP dynamic_space tool
func (c *HuggingFaceClient) ExecuteSkill(ctx context.Context, skillID string, params map[string]interface{}) (*types.MarketplaceResult, error) {
	// Use dynamic_space tool to execute the skill
	req := MCPRequest{
		Tool: "dynamic_space",
		Params: map[string]interface{}{
			"space_id":   skillID,
			"parameters": params,
		},
	}

	resp, err := c.callMCP(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("skill execution failed: %w", err)
	}

	result, err := c.parseExecutionResult(resp)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ListCategories returns available skill categories
func (c *HuggingFaceClient) ListCategories(ctx context.Context) ([]string, error) {
	// HuggingFace categories
	categories := []string{
		"computer-vision",
		"natural-language-processing",
		"audio",
		"tabular",
		"reinforcement-learning",
		"other",
	}
	return categories, nil
}

// MCPRequest represents a request to the MCP server
type MCPRequest struct {
	Tool   string                 `json:"tool"`
	Params map[string]interface{} `json:"params"`
}

// MCPResponse represents a response from the MCP server
type MCPResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data"`
	Error   string                 `json:"error,omitempty"`
}

// callMCP makes a call to the MCP server
func (c *HuggingFaceClient) callMCP(ctx context.Context, req MCPRequest) (*MCPResponse, error) {
	// Marshal request (for future use when implementing actual MCP protocol)
	_, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.mcpEndpoint+"/mcp/call", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Body = http.NoBody // Simplified for now

	// Make request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP server returned status %d", httpResp.StatusCode)
	}

	// Parse response
	var resp MCPResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("MCP error: %s", resp.Error)
	}

	return &resp, nil
}

// parseSkillResults parses search results into MarketplaceSkill objects
func (c *HuggingFaceClient) parseSkillResults(resp *MCPResponse) ([]*types.MarketplaceSkill, error) {
	var skills []*types.MarketplaceSkill

	// Parse the data field
	if spaces, ok := resp.Data["spaces"].([]interface{}); ok {
		for _, space := range spaces {
			if spaceMap, ok := space.(map[string]interface{}); ok {
				skill := &types.MarketplaceSkill{
					ID:          getStringField(spaceMap, "id"),
					Name:        getStringField(spaceMap, "name"),
					Description: getStringField(spaceMap, "description"),
					Author:      getStringField(spaceMap, "author"),
					Tags:        getStringSliceField(spaceMap, "tags"),
					SDK:         getStringField(spaceMap, "sdk"),
					Likes:       getIntField(spaceMap, "likes"),
					CreatedAt:   getStringField(spaceMap, "created_at"),
					UpdatedAt:   getStringField(spaceMap, "updated_at"),
				}
				skills = append(skills, skill)
			}
		}
	}

	return skills, nil
}

// parseSkillDetail parses detailed skill information
func (c *HuggingFaceClient) parseSkillDetail(resp *MCPResponse) (*types.MarketplaceSkill, error) {
	if space, ok := resp.Data["space"].(map[string]interface{}); ok {
		skill := &types.MarketplaceSkill{
			ID:          getStringField(space, "id"),
			Name:        getStringField(space, "name"),
			Description: getStringField(space, "description"),
			Author:      getStringField(space, "author"),
			Tags:        getStringSliceField(space, "tags"),
			SDK:         getStringField(space, "sdk"),
			Likes:       getIntField(space, "likes"),
			CreatedAt:   getStringField(space, "created_at"),
			UpdatedAt:   getStringField(space, "updated_at"),
			URL:         getStringField(space, "url"),
			Runtime:     getStringField(space, "runtime"),
		}
		return skill, nil
	}

	return nil, fmt.Errorf("invalid skill detail response")
}

// parseExecutionResult parses the execution result
func (c *HuggingFaceClient) parseExecutionResult(resp *MCPResponse) (*types.MarketplaceResult, error) {
	result := &types.MarketplaceResult{
		Success: resp.Success,
		Output:  resp.Data,
	}

	if error, ok := resp.Data["error"].(string); ok {
		result.Error = error
	}

	return result, nil
}

// Helper functions
func getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getIntField(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getStringSliceField(m map[string]interface{}, key string) []string {
	if arr, ok := m[key].([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}
	return nil
}

// Mock data functions for when MCP server is unavailable

func (c *HuggingFaceClient) getMockSearchResults(query string) []*types.MarketplaceSkill {
	// Return mock skills that match common queries
	allSkills := []*types.MarketplaceSkill{
		{
			ID:          "hf:paddleocr",
			Name:        "PaddleOCR",
			Description: "High-accuracy OCR for text extraction from images",
			Author:      "PaddlePaddle",
			Tags:        []string{"ocr", "text-extraction", "computer-vision"},
			SDK:         "gradio",
			Likes:       1250,
			CreatedAt:   "2023-01-15",
			UpdatedAt:   "2024-11-10",
			Runtime:     "python",
		},
		{
			ID:          "hf:tesseract-ocr",
			Name:        "Tesseract OCR",
			Description: "Open-source OCR engine for text recognition",
			Author:      "tesseract-team",
			Tags:        []string{"ocr", "text-recognition", "open-source"},
			SDK:         "gradio",
			Likes:       890,
			CreatedAt:   "2023-03-20",
			UpdatedAt:   "2024-10-25",
			Runtime:     "python",
		},
		{
			ID:          "hf:flux-image-generator",
			Name:        "FLUX Image Generator",
			Description: "State-of-the-art text-to-image generation",
			Author:      "black-forest-labs",
			Tags:        []string{"image-generation", "text-to-image", "ai-art"},
			SDK:         "gradio",
			Likes:       3420,
			CreatedAt:   "2024-08-01",
			UpdatedAt:   "2024-11-15",
			Runtime:     "python",
		},
		{
			ID:          "hf:stable-diffusion-xl",
			Name:        "Stable Diffusion XL",
			Description: "High-quality image generation from text prompts",
			Author:      "stability-ai",
			Tags:        []string{"image-generation", "diffusion", "ai-art"},
			SDK:         "gradio",
			Likes:       5280,
			CreatedAt:   "2023-07-10",
			UpdatedAt:   "2024-11-01",
			Runtime:     "python",
		},
		{
			ID:          "hf:whisper-large-v3",
			Name:        "Whisper Large V3",
			Description: "Advanced speech recognition and transcription",
			Author:      "openai",
			Tags:        []string{"speech-to-text", "transcription", "audio"},
			SDK:         "gradio",
			Likes:       2150,
			CreatedAt:   "2023-11-05",
			UpdatedAt:   "2024-11-12",
			Runtime:     "python",
		},
		{
			ID:          "hf:musicgen",
			Name:        "MusicGen",
			Description: "Generate music from text descriptions",
			Author:      "facebook",
			Tags:        []string{"music-generation", "audio", "ai-music"},
			SDK:         "gradio",
			Likes:       1680,
			CreatedAt:   "2023-06-15",
			UpdatedAt:   "2024-10-20",
			Runtime:     "python",
		},
	}

	// Filter skills based on query
	if query == "" {
		return allSkills
	}

	var filtered []*types.MarketplaceSkill
	queryLower := strings.ToLower(query)
	for _, skill := range allSkills {
		// Search in name, description, and tags
		if strings.Contains(strings.ToLower(skill.Name), queryLower) ||
			strings.Contains(strings.ToLower(skill.Description), queryLower) {
			filtered = append(filtered, skill)
			continue
		}

		for _, tag := range skill.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				filtered = append(filtered, skill)
				break
			}
		}
	}

	return filtered
}
