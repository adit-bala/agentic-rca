package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

func (c *Client) Get(ctx context.Context, path string, result interface{}) error {
	url := c.baseURL + path
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Forward common headers for tracing
	c.forwardHeaders(ctx, req)

	log.Info().Str("method", "GET").Str("url", url).Msg("Making HTTP request")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("HTTP request failed")
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Error().Int("status", resp.StatusCode).Str("url", url).Str("body", string(body)).Msg("HTTP request returned error")
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	log.Info().Int("status", resp.StatusCode).Str("url", url).Msg("HTTP request completed")
	return nil
}

func (c *Client) Post(ctx context.Context, path string, payload interface{}, result interface{}) error {
	url := c.baseURL + path
	
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshaling payload: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	
	// Forward common headers for tracing
	c.forwardHeaders(ctx, req)

	log.Info().Str("method", "POST").Str("url", url).Msg("Making HTTP request")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Str("url", url).Msg("HTTP request failed")
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		log.Error().Int("status", resp.StatusCode).Str("url", url).Str("body", string(body)).Msg("HTTP request returned error")
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	log.Info().Int("status", resp.StatusCode).Str("url", url).Msg("HTTP request completed")
	return nil
}

func (c *Client) forwardHeaders(ctx context.Context, req *http.Request) {
	// Forward trace ID if present in context
	if traceID := ctx.Value("trace-id"); traceID != nil {
		req.Header.Set("X-Trace-ID", traceID.(string))
	}
	
	// Forward user agent
	req.Header.Set("User-Agent", "simple-microservices/1.0")
}
