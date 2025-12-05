package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ossf/gemara/layer1/pipeline/types"
)

// OpenAIEnhancer uses OpenAI API for enhancement
type OpenAIEnhancer struct {
	EnhancerBase
	client *http.Client
}

// NewOpenAIEnhancer creates a new OpenAI enhancer
func NewOpenAIEnhancer(config types.LLMConfig) (*OpenAIEnhancer, error) {
	if config.Endpoint == "" {
		config.Endpoint = "https://api.openai.com/v1/chat/completions"
	}
	if config.Model == "" {
		config.Model = "gpt-4"
	}
	if config.Temperature == 0 {
		config.Temperature = 0.3 // Lower temperature for more consistent results
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 2000
	}
	
	e := &OpenAIEnhancer{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
	
	if err := e.Configure(config); err != nil {
		return nil, err
	}
	
	return e, nil
}

// Name returns the enhancer name
func (e *OpenAIEnhancer) Name() string {
	return fmt.Sprintf("openai-%s-v1.0", e.config.Model)
}

// OpenAIRequest represents an OpenAI API request
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

// OpenAIMessage represents a chat message
type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAIResponse represents an OpenAI API response
type OpenAIResponse struct {
	Choices []struct {
		Message OpenAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// callOpenAI makes a request to the OpenAI API
func (e *OpenAIEnhancer) callOpenAI(ctx context.Context, prompt string) (string, error) {
	req := OpenAIRequest{
		Model: e.config.Model,
		Messages: []OpenAIMessage{
			{Role: "system", Content: "You are an expert in compliance frameworks and security standards. Your task is to analyze and improve structured data extracted from PDF documents."},
			{Role: "user", Content: prompt},
		},
		Temperature: e.config.Temperature,
		MaxTokens:   e.config.MaxTokens,
	}
	
	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", e.config.Endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.config.APIKey)
	
	resp, err := e.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	var openAIResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	
	if openAIResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", openAIResp.Error.Message)
	}
	
	if len(openAIResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}
	
	return openAIResp.Choices[0].Message.Content, nil
}

// EnhanceSegmentation improves segmentation results
func (e *OpenAIEnhancer) EnhanceSegmentation(ctx context.Context, doc *types.SegmentedDocument) (*types.EnhancementResult, error) {
	// Build prompt
	prompt := fmt.Sprintf(`Review this document segmentation and suggest improvements:

Document: %s
Categories: %d
Total Guidelines: %d

Please analyze:
1. Are all categories properly identified?
2. Are guidelines correctly nested?
3. Should any guidelines be merged or split?
4. Are IDs formatted consistently?

Respond with JSON containing:
- confidence: 0-1 score
- issues: list of identified problems
- suggestions: list of improvements`,
		doc.DocumentMetadata.Title,
		len(doc.Categories),
		e.countGuidelines(doc))
	
	response, err := e.callOpenAI(ctx, prompt)
	if err != nil {
		return nil, err
	}
	
	// Parse response and create enhancement result
	result := &types.EnhancementResult{
		OriginalData: doc,
		EnhancedData: doc,
		Changes:      []types.EnhancementChange{},
		Confidence:   0.8,
		Provider:     e.Name(),
		Model:        e.config.Model,
		Timestamp:    time.Now(),
	}
	
	// TODO: Parse JSON response and extract actual changes
	// For now, just return the response as a change
	result.Changes = append(result.Changes, types.EnhancementChange{
		Path:       "segmentation",
		Type:       "modify",
		NewValue:   response,
		Reason:     "LLM analysis",
		Confidence: 0.8,
	})
	
	return result, nil
}

// ValidateMetadata validates and enriches metadata
func (e *OpenAIEnhancer) ValidateMetadata(ctx context.Context, meta *types.DocumentMetadata) (*types.EnhancementResult, error) {
	prompt := fmt.Sprintf(`Review and improve this document metadata:

Title: %s
Author: %s
Version: %s
Description: %s

Tasks:
1. Validate accuracy
2. Improve description if generic
3. Suggest document type (Standard/Regulation/Framework/Best Practice)
4. Identify applicable jurisdictions and industry sectors

Respond with JSON containing validated and enhanced metadata.`,
		meta.Title, meta.Author, meta.Version, meta.Description)
	
	response, err := e.callOpenAI(ctx, prompt)
	if err != nil {
		return nil, err
	}
	
	result := &types.EnhancementResult{
		OriginalData: meta,
		EnhancedData: meta,
		Changes:      []types.EnhancementChange{},
		Confidence:   0.85,
		Provider:     e.Name(),
		Model:        e.config.Model,
		Timestamp:    time.Now(),
	}
	
	result.Changes = append(result.Changes, types.EnhancementChange{
		Path:       "metadata",
		Type:       "modify",
		NewValue:   response,
		Reason:     "LLM validation",
		Confidence: 0.85,
	})
	
	return result, nil
}

// EnhanceGuideline improves individual guideline quality
func (e *OpenAIEnhancer) EnhanceGuideline(ctx context.Context, guideline *types.SegmentGuideline) (*types.EnhancementResult, error) {
	prompt := fmt.Sprintf(`Improve this guideline:

ID: %s
Title: %s
Objective: %s

Tasks:
1. Extract clear objective if missing
2. Identify key recommendations
3. Suggest if should be split into parts
4. Improve title clarity

Respond with enhanced guideline structure.`,
		guideline.ID, guideline.Title, guideline.Objective)
	
	response, err := e.callOpenAI(ctx, prompt)
	if err != nil {
		return nil, err
	}
	
	result := &types.EnhancementResult{
		OriginalData: guideline,
		EnhancedData: guideline,
		Changes:      []types.EnhancementChange{},
		Confidence:   0.8,
		Provider:     e.Name(),
		Model:        e.config.Model,
		Timestamp:    time.Now(),
	}
	
	result.Changes = append(result.Changes, types.EnhancementChange{
		Path:       "guideline." + guideline.ID,
		Type:       "modify",
		NewValue:   response,
		Reason:     "LLM enhancement",
		Confidence: 0.8,
	})
	
	return result, nil
}

// countGuidelines counts total guidelines in document
func (e *OpenAIEnhancer) countGuidelines(doc *types.SegmentedDocument) int {
	count := 0
	for _, cat := range doc.Categories {
		count += len(cat.Guidelines)
	}
	return count
}

// AnthropicEnhancer uses Anthropic Claude API for enhancement
type AnthropicEnhancer struct {
	EnhancerBase
	client *http.Client
}

// NewAnthropicEnhancer creates a new Anthropic enhancer
func NewAnthropicEnhancer(config types.LLMConfig) (*AnthropicEnhancer, error) {
	if config.Endpoint == "" {
		config.Endpoint = "https://api.anthropic.com/v1/messages"
	}
	if config.Model == "" {
		config.Model = "claude-3-sonnet-20240229"
	}
	if config.Temperature == 0 {
		config.Temperature = 0.3
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 2000
	}
	
	e := &AnthropicEnhancer{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
	
	if err := e.Configure(config); err != nil {
		return nil, err
	}
	
	return e, nil
}

// Name returns the enhancer name
func (e *AnthropicEnhancer) Name() string {
	return fmt.Sprintf("anthropic-%s-v1.0", e.config.Model)
}

// AnthropicRequest represents an Anthropic API request
type AnthropicRequest struct {
	Model       string              `json:"model"`
	Messages    []AnthropicMessage  `json:"messages"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature float64             `json:"temperature"`
}

// AnthropicMessage represents a message
type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnthropicResponse represents an Anthropic API response
type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// callAnthropic makes a request to the Anthropic API
func (e *AnthropicEnhancer) callAnthropic(ctx context.Context, prompt string) (string, error) {
	req := AnthropicRequest{
		Model: e.config.Model,
		Messages: []AnthropicMessage{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   e.config.MaxTokens,
		Temperature: e.config.Temperature,
	}
	
	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, "POST", e.config.Endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", e.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	
	resp, err := e.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()
	
	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}
	
	if anthropicResp.Error != nil {
		return "", fmt.Errorf("Anthropic API error: %s", anthropicResp.Error.Message)
	}
	
	if len(anthropicResp.Content) == 0 {
		return "", fmt.Errorf("no response from Anthropic")
	}
	
	return anthropicResp.Content[0].Text, nil
}

// EnhanceSegmentation, ValidateMetadata, and EnhanceGuideline follow similar patterns to OpenAI
func (e *AnthropicEnhancer) EnhanceSegmentation(ctx context.Context, doc *types.SegmentedDocument) (*types.EnhancementResult, error) {
	// Similar implementation using callAnthropic instead of callOpenAI
	return nil, fmt.Errorf("not implemented yet")
}

func (e *AnthropicEnhancer) ValidateMetadata(ctx context.Context, meta *types.DocumentMetadata) (*types.EnhancementResult, error) {
	return nil, fmt.Errorf("not implemented yet")
}

func (e *AnthropicEnhancer) EnhanceGuideline(ctx context.Context, guideline *types.SegmentGuideline) (*types.EnhancementResult, error) {
	return nil, fmt.Errorf("not implemented yet")
}

