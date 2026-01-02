package cache

import (
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
)

func TestFingerprint(t *testing.T) {
	tests := []struct {
		name     string
		req1     ports.CompletionRequest
		req2     ports.CompletionRequest
		wantSame bool
	}{
		{
			name: "identical requests have same fingerprint",
			req1: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "user", Content: "Hello"},
				},
				Temperature: 0.7,
				MaxTokens:   100,
			},
			req2: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "user", Content: "Hello"},
				},
				Temperature: 0.7,
				MaxTokens:   100,
			},
			wantSame: true,
		},
		{
			name: "different models have different fingerprints",
			req1: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			req2: ports.CompletionRequest{
				ModelID: "gpt-3.5-turbo",
				Messages: []ports.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			wantSame: false,
		},
		{
			name: "different messages have different fingerprints",
			req1: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			req2: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "user", Content: "Goodbye"},
				},
			},
			wantSame: false,
		},
		{
			name: "different temperature has different fingerprints",
			req1: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "user", Content: "Hello"},
				},
				Temperature: 0.7,
			},
			req2: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "user", Content: "Hello"},
				},
				Temperature: 0.9,
			},
			wantSame: false,
		},
		{
			name: "different max_tokens has different fingerprints",
			req1: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "user", Content: "Hello"},
				},
				MaxTokens: 100,
			},
			req2: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "user", Content: "Hello"},
				},
				MaxTokens: 200,
			},
			wantSame: false,
		},
		{
			name: "zero vs non-zero temperature",
			req1: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "user", Content: "Hello"},
				},
				Temperature: 0,
			},
			req2: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "user", Content: "Hello"},
				},
				Temperature: 0.7,
			},
			wantSame: false,
		},
		{
			name: "multiple messages in same order",
			req1: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "system", Content: "You are helpful"},
					{Role: "user", Content: "Hello"},
				},
			},
			req2: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "system", Content: "You are helpful"},
					{Role: "user", Content: "Hello"},
				},
			},
			wantSame: true,
		},
		{
			name: "multiple messages in different order",
			req1: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "system", Content: "You are helpful"},
					{Role: "user", Content: "Hello"},
				},
			},
			req2: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "user", Content: "Hello"},
					{Role: "system", Content: "You are helpful"},
				},
			},
			wantSame: false,
		},
		{
			name: "system prompt affects fingerprint",
			req1: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "user", Content: "Hello"},
				},
				SystemPrompt: "Be helpful",
			},
			req2: ports.CompletionRequest{
				ModelID: "gpt-4",
				Messages: []ports.Message{
					{Role: "user", Content: "Hello"},
				},
				SystemPrompt: "Be concise",
			},
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fp1 := Fingerprint(tt.req1)
			fp2 := Fingerprint(tt.req2)

			if tt.wantSame && fp1 != fp2 {
				t.Errorf("expected same fingerprints, got %s and %s", fp1, fp2)
			}
			if !tt.wantSame && fp1 == fp2 {
				t.Errorf("expected different fingerprints, got same: %s", fp1)
			}
		})
	}
}

func TestFingerprint_Deterministic(t *testing.T) {
	req := ports.CompletionRequest{
		ModelID: "gpt-4",
		Messages: []ports.Message{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	// Generate fingerprint multiple times
	fingerprints := make([]string, 10)
	for i := range fingerprints {
		fingerprints[i] = Fingerprint(req)
	}

	// All should be the same
	for i := 1; i < len(fingerprints); i++ {
		if fingerprints[i] != fingerprints[0] {
			t.Errorf("Fingerprint is not deterministic: got %s and %s", fingerprints[0], fingerprints[i])
		}
	}
}

func TestFingerprint_Length(t *testing.T) {
	req := ports.CompletionRequest{
		ModelID: "gpt-4",
		Messages: []ports.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	fp := Fingerprint(req)

	// SHA256 hex-encoded is 64 characters
	if len(fp) != 64 {
		t.Errorf("Fingerprint length = %d, want 64", len(fp))
	}
}
