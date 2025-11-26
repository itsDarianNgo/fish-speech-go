package backend

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// TTSRequest represents the payload expected by the Python TTS backend.
type TTSRequest struct {
	Text        string   `msgpack:"text"`
	ReferenceID string   `msgpack:"reference_id,omitempty"`
	Streaming   bool     `msgpack:"streaming"`
	Format      string   `msgpack:"format"`
	TopP        *float64 `msgpack:"top_p,omitempty"`
	Temperature *float64 `msgpack:"temperature,omitempty"`
}

// HTTPError surfaces non-successful responses from the backend.
type HTTPError struct {
	StatusCode int
	Message    string
}

func (e HTTPError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("backend responded with status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("backend responded with status %d", e.StatusCode)
}

// Client sends MessagePack encoded requests to the backend service.
type Client struct {
	baseURL    string
	timeout    time.Duration
	httpClient *http.Client
}

// NewClient constructs a backend client with the provided base URL. If httpClient is nil,
// http.DefaultClient is used. When timeout is non-zero, requests are bounded by it.
func NewClient(baseURL string, timeout time.Duration, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		timeout:    timeout,
		httpClient: httpClient,
	}
}

// StreamTTS sends the request to the backend and returns the raw HTTP response for streaming.
// The caller is responsible for closing the response body when the returned error is nil.
func (c *Client) StreamTTS(ctx context.Context, payload TTSRequest) (*http.Response, error) {
	encoded, err := encodeTTSRequest(payload)
	if err != nil {
		return nil, fmt.Errorf("encode msgpack: %w", err)
	}

	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/tts", bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/msgpack")
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		resp.Body.Close()
		return nil, HTTPError{StatusCode: resp.StatusCode, Message: string(message)}
	}

	return resp, nil
}

func encodeTTSRequest(payload TTSRequest) ([]byte, error) {
	count := 3 // text, streaming, format are always present
	if payload.ReferenceID != "" {
		count++
	}
	if payload.TopP != nil {
		count++
	}
	if payload.Temperature != nil {
		count++
	}

	buf := &bytes.Buffer{}
	if err := writeMapHeader(buf, count); err != nil {
		return nil, err
	}

	writeString(buf, "text")
	writeString(buf, payload.Text)

	if payload.ReferenceID != "" {
		writeString(buf, "reference_id")
		writeString(buf, payload.ReferenceID)
	}

	writeString(buf, "streaming")
	writeBool(buf, payload.Streaming)

	writeString(buf, "format")
	writeString(buf, payload.Format)

	if payload.TopP != nil {
		writeString(buf, "top_p")
		writeFloat64(buf, *payload.TopP)
	}

	if payload.Temperature != nil {
		writeString(buf, "temperature")
		writeFloat64(buf, *payload.Temperature)
	}

	return buf.Bytes(), nil
}

func writeMapHeader(buf *bytes.Buffer, size int) error {
	switch {
	case size < 0:
		return errors.New("negative map size")
	case size <= 15:
		buf.WriteByte(0x80 | byte(size))
	case size <= 0xffff:
		buf.WriteByte(0xde)
		_ = binary.Write(buf, binary.BigEndian, uint16(size))
	default:
		return fmt.Errorf("map too large: %d", size)
	}
	return nil
}

func writeString(buf *bytes.Buffer, value string) {
	length := len(value)
	switch {
	case length <= 31:
		buf.WriteByte(0xa0 | byte(length))
	case length <= 0xff:
		buf.WriteByte(0xd9)
		buf.WriteByte(byte(length))
	case length <= 0xffff:
		buf.WriteByte(0xda)
		_ = binary.Write(buf, binary.BigEndian, uint16(length))
	default:
		buf.WriteByte(0xdb)
		_ = binary.Write(buf, binary.BigEndian, uint32(length))
	}
	buf.WriteString(value)
}

func writeBool(buf *bytes.Buffer, value bool) {
	if value {
		buf.WriteByte(0xc3)
		return
	}
	buf.WriteByte(0xc2)
}

func writeFloat64(buf *bytes.Buffer, value float64) {
	buf.WriteByte(0xcb)
	_ = binary.Write(buf, binary.BigEndian, value)
}
