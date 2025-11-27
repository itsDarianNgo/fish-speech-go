package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/fish-speech-go/fish-speech-go/internal/schema"
)

var (
	serverURL     string
	outputFile    string
	format        string
	streaming     bool
	referenceFile string
	referenceText string
	temperature   float64
	topP          float64
	seed          int
	apiKey        string
)

var rootCmd = &cobra.Command{
	Use:   "fish-tts [text]",
	Short: "Generate speech from text using Fish-Speech",
	Long: `fish-tts is a command-line tool for text-to-speech generation.

Examples:
  # Basic TTS
  fish-tts "Hello, world!"

  # Save to file
  fish-tts -o output.wav "Hello, world!"

  # Use custom server
  fish-tts --server http://localhost:8080 "Hello, world!"

  # Voice cloning with reference audio
  fish-tts --reference voice.wav --reference-text "Sample text" "Hello in cloned voice"

  # Adjust generation parameters
  fish-tts --temperature 0.7 --top-p 0.9 "Hello, world!"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runTTS,
}

func init() {
	rootCmd.Flags().StringVarP(&serverURL, "server", "s", "http://localhost:8080", "Fish-Speech server URL")
	rootCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout/play)")
	rootCmd.Flags().StringVarP(&format, "format", "f", "wav", "Audio format: wav, mp3, pcm")
	rootCmd.Flags().BoolVar(&streaming, "stream", false, "Enable streaming mode")
	rootCmd.Flags().StringVar(&referenceFile, "reference", "", "Reference audio file for voice cloning")
	rootCmd.Flags().StringVar(&referenceText, "reference-text", "", "Text spoken in reference audio")
	rootCmd.Flags().Float64Var(&temperature, "temperature", 0.8, "Generation temperature (0.1-1.0)")
	rootCmd.Flags().Float64Var(&topP, "top-p", 0.8, "Top-p sampling (0.1-1.0)")
	rootCmd.Flags().IntVar(&seed, "seed", 0, "Random seed (0 = random)")
	rootCmd.Flags().StringVar(&apiKey, "api-key", "", "API key for authentication")
}

func runTTS(cmd *cobra.Command, args []string) error {
	text := args[0]

	req := schema.ServeTTSRequest{
		Text:        text,
		Format:      format,
		Streaming:   streaming,
		Temperature: temperature,
		TopP:        topP,
	}

	if seed != 0 {
		req.Seed = &seed
	}

	if referenceFile != "" {
		audioData, err := os.ReadFile(referenceFile)
		if err != nil {
			return fmt.Errorf("failed to read reference file: %w", err)
		}

		if referenceText == "" {
			return fmt.Errorf("--reference-text is required when using --reference")
		}

		req.References = []schema.ServeReferenceAudio{
			{
				Audio: audioData,
				Text:  referenceText,
			},
		}
	}

	audio, err := makeTTSRequest(&req)
	if err != nil {
		return err
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, audio, 0o644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Audio saved to %s (%d bytes)\n", outputFile, len(audio))
		return nil
	}

	_, err = os.Stdout.Write(audio)
	return err
}

func makeTTSRequest(req *schema.ServeTTSRequest) ([]byte, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/v1/tts", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return audio, nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
