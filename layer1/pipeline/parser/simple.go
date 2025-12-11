package parser

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ossf/gemara/layer1/pipeline/types"
)

// Pre-compiled regexes for performance
var (
	// Matches numbered headings like "1.", "1.1", "1.1.1", "1.1.1.1" followed by uppercase text
	// Also matches ALL CAPS headings
	headingRegex = regexp.MustCompile(`^([0-9]+\.)*[0-9]+\.?\s+[A-Z].*$|^[A-Z][A-Z\s]+$`)

	// Matches list item markers
	listRegex = regexp.MustCompile(`^\s*([0-9]+\.|[a-z]\.|•|\*|-)\s+`)

	// Matches empty lines
	emptyRegex = regexp.MustCompile(`^\s*$`)

	// Matches TOC lines with dotted leaders (4+ consecutive dots)
	tocDotPattern = regexp.MustCompile(`\.{4,}`)

	// Matches numbered prefix for heading level detection (e.g., "1.", "1.1.", "1.1.1")
	numberedPrefixRegex = regexp.MustCompile(`^([0-9]+\.)*[0-9]+`)

	// Matches ordered list markers
	orderedListRegex = regexp.MustCompile(`^[0-9]+\.`)
)

// SimpleParser uses pdftotext (poppler-utils) for basic PDF parsing
type SimpleParser struct {
	ParserBase
}

// NewSimpleParser creates a new simple parser
func NewSimpleParser(config types.ParserConfig) (*SimpleParser, error) {
	parser := &SimpleParser{}
	if err := parser.Configure(config); err != nil {
		return nil, err
	}
	return parser, nil
}

// Name returns the parser name
func (p *SimpleParser) Name() string {
	return "simple"
}

// Parse extracts content from a PDF file using pdftotext
func (p *SimpleParser) Parse(filePath string) (*types.ParsedDocument, error) {
	// Check if pdftotext is available
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return nil, fmt.Errorf("pdftotext not found (install poppler-utils): %w", err)
	}

	// Create temp file for text output
	tempDir := p.config.TempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	
	textFile := filepath.Join(tempDir, fmt.Sprintf("parsed-%d.txt", time.Now().Unix()))
	defer func() {
		if !p.config.KeepTempFiles {
			_ = os.Remove(textFile) // Ignore cleanup errors
		}
	}()

	// Run pdftotext with layout preservation
	cmd := exec.Command("pdftotext", "-layout", filePath, textFile)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("pdftotext failed: %w", err)
	}

	// Read extracted text
	content, err := os.ReadFile(textFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read text file: %w", err)
	}

	// Parse the text into structured blocks
	doc := &types.ParsedDocument{
		Metadata: types.ParsedMetadata{
			SourceFile: filePath,
			Parser:     "simple-v1.0",
			ParsedAt:   time.Now(),
		},
		Pages: p.parseTextContent(string(content)),
	}

	return doc, nil
}

// parseTextContent converts plain text into structured blocks
func (p *SimpleParser) parseTextContent(content string) []types.Page {
	lines := strings.Split(content, "\n")

	var pages []types.Page
	currentPage := types.Page{
		PageNumber: 1,
		Blocks:     []types.Block{},
	}
	
	var currentBlock *types.Block
	var currentText strings.Builder
	
	for _, line := range lines {
		// Detect page breaks (form feed character)
		if strings.Contains(line, "\f") {
			// Flush current block
			if currentBlock != nil && currentText.Len() > 0 {
				currentBlock.Text = strings.TrimSpace(currentText.String())
				currentPage.Blocks = append(currentPage.Blocks, *currentBlock)
			}
			
			// Save current page
			if len(currentPage.Blocks) > 0 {
				pages = append(pages, currentPage)
			}
			
			// Start new page
			currentPage = types.Page{
				PageNumber: len(pages) + 1,
				Blocks:     []types.Block{},
			}
			currentBlock = nil
			currentText.Reset()
			continue
		}
		
		// Skip empty lines
		if emptyRegex.MatchString(line) {
			// Flush current block on empty line
			if currentBlock != nil && currentText.Len() > 0 {
				currentBlock.Text = strings.TrimSpace(currentText.String())
				currentPage.Blocks = append(currentPage.Blocks, *currentBlock)
				currentBlock = nil
				currentText.Reset()
			}
			continue
		}
		
		// Skip or clean Table of Contents lines (lines with dotted leaders)
		if tocDotPattern.MatchString(line) {
			// This looks like a TOC line - skip it entirely
			continue
		}
		
		// Skip page headers, footers, copyright notices, table headers
		if isPageHeaderFooter(line) || isTableHeader(line) {
			continue
		}
		
		// Clean the line (normalize whitespace, remove TOC dots, etc.)
		line = cleanText(line)
		if line == "" {
			continue
		}
		
		// Detect headings
		if headingRegex.MatchString(strings.TrimSpace(line)) {
			// Flush previous block
			if currentBlock != nil && currentText.Len() > 0 {
				currentBlock.Text = strings.TrimSpace(currentText.String())
				currentPage.Blocks = append(currentPage.Blocks, *currentBlock)
				currentText.Reset()
			}
			
			// Create new heading block
			level := p.detectHeadingLevel(line)
			currentBlock = &types.Block{
				Type:       types.BlockTypeHeading,
				Level:      level,
				FontSize:   float64(18 - level*2),
				FontWeight: "bold",
			}
			currentText.WriteString(strings.TrimSpace(line))
			
			// Headings are usually one line, flush immediately
			currentBlock.Text = strings.TrimSpace(currentText.String())
			currentPage.Blocks = append(currentPage.Blocks, *currentBlock)
			currentBlock = nil
			currentText.Reset()
			continue
		}
		
		// Detect list items
		if matches := listRegex.FindStringSubmatch(line); matches != nil {
			// Flush previous block
			if currentBlock != nil && currentText.Len() > 0 {
				currentBlock.Text = strings.TrimSpace(currentText.String())
				currentPage.Blocks = append(currentPage.Blocks, *currentBlock)
				currentText.Reset()
			}

			// Create new list block
			listType := "unordered"
			if orderedListRegex.MatchString(matches[1]) {
				listType = "ordered"
			}
			
			currentBlock = &types.Block{
				Type: types.BlockTypeList,
				ListItem: &types.ListItem{
					Marker: matches[1],
					Type:   listType,
					Level:  p.detectIndentLevel(line),
				},
			}
			currentText.WriteString(strings.TrimSpace(line[len(matches[0]):]))
			continue
		}
		
		// Regular paragraph text
		if currentBlock == nil {
			currentBlock = &types.Block{
				Type: types.BlockTypeParagraph,
			}
		}
		
		// Append to current block
		if currentText.Len() > 0 {
			currentText.WriteString(" ")
		}
		currentText.WriteString(strings.TrimSpace(line))
	}
	
	// Flush final block
	if currentBlock != nil && currentText.Len() > 0 {
		currentBlock.Text = strings.TrimSpace(currentText.String())
		currentPage.Blocks = append(currentPage.Blocks, *currentBlock)
	}
	
	// Save final page
	if len(currentPage.Blocks) > 0 {
		pages = append(pages, currentPage)
	}
	
	return pages
}

// detectHeadingLevel determines the heading level based on formatting
func (p *SimpleParser) detectHeadingLevel(line string) int {
	// Check for numbered headings (1., 1.1, 1.1.1, etc.)
	if matches := numberedPrefixRegex.FindString(line); matches != "" {
		// Count the dots to determine level (1.1 = 2, 1.1.1 = 3, etc.)
		dots := strings.Count(matches, ".")
		level := dots + 1 // "1" = level 1, "1.1" = level 2, etc.
		if level > 0 && level <= 6 {
			return level
		}
	}

	// All caps likely level 1 or 2
	trimmed := strings.TrimSpace(line)
	if strings.ToUpper(trimmed) == trimmed {
		if len(trimmed) < 30 {
			return 1
		}
		return 2
	}

	return 3
}

// detectIndentLevel determines the indentation level
func (p *SimpleParser) detectIndentLevel(line string) int {
	leadingSpaces := len(line) - len(strings.TrimLeft(line, " \t"))
	// Assume 2-4 spaces per level
	return (leadingSpaces / 3) + 1
}

// ParseTextFile parses a plain text file (useful for testing)
func (p *SimpleParser) ParseTextFile(filePath string) (*types.ParsedDocument, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read text file: %w", err)
	}

	doc := &types.ParsedDocument{
		Metadata: types.ParsedMetadata{
			SourceFile: filePath,
			Parser:     "simple-v1.0",
			ParsedAt:   time.Now(),
		},
		Pages: p.parseTextContent(string(content)),
	}

	return doc, nil
}

// ExtractPDFText extracts text from PDF without structuring (useful for debugging)
func ExtractPDFText(filePath string) (string, error) {
	cmd := exec.Command("pdftotext", "-layout", filePath, "-")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext failed: %w", err)
	}
	return string(output), nil
}

// ExtractPDFMetadata extracts PDF metadata
func ExtractPDFMetadata(filePath string) (map[string]string, error) {
	if _, err := exec.LookPath("pdfinfo"); err != nil {
		return nil, fmt.Errorf("pdfinfo not found (install poppler-utils): %w", err)
	}

	cmd := exec.Command("pdfinfo", filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pdfinfo failed: %w", err)
	}

	metadata := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			metadata[key] = value
		}
	}

	return metadata, nil
}

// cleanTOCDots removes dotted leader patterns commonly found in tables of contents
// These patterns look like: "Chapter 1 .......... 15" or "1.1 Overview ... 23"
func cleanTOCDots(line string) string {
	// Pattern: multiple dots (3+) optionally followed by spaces and page numbers
	dotPattern := regexp.MustCompile(`\s*\.{3,}[\s\d]*$`)
	
	// Remove trailing dot patterns with page numbers
	cleaned := dotPattern.ReplaceAllString(line, "")
	
	// Also clean inline dots that separate sections
	// Pattern: space + 3+ dots + space
	inlineDotPattern := regexp.MustCompile(`\s+\.{3,}\s+`)
	cleaned = inlineDotPattern.ReplaceAllString(cleaned, " - ")
	
	return strings.TrimSpace(cleaned)
}

// isPageHeaderFooter checks if a line is a page header or footer
func isPageHeaderFooter(line string) bool {
	trimmed := strings.TrimSpace(line)
	
	// Common page footer patterns
	patterns := []string{
		`(?i)page\s+\d+`,                               // "Page 2", "page 123"
		`(?i)©\s*\d{4}`,                                // Copyright notice
		`(?i)all\s+rights\s+reserved`,                  // Rights notice
		`(?i)^\s*\d+\s*$`,                              // Just a page number
	}
	
	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, trimmed); matched {
			return true
		}
	}
	
	return false
}

// isTableHeader checks if a line appears to be a table header row
func isTableHeader(line string) bool {
	trimmed := strings.TrimSpace(line)
	
	// Common table header patterns with lots of spacing
	// e.g., "Date            Version                Description"
	tableHeaderPatterns := []string{
		`(?i)^date\s{4,}version`,                       // Document change table
		`(?i)^requirement\s{4,}testing`,               // Testing procedures table
		`(?i)^pci\s+dss\s+requirement`,                // Requirements table
		`(?i)^guidance\s{4,}`,                         // Guidance tables
	}
	
	for _, pattern := range tableHeaderPatterns {
		if matched, _ := regexp.MatchString(pattern, trimmed); matched {
			return true
		}
	}
	
	return false
}

// normalizeWhitespace collapses multiple spaces into single spaces
func normalizeWhitespace(text string) string {
	// Replace multiple spaces with single space
	multiSpacePattern := regexp.MustCompile(`\s{3,}`)
	cleaned := multiSpacePattern.ReplaceAllString(text, " ")
	
	// Clean up spacing around punctuation
	cleaned = strings.TrimSpace(cleaned)
	
	return cleaned
}

// cleanText applies all text cleaning operations
// Note: isPageHeaderFooter and isTableHeader checks are done in parseTextContent
// before this function is called, so we don't duplicate them here
func cleanText(text string) string {
	// Skip if empty
	if strings.TrimSpace(text) == "" {
		return ""
	}

	// Clean TOC dots
	cleaned := cleanTOCDots(text)

	// Normalize whitespace
	cleaned = normalizeWhitespace(cleaned)

	return cleaned
}
