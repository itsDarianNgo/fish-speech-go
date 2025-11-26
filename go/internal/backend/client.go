package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fish-speech-go/fish-speech-go/internal/config"
	"github.com/fish-speech-go/fish-speech-go/internal/schema"
)

// BackendClient handles communication with the Python Fish-Speech server.
type BackendClient struct {
	httpClient *http.Client
	endpoint   string
	timeout    time.Duration
}

// NewBackendClient creates a new backend client with connection pooling.
func NewBackendClient(cfg *config.BackendConfig) *BackendClient {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DisableCompression:  true,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}

	return &BackendClient{
		httpClient: client,
		endpoint:   cfg.URL,
		timeout:    cfg.Timeout,
	}
}

// Health checks if the Python backend is reachable.
func (c *BackendClient) Health(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/v1/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("backend unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("backend unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// TTS sends a TTS request and returns the complete audio response (non-streaming).
func (c *BackendClient) TTS(ctx context.Context, req *schema.ServeTTSRequest) ([]byte, string, error) {
	body, err := EncodeTTSRequest(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v1/tts", bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/msgpack")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, "", fmt.Errorf("%w: %v", ErrBackendTimeout, err)
		}
		return nil, "", fmt.Errorf("%w: %v", ErrBackendUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, "", &BackendError{StatusCode: resp.StatusCode, Message: string(bodyBytes)}
	}

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response: %w", err)
	}

	return audioData, req.Format, nil
}

// TTSStream sends a TTS request and returns a streaming response.
func (c *BackendClient) TTSStream(ctx context.Context, req *schema.ServeTTSRequest) (io.ReadCloser, error) {
	req.Streaming = true
	body, err := EncodeTTSRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v1/tts", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/msgpack")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("%w: %v", ErrBackendTimeout, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrBackendUnavailable, err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &BackendError{StatusCode: resp.StatusCode, Message: string(bodyBytes)}
	}

	return resp.Body, nil
}

// VQGANEncode sends audio to be encoded to tokens.
func (c *BackendClient) VQGANEncode(ctx context.Context, req *schema.ServeVQGANEncodeRequest) (*schema.ServeVQGANEncodeResponse, error) {
	body, err := EncodeMsgpack(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v1/vqgan/encode", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/msgpack")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &BackendError{StatusCode: resp.StatusCode, Message: string(bodyBytes)}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result schema.ServeVQGANEncodeResponse
	if err := DecodeMsgpack(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// VQGANDecode sends tokens to be decoded to audio.
func (c *BackendClient) VQGANDecode(ctx context.Context, req *schema.ServeVQGANDecodeRequest) (*schema.ServeVQGANDecodeResponse, error) {
	body, err := EncodeMsgpack(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v1/vqgan/decode", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/msgpack")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &BackendError{StatusCode: resp.StatusCode, Message: string(bodyBytes)}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result schema.ServeVQGANDecodeResponse
	if err := DecodeMsgpack(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// AddReference adds a new voice reference.
func (c *BackendClient) AddReference(ctx context.Context, req *schema.AddReferenceRequest) (*schema.AddReferenceResponse, error) {
	body, err := EncodeMsgpack(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/v1/references/add", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/msgpack")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &BackendError{StatusCode: resp.StatusCode, Message: string(bodyBytes)}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result schema.AddReferenceResponse
	if strings.Contains(resp.Header.Get("Content-Type"), "msgpack") {
		if err := DecodeMsgpack(respBody, &result); err != nil {
			return nil, err
		}
	} else {
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, err
		}
	}

	return &result, nil
}

// ListReferences returns all saved voice references.
func (c *BackendClient) ListReferences(ctx context.Context) (*schema.ListReferencesResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/v1/references", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &BackendError{StatusCode: resp.StatusCode, Message: string(bodyBytes)}
	}

	var result schema.ListReferencesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteReference removes a voice reference by ID.
func (c *BackendClient) DeleteReference(ctx context.Context, id string) (*schema.DeleteReferenceResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.endpoint+"/v1/references/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, &BackendError{StatusCode: resp.StatusCode, Message: string(bodyBytes)}
	}

	var result schema.DeleteReferenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
