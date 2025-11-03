package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	gatewayURL  string
	model       string
	stream      bool
	dryRun      bool
	showCost    bool
	showMetrics bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "llmgateway",
		Short: "LLM Gateway CLI",
		Long:  "Command-line interface for the LLM Gateway",
	}

	chatCmd := &cobra.Command{
		Use:   "chat [query]",
		Short: "Send a chat query to the gateway",
		Args:  cobra.MinimumNArgs(1),
		Run:   runChat,
	}
	chatCmd.Flags().StringVar(&gatewayURL, "url", "http://localhost:8080", "Gateway URL")
	chatCmd.Flags().StringVar(&model, "model", "openai", "LLM model to use")
	chatCmd.Flags().BoolVar(&stream, "stream", false, "Stream the response")
	chatCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Dry-run mode (estimate cost only)")
	chatCmd.Flags().BoolVar(&showCost, "show-cost", true, "Show cost information")
	chatCmd.Flags().BoolVar(&showMetrics, "show-metrics", false, "Show detailed metrics")

	costCmd := &cobra.Command{
		Use:   "cost [query]",
		Short: "Estimate the cost of a query",
		Args:  cobra.MinimumNArgs(1),
		Run:   runCost,
	}
	costCmd.Flags().StringVar(&gatewayURL, "url", "http://localhost:8080", "Gateway URL")
	costCmd.Flags().StringVar(&model, "model", "openai", "LLM model to use")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check provider status",
		Run:   runStatus,
	}
	statusCmd.Flags().StringVar(&gatewayURL, "url", "http://localhost:8080", "Gateway URL")

	jobCmd := &cobra.Command{
		Use:   "job",
		Short: "Manage async jobs",
	}

	jobSubmitCmd := &cobra.Command{
		Use:   "submit [query]",
		Short: "Submit an async job",
		Args:  cobra.MinimumNArgs(1),
		Run:   runJobSubmit,
	}
	jobSubmitCmd.Flags().StringVar(&gatewayURL, "url", "http://localhost:8080", "Gateway URL")
	jobSubmitCmd.Flags().StringVar(&model, "model", "openai", "LLM model to use")

	jobStatusCmd := &cobra.Command{
		Use:   "status [job-id]",
		Short: "Check job status",
		Args:  cobra.ExactArgs(1),
		Run:   runJobStatus,
	}
	jobStatusCmd.Flags().StringVar(&gatewayURL, "url", "http://localhost:8080", "Gateway URL")

	jobResultCmd := &cobra.Command{
		Use:   "result [job-id]",
		Short: "Get job result",
		Args:  cobra.ExactArgs(1),
		Run:   runJobResult,
	}
	jobResultCmd.Flags().StringVar(&gatewayURL, "url", "http://localhost:8080", "Gateway URL")

	jobCmd.AddCommand(jobSubmitCmd, jobStatusCmd, jobResultCmd)

	rootCmd.AddCommand(chatCmd, costCmd, statusCmd, jobCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runChat(cmd *cobra.Command, args []string) {
	query := strings.Join(args, " ")

	if stream {
		streamChat(query)
	} else if dryRun {
		dryRunChat(query)
	} else {
		normalChat(query)
	}
}

func normalChat(query string) {
	reqBody := map[string]interface{}{
		"query":     query,
		"model":     model,
		"task_type": "chat",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling request: %v\n", err)
		return
	}

	resp, err := http.Post(gatewayURL+"/v1/gateway/query", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error response (%d): %s\n", resp.StatusCode, string(body))
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
		return
	}

	fmt.Println(result["response"])

	if showCost {
		if cost, ok := result["cost_usd"].(float64); ok {
			fmt.Printf("\n💰 Cost: $%.6f\n", cost)
		}
		if inputTokens, ok := result["input_tokens"].(float64); ok {
			fmt.Printf("📊 Tokens: %d in, ", int(inputTokens))
			if outputTokens, ok := result["output_tokens"].(float64); ok {
				fmt.Printf("%d out, ", int(outputTokens))
				if totalTokens, ok := result["total_tokens"].(float64); ok {
					fmt.Printf("%d total\n", int(totalTokens))
				}
			}
		}
	}

	if showMetrics {
		if responseTime, ok := result["response_time_ms"].(float64); ok {
			fmt.Printf("⏱️  Response time: %dms\n", int(responseTime))
		}
		if modelVersion, ok := result["model_version"].(string); ok {
			fmt.Printf("🤖 Model: %s\n", modelVersion)
		}
		if cached, ok := result["cached"].(bool); ok && cached {
			fmt.Println("💾 Cached response")
		}
	}
}

func streamChat(query string) {
	reqBody := map[string]interface{}{
		"query": query,
		"model": model,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling request: %v\n", err)
		return
	}

	resp, err := http.Post(gatewayURL+"/v1/gateway/stream", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error response (%d): %s\n", resp.StatusCode, string(body))
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	totalCost := 0.0

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if token, ok := chunk["token"].(string); ok {
			fmt.Print(token)
		}

		if cost, ok := chunk["cost_usd"].(float64); ok {
			totalCost = cost
		}

		if done, ok := chunk["done"].(bool); ok && done {
			if finalCost, ok := chunk["total_cost_usd"].(float64); ok {
				totalCost = finalCost
			}
			break
		}

		if errMsg, ok := chunk["error"].(string); ok {
			fmt.Fprintf(os.Stderr, "\nError: %s\n", errMsg)
			return
		}
	}

	fmt.Println()

	if showCost && totalCost > 0 {
		fmt.Printf("\n💰 Total cost: $%.6f\n", totalCost)
	}
}

func dryRunChat(query string) {
	reqBody := map[string]interface{}{
		"query":     query,
		"model":     model,
		"task_type": "chat",
		"dry_run":   true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling request: %v\n", err)
		return
	}

	resp, err := http.Post(gatewayURL+"/v1/gateway/dry-run", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error response (%d): %s\n", resp.StatusCode, string(body))
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
		return
	}

	fmt.Println("🔍 Dry-run results:")
	if valid, ok := result["valid"].(bool); ok {
		if valid {
			fmt.Println("✅ Request is valid")
		} else {
			fmt.Println("❌ Request is invalid")
			if errors, ok := result["errors"].([]interface{}); ok {
				for _, err := range errors {
					fmt.Printf("   - %v\n", err)
				}
			}
		}
	}

	if cost, ok := result["estimated_cost_usd"].(float64); ok {
		fmt.Printf("💰 Estimated cost: $%.6f\n", cost)
	}

	if inputTokens, ok := result["input_tokens"].(float64); ok {
		fmt.Printf("📊 Estimated tokens: %d in, ", int(inputTokens))
		if outputTokens, ok := result["output_tokens"].(float64); ok {
			fmt.Printf("%d out\n", int(outputTokens))
		}
	}

	if provider, ok := result["provider"].(string); ok {
		fmt.Printf("🤖 Provider: %s", provider)
		if modelVersion, ok := result["model_version"].(string); ok {
			fmt.Printf(" (%s)", modelVersion)
		}
		fmt.Println()
	}

	if withinBudget, ok := result["within_budget"].(bool); ok {
		if withinBudget {
			fmt.Println("✅ Within budget limits")
		} else {
			fmt.Println("⚠️  Exceeds budget limits")
		}
	}
}

func runCost(cmd *cobra.Command, args []string) {
	query := strings.Join(args, " ")

	reqBody := map[string]interface{}{
		"query": query,
		"model": model,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling request: %v\n", err)
		return
	}

	resp, err := http.Post(gatewayURL+"/v1/gateway/cost-estimate", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error response (%d): %s\n", resp.StatusCode, string(body))
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
		return
	}

	fmt.Println("💰 Cost Estimate:")
	if cost, ok := result["estimated_cost_usd"].(float64); ok {
		fmt.Printf("   Total: $%.6f\n", cost)
	}

	if inputTokens, ok := result["input_tokens"].(float64); ok {
		fmt.Printf("📊 Tokens: %d in, ", int(inputTokens))
		if outputTokens, ok := result["output_tokens"].(float64); ok {
			fmt.Printf("%d out\n", int(outputTokens))
		}
	}

	if pricePerInput, ok := result["price_per_input_token"].(float64); ok {
		fmt.Printf("💵 Pricing: $%.8f per input token, ", pricePerInput)
		if pricePerOutput, ok := result["price_per_output_token"].(float64); ok {
			fmt.Printf("$%.8f per output token\n", pricePerOutput)
		}
	}

	if modelVersion, ok := result["model_version"].(string); ok {
		fmt.Printf("🤖 Model: %s (%s)\n", model, modelVersion)
	}
}

func runStatus(cmd *cobra.Command, args []string) {
	resp, err := http.Get(gatewayURL + "/api/status")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error response (%d): %s\n", resp.StatusCode, string(body))
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
		return
	}

	fmt.Println("🏥 Provider Status:")
	if providers, ok := result["providers"].(map[string]interface{}); ok {
		for name, info := range providers {
			if providerInfo, ok := info.(map[string]interface{}); ok {
				available, _ := providerInfo["available"].(bool)
				status := "❌ Unavailable"
				if available {
					status = "✅ Available"
				}
				fmt.Printf("   %s: %s\n", name, status)
			}
		}
	}
}

func runJobSubmit(cmd *cobra.Command, args []string) {
	query := strings.Join(args, " ")

	reqBody := map[string]interface{}{
		"query": query,
		"model": model,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling request: %v\n", err)
		return
	}

	resp, err := http.Post(gatewayURL+"/v1/gateway/jobs", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error response (%d): %s\n", resp.StatusCode, string(body))
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
		return
	}

	fmt.Println("✅ Job submitted successfully")
	if jobID, ok := result["job_id"].(string); ok {
		fmt.Printf("🆔 Job ID: %s\n", jobID)
	}
	if status, ok := result["status"].(string); ok {
		fmt.Printf("📊 Status: %s\n", status)
	}
	if cost, ok := result["estimated_cost_usd"].(float64); ok {
		fmt.Printf("💰 Estimated cost: $%.6f\n", cost)
	}
}

func runJobStatus(cmd *cobra.Command, args []string) {
	jobID := args[0]

	resp, err := http.Get(fmt.Sprintf("%s/v1/gateway/jobs/%s", gatewayURL, jobID))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error response (%d): %s\n", resp.StatusCode, string(body))
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
		return
	}

	fmt.Printf("🆔 Job ID: %s\n", jobID)
	if status, ok := result["status"].(string); ok {
		statusIcon := "📊"
		switch status {
		case "pending":
			statusIcon = "⏳"
		case "running":
			statusIcon = "🔄"
		case "completed":
			statusIcon = "✅"
		case "failed":
			statusIcon = "❌"
		}
		fmt.Printf("%s Status: %s\n", statusIcon, status)
	}

	if createdAt, ok := result["created_at"].(string); ok {
		t, _ := time.Parse(time.RFC3339, createdAt)
		fmt.Printf("🕐 Created: %s\n", t.Format("2006-01-02 15:04:05"))
	}

	if startedAt, ok := result["started_at"].(string); ok && startedAt != "" {
		t, _ := time.Parse(time.RFC3339, startedAt)
		fmt.Printf("▶️  Started: %s\n", t.Format("2006-01-02 15:04:05"))
	}

	if completedAt, ok := result["completed_at"].(string); ok && completedAt != "" {
		t, _ := time.Parse(time.RFC3339, completedAt)
		fmt.Printf("🏁 Completed: %s\n", t.Format("2006-01-02 15:04:05"))
	}

	if cost, ok := result["estimated_cost_usd"].(float64); ok {
		fmt.Printf("💰 Estimated cost: $%.6f\n", cost)
	}
}

func runJobResult(cmd *cobra.Command, args []string) {
	jobID := args[0]

	resp, err := http.Get(fmt.Sprintf("%s/v1/gateway/jobs/%s/result", gatewayURL, jobID))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "Error response (%d): %s\n", resp.StatusCode, string(body))
		return
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding response: %v\n", err)
		return
	}

	if status, ok := result["status"].(string); ok && status == "failed" {
		fmt.Println("❌ Job failed")
		if errMsg, ok := result["error"].(string); ok {
			fmt.Printf("Error: %s\n", errMsg)
		}
		return
	}

	fmt.Println("✅ Job completed successfully")
	fmt.Println()

	if resultText, ok := result["result"].(string); ok {
		fmt.Println(resultText)
		fmt.Println()
	}

	if cost, ok := result["actual_cost_usd"].(float64); ok {
		fmt.Printf("💰 Actual cost: $%.6f\n", cost)
	}

	if inputTokens, ok := result["input_tokens"].(float64); ok {
		fmt.Printf("📊 Tokens: %d in, ", int(inputTokens))
		if outputTokens, ok := result["output_tokens"].(float64); ok {
			fmt.Printf("%d out, ", int(outputTokens))
			if totalTokens, ok := result["total_tokens"].(float64); ok {
				fmt.Printf("%d total\n", int(totalTokens))
			}
		}
	}

	if provider, ok := result["provider"].(string); ok {
		fmt.Printf("🤖 Provider: %s\n", provider)
	}
}
