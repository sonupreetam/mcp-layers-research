package llm

import (
	"context"
	"testing"
	"time"

	"github.com/ossf/gemara/layer1/pipeline/types"
)

func TestMockEnhancer(t *testing.T) {
	config := types.LLMConfig{
		Provider: "mock",
	}
	
	enhancer, err := NewMockEnhancer(config)
	if err != nil {
		t.Fatalf("Failed to create mock enhancer: %v", err)
	}
	
	if enhancer.Name() != "mock-v1.0" {
		t.Errorf("Expected name 'mock-v1.0', got '%s'", enhancer.Name())
	}
}

func TestMockEnhancerSegmentation(t *testing.T) {
	config := types.LLMConfig{
		Provider: "mock",
	}
	
	enhancer, err := NewMockEnhancer(config)
	if err != nil {
		t.Fatalf("Failed to create enhancer: %v", err)
	}
	
	doc := &types.SegmentedDocument{
		Metadata: types.SegmentedMetadata{
			DocumentID: "test",
		},
		DocumentMetadata: types.DocumentMetadata{
			ID:          "TEST",
			Title:       "Test",
			Description: "Original description",
			Author:      "Test",
		},
		Categories: []types.SegmentCategory{},
	}
	
	ctx := context.Background()
	result, err := enhancer.EnhanceSegmentation(ctx, doc)
	if err != nil {
		t.Fatalf("Enhancement failed: %v", err)
	}
	
	if result.Provider != "mock-v1.0" {
		t.Errorf("Expected provider 'mock-v1.0', got '%s'", result.Provider)
	}
	
	if result.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", result.Confidence)
	}
	
	if len(result.Changes) == 0 {
		t.Error("Expected at least one change")
	}
}

func TestMockEnhancerMetadata(t *testing.T) {
	config := types.LLMConfig{
		Provider: "mock",
	}
	
	enhancer, err := NewMockEnhancer(config)
	if err != nil {
		t.Fatalf("Failed to create enhancer: %v", err)
	}
	
	meta := &types.DocumentMetadata{
		ID:          "TEST",
		Title:       "Test Document",
		Description: "Test description",
		Author:      "Test Author",
	}
	
	ctx := context.Background()
	result, err := enhancer.ValidateMetadata(ctx, meta)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}
	
	if result.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", result.Confidence)
	}
}

func TestMockEnhancerGuideline(t *testing.T) {
	config := types.LLMConfig{
		Provider: "mock",
	}
	
	enhancer, err := NewMockEnhancer(config)
	if err != nil {
		t.Fatalf("Failed to create enhancer: %v", err)
	}
	
	guideline := &types.SegmentGuideline{
		ID:        "TEST-1",
		Title:     "Test Guideline",
		Objective: "Test objective",
	}
	
	ctx := context.Background()
	result, err := enhancer.EnhanceGuideline(ctx, guideline)
	if err != nil {
		t.Fatalf("Enhancement failed: %v", err)
	}
	
	if result.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestEnhancerFactory(t *testing.T) {
	tests := []struct {
		provider string
		wantErr  bool
	}{
		{"mock", false},
		{"openai", false},
		{"anthropic", false},
		{"unknown", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			config := types.LLMConfig{
				Provider: tt.provider,
				APIKey:   "test-key",
			}
			
			_, err := NewEnhancer(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEnhancer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEnhancementPipeline(t *testing.T) {
	config := types.LLMConfig{
		Provider: "mock",
	}
	
	enhancer, err := NewMockEnhancer(config)
	if err != nil {
		t.Fatalf("Failed to create enhancer: %v", err)
	}
	
	pipeline := NewEnhancementPipeline(enhancer)
	
	// Add a test hook
	hookCalled := false
	testHook := func(ctx context.Context, data interface{}) (interface{}, error) {
		hookCalled = true
		return data, nil
	}
	
	pipeline.AddHook(testHook)
	
	ctx := context.Background()
	testData := "test"
	
	result, err := pipeline.Process(ctx, testData)
	if err != nil {
		t.Fatalf("Pipeline failed: %v", err)
	}
	
	if !hookCalled {
		t.Error("Hook was not called")
	}
	
	if result != testData {
		t.Error("Data was modified unexpectedly")
	}
}

func TestOpenAIEnhancerCreation(t *testing.T) {
	config := types.LLMConfig{
		Provider:    "openai",
		Model:       "gpt-4",
		APIKey:      "test-key",
		Temperature: 0.5,
		MaxTokens:   1000,
	}
	
	enhancer, err := NewOpenAIEnhancer(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI enhancer: %v", err)
	}
	
	if enhancer.Name() != "openai-gpt-4-v1.0" {
		t.Errorf("Expected name 'openai-gpt-4-v1.0', got '%s'", enhancer.Name())
	}
	
	// Verify defaults are set
	if enhancer.config.Endpoint == "" {
		t.Error("Expected default endpoint to be set")
	}
	
	if enhancer.config.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", enhancer.config.Temperature)
	}
}

func TestAnthropicEnhancerCreation(t *testing.T) {
	config := types.LLMConfig{
		Provider: "anthropic",
		Model:    "claude-3-sonnet-20240229",
		APIKey:   "test-key",
	}
	
	enhancer, err := NewAnthropicEnhancer(config)
	if err != nil {
		t.Fatalf("Failed to create Anthropic enhancer: %v", err)
	}
	
	if !contains(enhancer.Name(), "anthropic") {
		t.Errorf("Expected name to contain 'anthropic', got '%s'", enhancer.Name())
	}
}

func TestEnhancementResultStructure(t *testing.T) {
	result := &types.EnhancementResult{
		OriginalData: "original",
		EnhancedData: "enhanced",
		Changes: []types.EnhancementChange{
			{
				Path:       "field1",
				Type:       "modify",
				OldValue:   "old",
				NewValue:   "new",
				Reason:     "test",
				Confidence: 0.9,
			},
		},
		Confidence: 0.85,
		Provider:   "test",
		Model:      "test-model",
		Timestamp:  time.Now(),
	}
	
	if len(result.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(result.Changes))
	}
	
	if result.Changes[0].Confidence != 0.9 {
		t.Errorf("Expected change confidence 0.9, got %f", result.Changes[0].Confidence)
	}
}

func TestDefaultPrompts(t *testing.T) {
	if DefaultPrompts.MetadataValidation == "" {
		t.Error("MetadataValidation prompt is empty")
	}
	
	if DefaultPrompts.GuidelineEnhancement == "" {
		t.Error("GuidelineEnhancement prompt is empty")
	}
	
	if DefaultPrompts.SegmentationReview == "" {
		t.Error("SegmentationReview prompt is empty")
	}
	
	if DefaultPrompts.ObjectiveExtraction == "" {
		t.Error("ObjectiveExtraction prompt is empty")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || 
		   (len(s) > len(substr) && contains(s[1:], substr))
}


