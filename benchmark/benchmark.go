package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type target struct {
	Text        string `json:"text"`
	ReferenceID string `json:"reference_id"`
}

type requestPayload struct {
	Text        string  `json:"text"`
	ReferenceID string  `json:"reference_id,omitempty"`
	Streaming   bool    `json:"streaming"`
	Format      string  `json:"format"`
	TopP        float64 `json:"top_p,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

type BenchmarkClient struct {
	baseURL     string
	streaming   bool
	referenceID string
	text        string
	targets     []target
	targetIndex uint64
	client      *http.Client
}

type runResult struct {
	duration          time.Duration
	success           bool
	statusCode        int
	err               error
	connectionLatency time.Duration
	streamingDuration time.Duration
	chunkErrors       int
}

func newBenchmarkClient(baseURL string, streaming bool, text string, referenceID string, targets []target) *BenchmarkClient {
	return &BenchmarkClient{
		baseURL:     strings.TrimRight(baseURL, "/"),
		streaming:   streaming,
		referenceID: referenceID,
		text:        text,
		targets:     targets,
		client: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *BenchmarkClient) nextTarget() target {
	if len(c.targets) == 0 {
		return target{Text: c.text, ReferenceID: c.referenceID}
	}

	idx := atomic.AddUint64(&c.targetIndex, 1)
	return c.targets[(idx-1)%uint64(len(c.targets))]
}

func (c *BenchmarkClient) Do(ctx context.Context) runResult {
	start := time.Now()
	tgt := c.nextTarget()

	payload := requestPayload{
		Text:        tgt.Text,
		ReferenceID: tgt.ReferenceID,
		Streaming:   c.streaming,
		Format:      "wav",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return runResult{err: fmt.Errorf("encode request: %w", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/tts", bytes.NewReader(body))
	if err != nil {
		return runResult{err: fmt.Errorf("build request: %w", err)}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "fish-speech-benchmark/0.1")

	var connectionLatency time.Duration
	var firstByteAt time.Time

	trace := &httptrace.ClientTrace{
		GotFirstResponseByte: func() {
			connectionLatency = time.Since(start)
			firstByteAt = time.Now()
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	resp, err := c.client.Do(req)
	if err != nil {
		return runResult{duration: time.Since(start), err: err}
	}
	defer resp.Body.Close()

	var streamingDuration time.Duration
	chunkErrors := 0

	if c.streaming {
		buf := make([]byte, 4096)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 && firstByteAt.IsZero() {
				// Response without httptrace callback
				connectionLatency = time.Since(start)
				firstByteAt = time.Now()
			}
			if n > 0 {
				streamingDuration = time.Since(firstByteAt)
			}

			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				chunkErrors++
				err = readErr
				break
			}
		}
	} else {
		_, err = io.Copy(io.Discard, resp.Body)
	}

	duration := time.Since(start)
	success := err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300

	return runResult{
		duration:          duration,
		success:           success,
		statusCode:        resp.StatusCode,
		err:               err,
		connectionLatency: connectionLatency,
		streamingDuration: streamingDuration,
		chunkErrors:       chunkErrors,
	}
}

type summary struct {
	durations           []time.Duration
	connectionLatencies []time.Duration
	streamingDurations  []time.Duration
	total               int
	success             int
	chunkErrors         int
}

func (s *summary) add(result runResult) {
	s.total++
	if result.success {
		s.success++
		s.durations = append(s.durations, result.duration)
		if result.connectionLatency > 0 {
			s.connectionLatencies = append(s.connectionLatencies, result.connectionLatency)
		}
		if result.streamingDuration > 0 {
			s.streamingDurations = append(s.streamingDurations, result.streamingDuration)
		}
	}
	s.chunkErrors += result.chunkErrors
}

func percentile(values []time.Duration, p float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	rank := p * float64(len(values)-1)
	lower := int(rank)
	upper := lower + 1
	if upper >= len(values) {
		return values[lower]
	}
	weight := rank - float64(lower)
	return time.Duration(float64(values[lower])*(1-weight) + float64(values[upper])*weight)
}

func average(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}
	var total time.Duration
	for _, v := range values {
		total += v
	}
	return total / time.Duration(len(values))
}

func loadTargets(path string) ([]target, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items []target
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func main() {
	baseURL := flag.String("base-url", "http://127.0.0.1:8080", "Benchmark target base URL")
	count := flag.Int("count", 1, "Number of requests to send")
	concurrency := flag.Int("concurrency", 1, "Number of concurrent workers")
	streaming := flag.Bool("streaming", false, "Enable streaming mode")
	text := flag.String("text", "你好，世界", "Text to synthesize")
	referenceID := flag.String("reference-id", "", "Reference voice ID")
	endpointsFile := flag.String("endpoint", "", "Path to JSON file with request targets")
	loop := flag.Bool("loop", false, "Send requests continuously until interrupted")
	flag.Parse()

	var targets []target
	if *endpointsFile != "" {
		loadedTargets, err := loadTargets(*endpointsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load endpoints: %v\n", err)
			os.Exit(1)
		}
		targets = loadedTargets
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	client := newBenchmarkClient(*baseURL, *streaming, *text, *referenceID, targets)

	jobs := make(chan struct{}, *concurrency)
	results := make(chan runResult, *concurrency)
	var workers sync.WaitGroup

	for i := 0; i < *concurrency; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}
				results <- client.Do(ctx)
			}
		}()
	}

	go func() {
		if *loop {
			for {
				select {
				case <-ctx.Done():
					close(jobs)
					return
				case jobs <- struct{}{}:
				}
			}
		}

		for i := 0; i < *count; i++ {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case jobs <- struct{}{}:
			}
		}
		close(jobs)
	}()

	go func() {
		workers.Wait()
		close(results)
	}()

	var sum summary
	for res := range results {
		sum.add(res)
		if res.err != nil {
			fmt.Fprintf(os.Stderr, "request error: %v\n", res.err)
		}
	}

	fmt.Printf("Total requests: %d\n", sum.total)
	fmt.Printf("Success: %d, Failed: %d\n", sum.success, sum.total-sum.success)

	if len(sum.durations) > 0 {
		fmt.Printf("Average duration: %s\n", average(sum.durations))
		fmt.Printf("P50: %s\n", percentile(sum.durations, 0.50))
		fmt.Printf("P75: %s\n", percentile(sum.durations, 0.75))
		fmt.Printf("P90: %s\n", percentile(sum.durations, 0.90))
		fmt.Printf("P95: %s\n", percentile(sum.durations, 0.95))
	}

	if *streaming {
		fmt.Println("Streaming metrics:")
		if len(sum.connectionLatencies) > 0 {
			fmt.Printf("  Avg connection latency: %s\n", average(sum.connectionLatencies))
			fmt.Printf("  P50 connection latency: %s\n", percentile(sum.connectionLatencies, 0.50))
		}
		if len(sum.streamingDurations) > 0 {
			fmt.Printf("  Avg streaming duration: %s\n", average(sum.streamingDurations))
			fmt.Printf("  P50 streaming duration: %s\n", percentile(sum.streamingDurations, 0.50))
		}
		fmt.Printf("  Chunk errors: %d\n", sum.chunkErrors)
	}
}
