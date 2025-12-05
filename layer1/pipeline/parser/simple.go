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
			os.Remove(textFile)
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
	
	// Simple heuristics for structure detection
	headingRegex := regexp.MustCompile(`^([0-9]+\.)+\s+[A-Z].*$|^[A-Z][A-Z\s]+$`)
	listRegex := regexp.MustCompile(`^\s*([0-9]+\.|[a-z]\.|•|\*|-)\s+`)
	emptyRegex := regexp.MustCompile(`^\s*$`)
	
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
			if regexp.MustCompile(`^[0-9]+\.`).MatchString(matches[1]) {
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
	// Check for numbered headings (1., 1.1., 1.1.1., etc.)
	if matches := regexp.MustCompile(`^([0-9]+\.)+`).FindString(line); matches != "" {
		dots := strings.Count(matches, ".")
		if dots > 0 && dots <= 6 {
			return dots
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

