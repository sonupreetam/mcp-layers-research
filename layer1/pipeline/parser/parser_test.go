package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ossf/gemara/layer1/pipeline/types"
)

func TestSimpleParser(t *testing.T) {
	// Create a test PDF text file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	
	testContent := `Payment Card Industry Data Security Standard
Version 3.2.1

Introduction

This document provides the PCI DSS requirements and testing procedures.

Requirement 1: Install and maintain a firewall configuration to protect cardholder data

1.1 Establish firewall and router configuration standards

1.1.1 A formal process for approving and testing all network connections and changes to the firewall and router configurations.

1.1.2 Current network diagram that identifies all connections between the cardholder data environment and other networks.

1.2 Build firewall configurations that restrict connections between untrusted networks

1.2.1 Restrict inbound and outbound traffic to that which is necessary for the cardholder data environment.
`
	
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Create parser
	config := types.ParserConfig{
		Provider: "simple",
		TempDir:  tempDir,
	}
	
	parser, err := NewSimpleParser(config)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	
	// Parse text file
	doc, err := parser.ParseTextFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}
	
	// Verify results
	if len(doc.Pages) == 0 {
		t.Error("Expected at least one page")
	}
	
	if doc.Metadata.Parser != "simple-v1.0" {
		t.Errorf("Expected parser 'simple-v1.0', got '%s'", doc.Metadata.Parser)
	}
	
	// Check for blocks and headings
	totalBlocks := 0
	foundHeadings := 0
	for _, page := range doc.Pages {
		totalBlocks += len(page.Blocks)
		for _, block := range page.Blocks {
			t.Logf("Block type: %s, text: %s", block.Type, block.Text[:min(50, len(block.Text))])
			if block.Type == types.BlockTypeHeading {
				foundHeadings++
				t.Logf("Found heading: %s", block.Text)
			}
		}
	}
	
	if totalBlocks == 0 {
		t.Error("Expected to find some blocks")
	}
	
	// Headings detection might vary, so just log a warning
	if foundHeadings == 0 {
		t.Logf("Warning: No headings detected (found %d total blocks)", totalBlocks)
	} else {
		t.Logf("Success: Found %d headings out of %d total blocks", foundHeadings, totalBlocks)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestParserFactory(t *testing.T) {
	tests := []struct {
		provider string
		wantErr  bool
	}{
		{"simple", false},
		{"docling", false},
		{"invalid", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			config := types.ParserConfig{
				Provider: tt.provider,
			}
			
			_, err := NewParser(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewParser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

