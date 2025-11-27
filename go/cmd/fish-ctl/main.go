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
)

var (
	serverURL string
	apiKey    string
	output    string
)

var rootCmd = &cobra.Command{
	Use:   "fish-ctl",
	Short: "Fish-Speech server management tool",
	Long: `fish-ctl is a management tool for Fish-Speech-Go servers.

Commands:
  health      Check server health
  references  Manage voice references`,
}

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check server health",
	RunE:  runHealth,
}

var referencesCmd = &cobra.Command{
	Use:   "references",
	Short: "Manage voice references",
}

var referencesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all voice references",
	RunE:  runReferencesList,
}

var referencesAddCmd = &cobra.Command{
	Use:   "add [id] [audio-file] [text]",
	Short: "Add a voice reference",
	Args:  cobra.ExactArgs(3),
	RunE:  runReferencesAdd,
}

var referencesDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a voice reference",
	Args:  cobra.ExactArgs(1),
	RunE:  runReferencesDelete,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&serverURL, "server", "s", "http://localhost:8080", "Fish-Speech server URL")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for authentication")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "text", "Output format: text, json")

	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(referencesCmd)

	referencesCmd.AddCommand(referencesListCmd)
	referencesCmd.AddCommand(referencesAddCmd)
	referencesCmd.AddCommand(referencesDeleteCmd)

	healthCmd.Flags().Bool("detailed", false, "Show detailed health information")
}

func runHealth(cmd *cobra.Command, args []string) error {
	detailed, _ := cmd.Flags().GetBool("detailed")

	url := serverURL + "/v1/health"
	if detailed {
		url += "?detailed=true"
	}

	resp, err := makeRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if output == "json" {
		fmt.Println(string(resp))
		return nil
	}

	var health map[string]interface{}
	_ = json.Unmarshal(resp, &health)

	fmt.Printf("Status: %s\n", health["status"])
	if backend, ok := health["backend"].(map[string]interface{}); ok {
		fmt.Printf("Backend: %s", backend["status"])
		if latency, ok := backend["latency_ms"].(float64); ok {
			fmt.Printf(" (latency: %.0fms)", latency)
		}
		fmt.Println()
		if errMsg, ok := backend["error"].(string); ok {
			fmt.Printf("Backend Error: %s\n", errMsg)
		}
	}

	return nil
}

func runReferencesList(cmd *cobra.Command, args []string) error {
	resp, err := makeRequest(http.MethodGet, serverURL+"/v1/references", nil)
	if err != nil {
		return err
	}

	if output == "json" {
		fmt.Println(string(resp))
		return nil
	}

	var refs struct {
		Success      bool     `json:"success"`
		ReferenceIDs []string `json:"reference_ids"`
	}
	_ = json.Unmarshal(resp, &refs)

	if len(refs.ReferenceIDs) == 0 {
		fmt.Println("No references found")
		return nil
	}

	fmt.Println("Voice References:")
	for _, id := range refs.ReferenceIDs {
		fmt.Printf("  - %s\n", id)
	}

	return nil
}

func runReferencesAdd(cmd *cobra.Command, args []string) error {
	id := args[0]
	audioFile := args[1]
	text := args[2]

	audioData, err := os.ReadFile(audioFile)
	if err != nil {
		return fmt.Errorf("failed to read audio file: %w", err)
	}

	reqBody := map[string]interface{}{
		"id":    id,
		"audio": audioData,
		"text":  text,
	}
	body, _ := json.Marshal(reqBody)

	resp, err := makeRequest(http.MethodPost, serverURL+"/v1/references/add", body)
	if err != nil {
		return err
	}

	if output == "json" {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(resp, &result)

	if result.Success {
		fmt.Printf("✓ Reference '%s' added successfully\n", id)
	} else {
		fmt.Printf("✗ Failed: %s\n", result.Message)
	}

	return nil
}

func runReferencesDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	resp, err := makeRequest(http.MethodDelete, serverURL+"/v1/references/"+id, nil)
	if err != nil {
		return err
	}

	if output == "json" {
		fmt.Println(string(resp))
		return nil
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(resp, &result)

	if result.Success {
		fmt.Printf("✓ Reference '%s' deleted successfully\n", id)
	} else {
		fmt.Printf("✗ Failed: %s\n", result.Message)
	}

	return nil
}

func makeRequest(method, url string, body []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
