package model_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/citizenofai/agent-sdk-go/pkg/model"
	"github.com/citizenofai/agent-sdk-go/pkg/model/providers/openai"
	"github.com/stretchr/testify/assert"
)

func TestOpenAIProvider(t *testing.T) {
	t.Run("NewProvider", func(t *testing.T) {
		provider := openai.NewProvider("test-key")
		assert.NotNil(t, provider)
		assert.Equal(t, "test-key", provider.APIKey)
		assert.Equal(t, openai.DefaultMaxRetries, provider.MaxRetries)
	})

	t.Run("WithAPIKey", func(t *testing.T) {
		provider := openai.NewProvider("initial-key")
		provider = provider.WithAPIKey("new-key")
		assert.Equal(t, "new-key", provider.APIKey)
	})

	t.Run("WithOrganization", func(t *testing.T) {
		provider := openai.NewProvider("test-key")
		provider = provider.WithOrganization("test-org")
		assert.Equal(t, "test-org", provider.Organization)
	})

	t.Run("WithDefaultModel", func(t *testing.T) {
		provider := openai.NewProvider("test-key")
		provider = provider.WithDefaultModel("gpt-4")
		assert.Equal(t, "gpt-4", provider.DefaultModel)
	})

	t.Run("GetModel", func(t *testing.T) {
		provider := openai.NewProvider("test-key")
		provider.WithDefaultModel("gpt-3.5-turbo")

		openaiModel, err := provider.GetModel("gpt-4")
		assert.NoError(t, err)
		assert.NotNil(t, openaiModel)
		assert.Equal(t, "gpt-4", openaiModel.(*openai.Model).ModelName)

		// Test with default model
		openaiModel, err = provider.GetModel("")
		assert.NoError(t, err)
		assert.NotNil(t, openaiModel)
		assert.Equal(t, "gpt-3.5-turbo", openaiModel.(*openai.Model).ModelName)
	})
}

func TestOpenAIModel(t *testing.T) {
	t.Run("GetResponse_Success", func(t *testing.T) {
		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "/chat/completions", r.URL.Path)
			assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			// Return a mock response
			response := map[string]interface{}{
				"id":      "test-id",
				"object":  "chat.completion",
				"created": time.Now().Unix(),
				"model":   "gpt-3.5-turbo",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role":    "assistant",
							"content": "Test response",
						},
						"finish_reason": "stop",
					},
				},
				"usage": map[string]interface{}{
					"prompt_tokens":     10,
					"completion_tokens": 5,
					"total_tokens":      15,
				},
			}
			err := json.NewEncoder(w).Encode(response)
			assert.NoError(t, err)
		}))
		defer server.Close()

		// Create provider and model
		provider := openai.NewProvider("test-key")
		provider.SetBaseURL(server.URL)
		openaiModel, err := provider.GetModel("gpt-3.5-turbo")
		assert.NoError(t, err)

		// Test request
		request := &model.Request{
			Input:              "Test input",
			SystemInstructions: "Test system instructions",
		}

		response, err := openaiModel.GetResponse(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "Test response", response.Content)
		assert.Equal(t, 15, response.Usage.TotalTokens)
	})

	t.Run("GetResponse_WithTools", func(t *testing.T) {
		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Parse request body to verify tools
			var requestBody map[string]interface{}
			var err error
			err = json.NewDecoder(r.Body).Decode(&requestBody)
			assert.NoError(t, err)

			// Verify tools are included
			tools, ok := requestBody["tools"].([]interface{})
			assert.True(t, ok)
			assert.Len(t, tools, 1)

			// Return a mock response with tool calls
			response := map[string]interface{}{
				"id":      "test-id",
				"object":  "chat.completion",
				"created": time.Now().Unix(),
				"model":   "gpt-3.5-turbo",
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"message": map[string]interface{}{
							"role": "assistant",
							"tool_calls": []map[string]interface{}{
								{
									"id":   "call_123",
									"type": "function",
									"function": map[string]interface{}{
										"name":      "test_tool",
										"arguments": `{"param1": "value1"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
				"usage": map[string]interface{}{
					"total_tokens": 15,
				},
			}
			err = json.NewEncoder(w).Encode(response)
			assert.NoError(t, err)
		}))
		defer server.Close()

		// Create provider and model
		provider := openai.NewProvider("test-key")
		provider.SetBaseURL(server.URL)
		openaiModel, err := provider.GetModel("gpt-3.5-turbo")
		assert.NoError(t, err)

		// Create a test tool
		testTool := map[string]interface{}{
			"name":        "test_tool",
			"description": "A test tool",
			"parameters": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"param1": map[string]interface{}{
						"type": "string",
					},
				},
			},
		}

		// Test request with tool
		request := &model.Request{
			Input: "Test input",
			Tools: []interface{}{testTool},
		}

		response, err := openaiModel.GetResponse(context.Background(), request)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.Len(t, response.ToolCalls, 1)
		assert.Equal(t, "test_tool", response.ToolCalls[0].Name)
		assert.Equal(t, "value1", response.ToolCalls[0].Parameters["param1"])
	})

	t.Run("GetResponse_Error", func(t *testing.T) {
		// Create a test server that returns an error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			err := json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Rate limit exceeded",
					"type":    "rate_limit_error",
				},
			})
			assert.NoError(t, err)
		}))
		defer server.Close()

		// Create provider and model
		provider := openai.NewProvider("test-key")
		provider.SetBaseURL(server.URL)
		provider.WithRetryConfig(1, time.Millisecond) // Set low retry count for test
		openaiModel, err := provider.GetModel("gpt-3.5-turbo")
		assert.NoError(t, err)

		// Test request
		request := &model.Request{
			Input: "Test input",
		}

		response, err := openaiModel.GetResponse(context.Background(), request)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "Rate limit exceeded")
	})

	t.Run("StreamResponse", func(t *testing.T) {
		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify streaming is requested
			var requestBody map[string]interface{}
			var err error
			err = json.NewDecoder(r.Body).Decode(&requestBody)
			assert.NoError(t, err)
			assert.True(t, requestBody["stream"].(bool))

			// Write SSE response
			flusher, ok := w.(http.Flusher)
			assert.True(t, ok)

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// Send content chunks
			events := []string{
				`{"choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}`,
				`{"choices":[{"delta":{"content":" world"},"finish_reason":null}]}`,
				`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
			}

			for _, event := range events {
				_, err = w.Write([]byte("data: " + event + "\n\n"))
				assert.NoError(t, err)
				flusher.Flush()
				time.Sleep(10 * time.Millisecond)
			}
			_, err = w.Write([]byte("data: [DONE]\n\n"))
			assert.NoError(t, err)
		}))
		defer server.Close()

		// Create provider and model
		provider := openai.NewProvider("test-key")
		provider.SetBaseURL(server.URL)
		openaiModel, err := provider.GetModel("gpt-3.5-turbo")
		assert.NoError(t, err)

		// Test streaming request
		request := &model.Request{
			Input: "Test input",
		}

		stream, err := openaiModel.StreamResponse(context.Background(), request)
		assert.NoError(t, err)

		var content string
		for event := range stream {
			if event.Error != nil {
				t.Fatalf("Stream error: %v", event.Error)
			}
			if event.Type == model.StreamEventTypeContent {
				content += event.Content
			}
		}

		assert.Equal(t, "Hello world", content)
	})
}

func TestOpenAI_ConstructRequest_WithOutputSchema(t *testing.T) {
	outputSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type": "string",
			},
			"age": map[string]interface{}{
				"type": "integer",
			},
		},
		"required":             []string{"name", "age"},
		"additionalProperties": false,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse request body to verify response_format
		var requestBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		assert.NoError(t, err)

		// Verify response_format is set with json_schema type
		responseFormat, ok := requestBody["response_format"].(map[string]interface{})
		assert.True(t, ok, "response_format should be present in request")
		assert.Equal(t, "json_schema", responseFormat["type"])

		jsonSchema, ok := responseFormat["json_schema"].(map[string]interface{})
		assert.True(t, ok, "json_schema should be present in response_format")
		assert.Equal(t, "structured_output", jsonSchema["name"])
		assert.Equal(t, true, jsonSchema["strict"])

		schema, ok := jsonSchema["schema"].(map[string]interface{})
		assert.True(t, ok, "schema should be present in json_schema")
		assert.Equal(t, "object", schema["type"])

		properties, ok := schema["properties"].(map[string]interface{})
		assert.True(t, ok, "properties should be present in schema")
		assert.Contains(t, properties, "name")
		assert.Contains(t, properties, "age")

		// Return a mock structured response
		response := map[string]interface{}{
			"id":      "test-id",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": `{"name":"Alice","age":30}`,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     20,
				"completion_tokens": 10,
				"total_tokens":      30,
			},
		}
		err = json.NewEncoder(w).Encode(response)
		assert.NoError(t, err)
	}))
	defer server.Close()

	provider := openai.NewProvider("test-key")
	provider.SetBaseURL(server.URL)
	openaiModel, err := provider.GetModel("gpt-4")
	assert.NoError(t, err)

	request := &model.Request{
		Input:        "Return a person object with name and age.",
		OutputSchema: outputSchema,
	}

	resp, err := openaiModel.GetResponse(context.Background(), request)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, `{"name":"Alice","age":30}`, resp.Content)
	assert.Equal(t, 30, resp.Usage.TotalTokens)
}

func TestOpenAI_ConstructRequest_WithoutOutputSchema(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		assert.NoError(t, err)

		// Verify response_format is NOT set when OutputSchema is nil
		_, hasResponseFormat := requestBody["response_format"]
		assert.False(t, hasResponseFormat, "response_format should not be present when OutputSchema is nil")

		response := map[string]interface{}{
			"id":      "test-id",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "gpt-4",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Plain text response",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{"total_tokens": 10},
		}
		err = json.NewEncoder(w).Encode(response)
		assert.NoError(t, err)
	}))
	defer server.Close()

	provider := openai.NewProvider("test-key")
	provider.SetBaseURL(server.URL)
	openaiModel, err := provider.GetModel("gpt-4")
	assert.NoError(t, err)

	request := &model.Request{
		Input: "Hello",
	}

	resp, err := openaiModel.GetResponse(context.Background(), request)
	assert.NoError(t, err)
	assert.Equal(t, "Plain text response", resp.Content)
}

func TestOpenAIRateLimiting(t *testing.T) {
	t.Run("RequestRateLimit", func(t *testing.T) {
		provider := openai.NewProvider("test-key")
		provider.WithRateLimit(2, 1000)

		// First two requests should not block
		start := time.Now()
		provider.WaitForRateLimit()
		provider.WaitForRateLimit()
		assert.Less(t, time.Since(start), 100*time.Millisecond)

		// Third request should block
		start = time.Now()
		provider.WaitForRateLimit()
		assert.Greater(t, time.Since(start), 100*time.Millisecond)
	})

	t.Run("UpdateTokenCount", func(t *testing.T) {
		provider := openai.NewProvider("test-key")

		// Update token count
		provider.UpdateTokenCount(500)

		// We can't directly verify the token count as it's an internal field,
		// but we can test the behavior: if we've used a lot of tokens,
		// the rate limiter should block
		provider.WithRateLimit(100, 400) // 400 tokens per minute

		// Should block because we've already used 500 tokens
		start := time.Now()
		provider.WaitForRateLimit()
		assert.Greater(t, time.Since(start), 50*time.Millisecond)
	})

	t.Run("ResetRateLimiter", func(t *testing.T) {
		provider := openai.NewProvider("test-key")
		provider.WithRateLimit(1, 100) // Very restrictive limits

		// Update counts
		provider.UpdateTokenCount(500)
		provider.WaitForRateLimit() // This should block

		// Reset
		provider.ResetRateLimiter()

		// After reset, we should be able to make requests without blocking
		start := time.Now()
		provider.WaitForRateLimit()
		assert.Less(t, time.Since(start), 50*time.Millisecond)
	})
}
