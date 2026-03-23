package model_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/citizenofai/agent-sdk-go/pkg/model"
	"github.com/citizenofai/agent-sdk-go/pkg/model/providers/mistral"
	"github.com/stretchr/testify/assert"
)

func TestMistralProvider_NewProvider(t *testing.T) {
	p := mistral.NewProvider("test-key")
	assert.Equal(t, "test-key", p.APIKey)
}

func TestMistralProvider_GetModel(t *testing.T) {
	p := mistral.NewProvider("test-key").WithDefaultModel("mistral-small-latest")

	m, err := p.GetModel("mistral-large-latest")
	assert.NoError(t, err)
	assert.Equal(t, "mistral-large-latest", m.(*mistral.Model).ModelName)

	m, err = p.GetModel("")
	assert.NoError(t, err)
	assert.Equal(t, "mistral-small-latest", m.(*mistral.Model).ModelName)
}

func TestMistralProvider_GetModel_NoAPIKey(t *testing.T) {
	p := mistral.NewProvider("")
	_, err := p.GetModel("mistral-small-latest")
	assert.Error(t, err)
}

func TestMistralModel_GetResponse_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		assert.NoError(t, err)

		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "mistral-small-latest",
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
				"prompt_tokens":     5,
				"completion_tokens": 7,
				"total_tokens":      12,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := mistral.NewProvider("test-key").WithDefaultModel("mistral-small-latest")
	p.SetEndpoint(server.URL)

	m, err := p.GetModel("")
	assert.NoError(t, err)

	req := &model.Request{
		SystemInstructions: "You are a test assistant.",
		Input:              "Say hello",
	}

	resp, err := m.GetResponse(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "Test response", resp.Content)
	assert.Equal(t, 12, resp.Usage.TotalTokens)
}

func TestMistralModel_GetResponse_VisionImage(t *testing.T) {
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0} // minimal JPEG header bytes

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		assert.NoError(t, err)

		messages, _ := body["messages"].([]interface{})
		assert.Len(t, messages, 1)
		msg, _ := messages[0].(map[string]interface{})
		assert.Equal(t, "user", msg["role"])

		content, _ := msg["content"].([]interface{})
		assert.Len(t, content, 2)

		textPart, _ := content[0].(map[string]interface{})
		assert.Equal(t, "text", textPart["type"])
		assert.Equal(t, "What is in this image?", textPart["text"])

		imagePart, _ := content[1].(map[string]interface{})
		assert.Equal(t, "image_url", imagePart["type"])
		imageURL, _ := imagePart["image_url"].(map[string]interface{})
		url, _ := imageURL["url"].(string)
		assert.True(t, strings.HasPrefix(url, "data:image/jpeg;base64,"), "image URL should be a base64 data URI")

		resp := map[string]interface{}{
			"id":      "chatcmpl-vision-test",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "mistral-small-2603",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "I see an image."},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{"total_tokens": 15},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := mistral.NewProvider("test-key").WithDefaultModel("mistral-small-2603")
	p.SetEndpoint(server.URL)

	m, err := p.GetModel("")
	assert.NoError(t, err)

	req := &model.Request{
		Input: "What is in this image?",
		InputParts: []model.ContentPart{
			{
				Type:     model.ContentPartTypeImage,
				MimeType: "image/jpeg",
				Data:     imageData,
				Name:     "photo.jpg",
			},
		},
	}

	resp, err := m.GetResponse(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "I see an image.", resp.Content)
}

func TestMistralModel_GetResponse_VisionRejectedOnNonVisionModel(t *testing.T) {
	// Use an unknown model name so no vision capability matches → validateInputParts rejects the image.
	p := mistral.NewProvider("test-key").WithDefaultModel("mistral-text-only-test")

	m, err := p.GetModel("")
	assert.NoError(t, err)

	req := &model.Request{
		InputParts: []model.ContentPart{
			{
				Type:     model.ContentPartTypeImage,
				MimeType: "image/jpeg",
				Data:     []byte{0xFF, 0xD8},
				Name:     "photo.jpg",
			},
		},
	}

	_, err = m.GetResponse(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image")
}

func TestMistralModel_StreamResponse_VisionImage(t *testing.T) {
	imageData := []byte{0xFF, 0xD8, 0xFF, 0xE0}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		assert.NoError(t, err)
		assert.True(t, body["stream"].(bool))

		messages, _ := body["messages"].([]interface{})
		assert.Len(t, messages, 1)
		msg, _ := messages[0].(map[string]interface{})
		content, _ := msg["content"].([]interface{})
		assert.Len(t, content, 1)
		imagePart, _ := content[0].(map[string]interface{})
		assert.Equal(t, "image_url", imagePart["type"])

		flusher, ok := w.(http.Flusher)
		assert.True(t, ok)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []string{
			`{"id":"stream-1","model":"mistral-small-2603","choices":[{"index":0,"delta":{"role":"assistant","content":"An"},"finish_reason":null}]}`,
			`{"id":"stream-1","model":"mistral-small-2603","choices":[{"index":0,"delta":{"content":" image."},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`,
		}
		for _, chunk := range chunks {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
		_, _ = fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	p := mistral.NewProvider("test-key").WithDefaultModel("mistral-small-2603")
	p.SetEndpoint(server.URL)

	m, err := p.GetModel("")
	assert.NoError(t, err)

	req := &model.Request{
		InputParts: []model.ContentPart{
			{
				Type:     model.ContentPartTypeImage,
				MimeType: "image/jpeg",
				Data:     imageData,
				Name:     "photo.jpg",
			},
		},
	}

	events, err := m.StreamResponse(context.Background(), req)
	assert.NoError(t, err)

	var content string
	var doneEvent *model.StreamEvent
	for event := range events {
		switch event.Type {
		case model.StreamEventTypeContent:
			content += event.Content
		case model.StreamEventTypeDone:
			ev := event
			doneEvent = &ev
		}
	}

	assert.Equal(t, "An image.", content)
	assert.NotNil(t, doneEvent)
	assert.Equal(t, "An image.", doneEvent.Response.Content)
}

func TestMistralModel_GetResponse_DocumentRejectedOnNonVisionModel(t *testing.T) {
	// Use an unknown model name so no vision capability matches → validateInputParts rejects the document.
	p := mistral.NewProvider("test-key").WithDefaultModel("mistral-text-only-test")

	m, err := p.GetModel("")
	assert.NoError(t, err)

	req := &model.Request{
		InputParts: []model.ContentPart{
			{
				Type:     model.ContentPartTypeDocument,
				MimeType: "application/pdf",
				Data:     []byte("%PDF-1.4 test"),
				Name:     "report.pdf",
			},
		},
	}

	_, err = m.GetResponse(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document")
}

func TestMistralModel_UsesCompletionsEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)

		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "mistral-large-2512",
			"choices": []map[string]interface{}{
				{
					"index":         0,
					"message":       map[string]interface{}{"role": "assistant", "content": "Hello"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{"total_tokens": 5},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := mistral.NewProvider("test-key").WithDefaultModel("mistral-large-2512")
	p.SetEndpoint(server.URL)

	m, err := p.GetModel("")
	assert.NoError(t, err)

	req := &model.Request{Input: "Hello"}
	resp, err := m.GetResponse(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "Hello", resp.Content)
}

func TestMistralModel_GetResponse_WithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)

		tools, ok := body["tools"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, tools, 1)

		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "mistral-small-latest",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role": "assistant",
						"tool_calls": []map[string]interface{}{
							{
								"id":   "tool_1",
								"type": "function",
								"function": map[string]interface{}{
									"name":      "test_tool",
									"arguments": `{"param1":"value1"}`,
								},
							},
						},
					},
					"finish_reason": "tool_calls",
				},
			},
			"usage": map[string]interface{}{
				"total_tokens": 10,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := mistral.NewProvider("test-key").WithDefaultModel("mistral-small-latest")
	p.SetEndpoint(server.URL)
	m, err := p.GetModel("")
	assert.NoError(t, err)

	toolDef := map[string]interface{}{
		"name":        "test_tool",
		"description": "A test tool",
		"parameters": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"param1": map[string]interface{}{"type": "string"},
			},
		},
	}

	req := &model.Request{
		Input: "Test input",
		Tools: []interface{}{toolDef},
	}

	resp, err := m.GetResponse(context.Background(), req)
	assert.NoError(t, err)
	assert.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "test_tool", resp.ToolCalls[0].Name)
	assert.Equal(t, "value1", resp.ToolCalls[0].Parameters["param1"])
}
