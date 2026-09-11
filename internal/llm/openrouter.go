package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenRouter struct {
	APIKey, Model, BaseURL string
	HTTPClient             *http.Client
}
type completionResponse struct {
	Choices []struct {
		Message Message `json:"message"`
		Delta   struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func (o *OpenRouter) Complete(ctx context.Context, request CompletionRequest) (string, error) {
	payload := map[string]any{"model": o.Model, "messages": request.Messages}
	if request.JSONMode {
		payload["response_format"] = map[string]string{"type": "json_object"}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	response, err := o.do(ctx, body, false)
	if err != nil {
		return "", err
	}
	if len(response.Choices) == 0 {
		return "", errors.New("llm returned no choices")
	}
	return response.Choices[0].Message.Content, nil
}

func (o *OpenRouter) Stream(ctx context.Context, request CompletionRequest, callback func(string) error) error {
	body, err := json.Marshal(map[string]any{"model": o.Model, "messages": request.Messages, "stream": true})
	if err != nil {
		return err
	}
	response, err := o.request(ctx, body)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	scanner := bufio.NewScanner(response.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk completionResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) > 0 {
			if err := callback(chunk.Choices[0].Delta.Content); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

func (o *OpenRouter) do(ctx context.Context, body []byte, stream bool) (completionResponse, error) {
	response, err := o.request(ctx, body)
	if err != nil {
		return completionResponse{}, err
	}
	defer response.Body.Close()
	var result completionResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return result, err
	}
	return result, nil
}
func (o *OpenRouter) request(ctx context.Context, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(o.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+o.APIKey)
	req.Header.Set("Content-Type", "application/json")
	response, err := o.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= 300 {
		defer response.Body.Close()
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return nil, fmt.Errorf("llm provider status %d: %s", response.StatusCode, detail)
	}
	return response, nil
}

func NewOpenRouter(apiKey, model, baseURL string) *OpenRouter {
	return &OpenRouter{APIKey: apiKey, Model: model, BaseURL: baseURL, HTTPClient: &http.Client{Timeout: 45 * time.Second}}
}
