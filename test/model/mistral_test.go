package model_test

import (
	"context"
	"encoding/json"
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

func TestMistralOCRModel_GetResponse_WithURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/ocr", r.URL.Path)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "mistral-ocr-2512", body["model"])
		doc, _ := body["document"].(map[string]interface{})
		assert.Equal(t, "document_url", doc["type"])
		assert.Equal(t, "https://example.com/doc.pdf", doc["document_url"])

		resp := map[string]interface{}{
			"id":    "ocr-test",
			"model": "mistral-ocr-2512",
			"usage": map[string]interface{}{
				"prompt_tokens": 100,
				"total_tokens":  100,
			},
			"pages": []map[string]interface{}{
				{"index": 0, "markdown": "# Page 1\nContent"},
				{"index": 1, "markdown": "## Page 2\nMore content"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := mistral.NewProvider("test-key").WithDefaultModel("mistral-ocr-2512")
	p.SetEndpoint(server.URL)

	m, err := p.GetModel("")
	assert.NoError(t, err)

	req := &model.Request{
		Input: "https://example.com/doc.pdf",
	}

	resp, err := m.GetResponse(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "# Page 1\nContent\n\n## Page 2\nMore content", resp.Content)
	assert.Equal(t, 100, resp.Usage.TotalTokens)
}

func TestMistralOCRModel_GetResponse_WithBase64Document(t *testing.T) {
	docData := []byte("%PDF-1.4 test content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/ocr", r.URL.Path)

		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		doc, _ := body["document"].(map[string]interface{})
		assert.Equal(t, "document_url", doc["type"])
		assert.Nil(t, doc["document_name"])
		assert.Nil(t, doc["base64_document"])
		docURL, _ := doc["document_url"].(string)
		assert.True(t, strings.HasPrefix(docURL, "data:application/pdf;base64,"), "document_url should be a base64 data URL")

		resp := map[string]interface{}{
			"id":    "ocr-test",
			"model": "mistral-ocr-2512",
			"usage": map[string]interface{}{"total_tokens": 50},
			"pages": []map[string]interface{}{
				{"index": 0, "markdown": "Invoice content"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := mistral.NewProvider("test-key").WithDefaultModel("mistral-ocr-2512")
	p.SetEndpoint(server.URL)

	m, err := p.GetModel("")
	assert.NoError(t, err)

	req := &model.Request{
		InputParts: []model.ContentPart{
			{
				Type:     model.ContentPartTypeDocument,
				MimeType: "application/pdf",
				Data:     docData,
				Name:     "invoice.pdf",
			},
		},
	}

	resp, err := m.GetResponse(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "Invoice content", resp.Content)
}

func TestMistralOCRModel_StreamResponse_WrapsSync(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/ocr", r.URL.Path)

		resp := map[string]interface{}{
			"id":    "ocr-stream-test",
			"model": "mistral-ocr-2512",
			"usage": map[string]interface{}{"total_tokens": 20},
			"pages": []map[string]interface{}{
				{"index": 0, "markdown": "Streamed OCR content"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := mistral.NewProvider("test-key").WithDefaultModel("mistral-ocr-2512")
	p.SetEndpoint(server.URL)

	m, err := p.GetModel("")
	assert.NoError(t, err)

	req := &model.Request{
		Input: "https://example.com/document.pdf",
	}

	events, err := m.StreamResponse(context.Background(), req)
	assert.NoError(t, err)

	var contentEvents []string
	var doneEvent *model.StreamEvent
	for event := range events {
		switch event.Type {
		case model.StreamEventTypeContent:
			contentEvents = append(contentEvents, event.Content)
		case model.StreamEventTypeDone:
			ev := event
			doneEvent = &ev
		}
	}

	assert.Equal(t, []string{"Streamed OCR content"}, contentEvents)
	assert.NotNil(t, doneEvent)
	assert.Equal(t, "Streamed OCR content", doneEvent.Response.Content)
}

func TestMistralOCRModel_GetResponse_NoDocument(t *testing.T) {
	p := mistral.NewProvider("test-key").WithDefaultModel("mistral-ocr-2512")

	m, err := p.GetModel("")
	assert.NoError(t, err)

	req := &model.Request{}
	_, err = m.GetResponse(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "document")
}

func TestMistralNonOCRModel_UsesCompletionsEndpoint(t *testing.T) {
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
