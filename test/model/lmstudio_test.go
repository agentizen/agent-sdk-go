package model_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agentizen/agent-sdk-go/pkg/model"
	"github.com/agentizen/agent-sdk-go/pkg/model/providers/lmstudio"
	"github.com/stretchr/testify/assert"
)

func TestLMStudioProvider_NewProvider(t *testing.T) {
	t.Run("NewLMStudioProvider_DefaultURL", func(t *testing.T) {
		provider := lmstudio.NewLMStudioProvider("")
		assert.NotNil(t, provider)
		assert.Equal(t, lmstudio.DefaultBaseURL, provider.BaseURL)
	})

	t.Run("NewLMStudioProvider_CustomURL", func(t *testing.T) {
		provider := lmstudio.NewLMStudioProvider("http://localhost:5000/v1")
		assert.NotNil(t, provider)
		assert.Equal(t, "http://localhost:5000/v1", provider.BaseURL)
	})

	t.Run("NewProvider", func(t *testing.T) {
		provider := lmstudio.NewProvider()
		assert.NotNil(t, provider)
		assert.Equal(t, lmstudio.DefaultBaseURL, provider.BaseURL)
	})
}

func TestLMStudioProvider_WithAPIKey(t *testing.T) {
	provider := lmstudio.NewProvider()
	result := provider.WithAPIKey("my-api-key")
	assert.Equal(t, "my-api-key", provider.APIKey)
	assert.Equal(t, provider, result)
}

func TestLMStudioProvider_WithHTTPClient(t *testing.T) {
	provider := lmstudio.NewProvider()
	client := &http.Client{Timeout: 30 * time.Second}
	result := provider.WithHTTPClient(client)
	assert.Equal(t, client, provider.HTTPClient)
	assert.Equal(t, provider, result)
}

func TestLMStudioProvider_WithDefaultModel(t *testing.T) {
	provider := lmstudio.NewProvider()
	result := provider.WithDefaultModel("llama-3")
	assert.Equal(t, "llama-3", provider.DefaultModel)
	assert.Equal(t, provider, result)
}

func TestLMStudioProvider_SetBaseURL(t *testing.T) {
	provider := lmstudio.NewProvider()
	result := provider.SetBaseURL("http://custom:8080/v1")
	assert.Equal(t, "http://custom:8080/v1", provider.BaseURL)
	assert.Equal(t, provider, result)
}

func TestLMStudioProvider_SetDefaultModel(t *testing.T) {
	provider := lmstudio.NewProvider()
	result := provider.SetDefaultModel("mistral-7b")
	assert.Equal(t, "mistral-7b", provider.DefaultModel)
	assert.Equal(t, provider, result)
}

func TestLMStudioProvider_GetModel(t *testing.T) {
	t.Run("GetModel_ByName", func(t *testing.T) {
		provider := lmstudio.NewProvider()
		m, err := provider.GetModel("llama-3")
		assert.NoError(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, "llama-3", m.(*lmstudio.Model).ModelName)
	})

	t.Run("GetModel_UsesDefault", func(t *testing.T) {
		provider := lmstudio.NewProvider()
		provider.WithDefaultModel("default-model")
		m, err := provider.GetModel("")
		assert.NoError(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, "default-model", m.(*lmstudio.Model).ModelName)
	})

	t.Run("GetModel_NoNameNoDefault", func(t *testing.T) {
		provider := lmstudio.NewProvider()
		m, err := provider.GetModel("")
		assert.Error(t, err)
		assert.Nil(t, m)
		assert.Contains(t, err.Error(), "no model name provided")
	})
}

func TestLMStudioModel_GetResponse_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		response := map[string]interface{}{
			"id":      "chatcmpl-1",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "llama-3",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "Hello from LM Studio!",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     8,
				"completion_tokens": 5,
				"total_tokens":      13,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(response)
		assert.NoError(t, err)
	}))
	defer server.Close()

	provider := lmstudio.NewLMStudioProvider(server.URL + "/v1")
	m, err := provider.GetModel("llama-3")
	assert.NoError(t, err)

	request := &model.Request{
		Input:              "Say hello",
		SystemInstructions: "You are helpful",
	}

	resp, err := m.GetResponse(context.Background(), request)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Hello from LM Studio!", resp.Content)
	assert.Equal(t, 13, resp.Usage.TotalTokens)
}

func TestLMStudioModel_GetResponse_WithAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer my-secret-key", r.Header.Get("Authorization"))

		response := map[string]interface{}{
			"id":    "chatcmpl-2",
			"model": "llama-3",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "OK"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{"total_tokens": 5},
		}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(response)
		assert.NoError(t, err)
	}))
	defer server.Close()

	provider := lmstudio.NewLMStudioProvider(server.URL + "/v1")
	provider.WithAPIKey("my-secret-key")
	m, err := provider.GetModel("llama-3")
	assert.NoError(t, err)

	resp, err := m.GetResponse(context.Background(), &model.Request{Input: "test"})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "OK", resp.Content)
}

func TestLMStudioModel_GetResponse_WithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		assert.NoError(t, err)

		tools, ok := reqBody["tools"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, tools, 1)

		response := map[string]interface{}{
			"id":    "chatcmpl-3",
			"model": "llama-3",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role": "assistant",
						"tool_calls": []map[string]interface{}{
							{
								"id":   "call_abc",
								"type": "function",
								"function": map[string]interface{}{
									"name":      "search",
									"arguments": `{"query":"golang"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]interface{}{"total_tokens": 20},
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(response)
		assert.NoError(t, err)
	}))
	defer server.Close()

	provider := lmstudio.NewLMStudioProvider(server.URL + "/v1")
	m, err := provider.GetModel("llama-3")
	assert.NoError(t, err)

	tool := map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        "search",
			"description": "Search the web",
			"parameters": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	resp, err := m.GetResponse(context.Background(), &model.Request{
		Input: "search for something",
		Tools: []interface{}{tool},
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "search", resp.ToolCalls[0].Name)
	assert.Equal(t, "golang", resp.ToolCalls[0].Parameters["query"])
}

func TestLMStudioModel_GetResponse_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		err := json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "internal server error",
				"type":    "server_error",
			},
		})
		assert.NoError(t, err)
	}))
	defer server.Close()

	provider := lmstudio.NewLMStudioProvider(server.URL + "/v1")
	m, err := provider.GetModel("llama-3")
	assert.NoError(t, err)

	resp, err := m.GetResponse(context.Background(), &model.Request{Input: "test"})
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestLMStudioModel_GetResponse_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"id":      "chatcmpl-4",
			"model":   "llama-3",
			"choices": []interface{}{},
			"usage":   map[string]interface{}{"total_tokens": 0},
		}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(response)
		assert.NoError(t, err)
	}))
	defer server.Close()

	provider := lmstudio.NewLMStudioProvider(server.URL + "/v1")
	m, err := provider.GetModel("llama-3")
	assert.NoError(t, err)

	resp, err := m.GetResponse(context.Background(), &model.Request{Input: "test"})
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "no choices")
}

func TestLMStudioModel_GetResponse_WithSettings(t *testing.T) {
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&capturedBody)
		assert.NoError(t, err)

		response := map[string]interface{}{
			"id":    "chatcmpl-5",
			"model": "llama-3",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "OK"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{"total_tokens": 5},
		}
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(response)
		assert.NoError(t, err)
	}))
	defer server.Close()

	provider := lmstudio.NewLMStudioProvider(server.URL + "/v1")
	m, err := provider.GetModel("llama-3")
	assert.NoError(t, err)

	temperature := 0.7
	maxTokens := 100
	settings := &model.Settings{
		Temperature: &temperature,
		MaxTokens:   &maxTokens,
	}

	resp, err := m.GetResponse(context.Background(), &model.Request{
		Input:    "test",
		Settings: settings,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.InDelta(t, 0.7, capturedBody["temperature"], 0.001)
	assert.Equal(t, float64(100), capturedBody["max_tokens"])
}

func TestLMStudioModel_StreamResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		assert.NoError(t, err)
		assert.True(t, reqBody["stream"].(bool))

		flusher, ok := w.(http.Flusher)
		assert.True(t, ok)

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		events := []string{
			`{"choices":[{"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`{"choices":[{"delta":{"content":" world"},"finish_reason":null}]}`,
			`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
		}

		for _, event := range events {
			_, err = w.Write([]byte("data: " + event + "\n\n"))
			assert.NoError(t, err)
			flusher.Flush()
			time.Sleep(5 * time.Millisecond)
		}
		_, err = w.Write([]byte("data: [DONE]\n\n"))
		assert.NoError(t, err)
		flusher.Flush()
	}))
	defer server.Close()

	provider := lmstudio.NewLMStudioProvider(server.URL + "/v1")
	m, err := provider.GetModel("llama-3")
	assert.NoError(t, err)

	stream, err := m.StreamResponse(context.Background(), &model.Request{Input: "say hello"})
	assert.NoError(t, err)
	assert.NotNil(t, stream)

	var content string
	for event := range stream {
		if event.Type == model.StreamEventTypeContent {
			content += event.Content
		}
	}

	assert.Equal(t, "Hello world", content)
}
