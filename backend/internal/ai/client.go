package ai

import (
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

// Client — интерфейс провайдера ИИ.
type Client interface {
	Complete(ctx context.Context, system, user string) (string, error)
	Name() string
}

// NewClient: если url пуст, возвращается демо-режим без обращения к внешнему API.
func NewClient(url, key, model string) Client {
	if strings.TrimSpace(url) == "" {
		return mockClient{}
	}
	return &httpClient{url: strings.TrimRight(url, "/"), key: key, model: model, hc: &http.Client{Timeout: 30 * time.Second}}
}

type mockClient struct{}

func (mockClient) Name() string { return "mock" }
func (mockClient) Complete(context.Context, string, string) (string, error) {
	return "", errors.New("mock client has no model")
}

// httpClient работает с любым OpenAI-совместимым endpoint /chat/completions.
type httpClient struct {
	url, key, model string
	hc              *http.Client
}

func (c *httpClient) Name() string { return c.model }

func (c *httpClient) Complete(ctx context.Context, system, user string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.7,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm status %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("llm returned no choices")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}
