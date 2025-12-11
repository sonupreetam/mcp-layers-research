package llm

import (
	"context"
	"fmt"
	"time"

	"github.com/ossf/gemara/layer1/pipeline/types"
)

// Enhancer provides LLM-based enhancement capabilities
type Enhancer interface {
	// EnhanceSegmentation improves segmentation results
	EnhanceSegmentation(ctx context.Context, doc *types.SegmentedDocument) (*types.EnhancementResult, error)
	
	// ValidateMetadata validates and enriches metadata
	ValidateMetadata(ctx context.Context, meta *types.DocumentMetadata) (*types.EnhancementResult, error)
	
	// EnhanceGuideline improves individual guideline quality
	EnhanceGuideline(ctx context.Context, guideline *types.SegmentGuideline) (*types.EnhancementResult, error)
	
	// Name returns the enhancer name
	Name() string
	
	// Configure sets enhancer configuration
	Configure(config types.LLMConfig) error
}

// EnhancerBase provides common LLM functionality
type EnhancerBase struct {
	config types.LLMConfig
}

// Configure sets the LLM configuration
func (e *EnhancerBase) Configure(config types.LLMConfig) error {
	e.config = config
	return nil
}

// GetConfig returns the LLM configuration
func (e *EnhancerBase) GetConfig() types.LLMConfig {
	return e.config
}

// NewEnhancer creates an enhancer based on provider
func NewEnhancer(config types.LLMConfig) (Enhancer, error) {
	switch config.Provider {
	case "openai":
		return NewOpenAIEnhancer(config)
	case "anthropic":
		return NewAnthropicEnhancer(config)
	case "mock":
		return NewMockEnhancer(config)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", config.Provider)
	}
}

// EnhancementTask represents a specific enhancement task
type EnhancementTask struct {
	Type        string      `json:"type"` // "metadata", "guideline", "segmentation"
	Input       interface{} `json:"input"`
	Prompt      string      `json:"prompt"`
	Temperature float64     `json:"temperature"`
	MaxTokens   int         `json:"max_tokens"`
}

// MockEnhancer provides a mock implementation for testing
type MockEnhancer struct {
	EnhancerBase
}

// NewMockEnhancer creates a new mock enhancer
func NewMockEnhancer(config types.LLMConfig) (*MockEnhancer, error) {
	e := &MockEnhancer{}
	if err := e.Configure(config); err != nil {
		return nil, err
	}
	return e, nil
}

// Name returns the enhancer name
func (e *MockEnhancer) Name() string {
	return "mock-v1.0"
}

// EnhanceSegmentation provides mock enhancement
func (e *MockEnhancer) EnhanceSegmentation(ctx context.Context, doc *types.SegmentedDocument) (*types.EnhancementResult, error) {
	result := &types.EnhancementResult{
		OriginalData: doc,
		EnhancedData: doc,
		Changes:      []types.EnhancementChange{},
		Confidence:   0.95,
		Provider:     e.Name(),
		Model:        "mock",
		Timestamp:    time.Now(),
	}
	
	// Mock: Add a change to show enhancement happened
	result.Changes = append(result.Changes, types.EnhancementChange{
		Path:       "metadata.description",
		Type:       "modify",
		OldValue:   doc.DocumentMetadata.Description,
		NewValue:   doc.DocumentMetadata.Description + " (Enhanced)",
		Reason:     "Mock enhancement for testing",
		Confidence: 0.95,
	})
	
	return result, nil
}

// ValidateMetadata provides mock validation
func (e *MockEnhancer) ValidateMetadata(ctx context.Context, meta *types.DocumentMetadata) (*types.EnhancementResult, error) {
	result := &types.EnhancementResult{
		OriginalData: meta,
		EnhancedData: meta,
		Changes:      []types.EnhancementChange{},
		Confidence:   0.95,
		Provider:     e.Name(),
		Model:        "mock",
		Timestamp:    time.Now(),
	}
	
	return result, nil
}

// EnhanceGuideline provides mock enhancement
func (e *MockEnhancer) EnhanceGuideline(ctx context.Context, guideline *types.SegmentGuideline) (*types.EnhancementResult, error) {
	result := &types.EnhancementResult{
		OriginalData: guideline,
		EnhancedData: guideline,
		Changes:      []types.EnhancementChange{},
		Confidence:   0.95,
		Provider:     e.Name(),
		Model:        "mock",
		Timestamp:    time.Now(),
	}
	
	return result, nil
}

// PromptTemplates contains prompts for different enhancement tasks
type PromptTemplates struct {
	MetadataValidation   string
	GuidelineEnhancement string
	SegmentationReview   string
	ObjectiveExtraction  string
}

// DefaultPrompts provides default prompt templates
var DefaultPrompts = PromptTemplates{
	MetadataValidation: `Review the following document metadata extracted from a PDF:

Title: {{.Title}}
Author: {{.Author}}
Version: {{.Version}}
Description: {{.Description}}

Tasks:
1. Validate that the metadata is accurate and complete
2. Suggest improvements to the description if it's too generic
3. Identify any missing fields that should be extracted
4. Determine the most appropriate document type (Standard, Regulation, Framework, Best Practice)

Respond with a JSON object containing:
- validated_metadata: corrected/enhanced metadata
- confidence: 0-1 score
- suggested_changes: list of changes with reasons`,

	GuidelineEnhancement: `Review the following guideline extracted from a compliance/security framework:

ID: {{.ID}}
Title: {{.Title}}
Objective: {{.Objective}}
Text: {{.Text}}

Tasks:
1. Extract a clear, concise objective if missing or unclear
2. Identify key recommendations from the text
3. Determine if this should be split into multiple parts
4. Suggest better title if current one is unclear

Respond with a JSON object containing:
- enhanced_guideline: improved guideline structure
- confidence: 0-1 score
- changes: list of changes made`,

	SegmentationReview: `Review the document segmentation:

Categories: {{.CategoryCount}}
Guidelines: {{.GuidelineCount}}

Issues to check:
1. Are categories properly identified?
2. Are guidelines correctly nested under categories?
3. Are there any orphaned or misplaced guidelines?
4. Should any guidelines be merged or split?

Respond with suggested corrections and confidence score.`,

	ObjectiveExtraction: `Extract the primary objective from this text:

{{.Text}}

The objective should be:
- Clear and concise (1-2 sentences)
- Focused on the "what" and "why"
- Written in active voice

Respond with only the extracted objective text.`,
}

// EnhancementHook allows custom enhancement logic
type EnhancementHook func(ctx context.Context, data interface{}) (interface{}, error)

// EnhancementPipeline chains multiple enhancement steps
type EnhancementPipeline struct {
	enhancer Enhancer
	hooks    []EnhancementHook
}

// NewEnhancementPipeline creates a new enhancement pipeline
func NewEnhancementPipeline(enhancer Enhancer) *EnhancementPipeline {
	return &EnhancementPipeline{
		enhancer: enhancer,
		hooks:    []EnhancementHook{},
	}
}

// AddHook adds a custom enhancement hook
func (p *EnhancementPipeline) AddHook(hook EnhancementHook) {
	p.hooks = append(p.hooks, hook)
}

// Process runs data through the enhancement pipeline
func (p *EnhancementPipeline) Process(ctx context.Context, data interface{}) (interface{}, error) {
	current := data
	var err error
	
	for i, hook := range p.hooks {
		current, err = hook(ctx, current)
		if err != nil {
			return nil, fmt.Errorf("hook %d failed: %w", i, err)
		}
	}
	
	return current, nil
}


