package backend

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

// Request mirrors the Python backend TTS request payload.
type Request struct {
	Text              string      `msgpack:"text"`
	ReferenceID       *string     `msgpack:"reference_id,omitempty"`
	References        []Reference `msgpack:"references,omitempty"`
	MaxNewTokens      int         `msgpack:"max_new_tokens"`
	ChunkLength       int         `msgpack:"chunk_length"`
	TopP              float64     `msgpack:"top_p"`
	Temperature       float64     `msgpack:"temperature"`
	RepetitionPenalty float64     `msgpack:"repetition_penalty"`
	Format            string      `msgpack:"format"`
	Streaming         bool        `msgpack:"streaming"`
	Seed              *int        `msgpack:"seed,omitempty"`
}

// Reference contains reference audio and optional transcript.
type Reference struct {
	Audio string `msgpack:"audio"`
	Text  string `msgpack:"text"`
}

// Client provides a minimal HTTP client for the Python backend.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient constructs a backend client with sane defaults.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{baseURL: baseURL, httpClient: httpClient}
}

// StreamTTS sends the request to the backend and streams the response body to w.
func (c *Client) StreamTTS(ctx context.Context, req *Request, w io.Writer) error {
	if c == nil {
		return fmt.Errorf("backend client not configured")
	}
	if req == nil {
		return fmt.Errorf("backend request is nil")
	}

	payload, err := msgpack.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal backend request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/tts", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create backend request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/msgpack")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("call backend: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("backend returned status %d", resp.StatusCode)
	}

	if _, err := io.CopyBuffer(w, resp.Body, make([]byte, 4096)); err != nil {
		return fmt.Errorf("stream backend body: %w", err)
	}

	return nil
}
