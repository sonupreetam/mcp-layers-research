package parser

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ossf/gemara/layer1/pipeline/types"
)

// DoclingParser uses docling Python library directly for PDF parsing
type DoclingParser struct {
	ParserBase
	scriptPath string
}

// NewDoclingParser creates a new Docling parser
func NewDoclingParser(config types.ParserConfig) (*DoclingParser, error) {
	parser := &DoclingParser{}
	if err := parser.Configure(config); err != nil {
		return nil, err
	}

	// Find the Python script path relative to this Go file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("failed to get current file path")
	}
	parser.scriptPath = filepath.Join(filepath.Dir(filename), "docling_convert.py")

	return parser, nil
}

// Name returns the parser name
func (p *DoclingParser) Name() string {
	return "docling"
}

// DoclingConvertResponse represents the response from the Python script
type DoclingConvertResponse struct {
	Status   string          `json:"status"`
	Document DoclingDocument `json:"document"`
	Errors   []DoclingError  `json:"errors"`
}

// DoclingError represents an error from docling
type DoclingError struct {
	ErrorMessage string `json:"error_message"`
}

// DoclingDocument represents the converted document
type DoclingDocument struct {
	Name   string                     `json:"name"`
	Texts  []DoclingTextItem          `json:"texts"`
	Tables []DoclingTable             `json:"tables"`
	Pages  map[string]DoclingPageInfo `json:"pages"`
}

// DoclingTextItem represents a text element
type DoclingTextItem struct {
	SelfRef    string        `json:"self_ref"`
	Label      string        `json:"label"`
	Text       string        `json:"text"`
	Prov       []DoclingProv `json:"prov"`
	Level      int           `json:"level,omitempty"`
	Marker     string        `json:"marker,omitempty"`
	Enumerated bool          `json:"enumerated,omitempty"`
}

// DoclingProv contains provenance info (page/bbox)
type DoclingProv struct {
	PageNo int         `json:"page_no"`
	BBox   DoclingBBox `json:"bbox"`
}

// DoclingBBox represents a bounding box
type DoclingBBox struct {
	L float64 `json:"l"`
	T float64 `json:"t"`
	R float64 `json:"r"`
	B float64 `json:"b"`
}

// DoclingTable represents a table
type DoclingTable struct {
	SelfRef string        `json:"self_ref"`
	Label   string        `json:"label"`
	Prov    []DoclingProv `json:"prov"`
}

// DoclingPageInfo contains page dimensions
type DoclingPageInfo struct {
	PageNo int         `json:"page_no"`
	Size   DoclingSize `json:"size"`
}

// DoclingSize represents page dimensions
type DoclingSize struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Parse extracts content from a PDF file using docling Python library
func (p *DoclingParser) Parse(filePath string) (*types.ParsedDocument, error) {
	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Run the Python script
	cmd := exec.Command("python3", p.scriptPath, absPath)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("docling conversion failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to run docling: %w", err)
	}

	// Parse JSON output
	var resp DoclingConvertResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse docling output: %w", err)
	}

	if resp.Status != "success" {
		errMsgs := ""
		for _, e := range resp.Errors {
			errMsgs += e.ErrorMessage + "; "
		}
		return nil, fmt.Errorf("docling conversion failed: %s", errMsgs)
	}

	// Convert to ParsedDocument
	doc := p.convertDocument(filePath, &resp.Document)
	return doc, nil
}

// convertDocument converts DoclingDocument to ParsedDocument
func (p *DoclingParser) convertDocument(filePath string, docling *DoclingDocument) *types.ParsedDocument {
	doc := &types.ParsedDocument{
		Metadata: types.ParsedMetadata{
			SourceFile: filePath,
			Parser:     "docling-v2.0",
			ParsedAt:   time.Now(),
		},
		Pages: []types.Page{},
	}

	// Group blocks by page
	pageBlocks := make(map[int][]types.Block)

	for _, text := range docling.Texts {
		block := p.convertTextItem(&text)

		// Get page number from provenance
		pageNo := 1
		if len(text.Prov) > 0 {
			pageNo = text.Prov[0].PageNo
		}

		pageBlocks[pageNo] = append(pageBlocks[pageNo], block)
	}

	// Convert tables
	for _, table := range docling.Tables {
		block := types.Block{
			Type: types.BlockTypeTable,
			Text: "[Table]",
		}

		pageNo := 1
		if len(table.Prov) > 0 {
			pageNo = table.Prov[0].PageNo
			block.BBox = &types.BBox{
				X1: table.Prov[0].BBox.L,
				Y1: table.Prov[0].BBox.T,
				X2: table.Prov[0].BBox.R,
				Y2: table.Prov[0].BBox.B,
			}
		}

		pageBlocks[pageNo] = append(pageBlocks[pageNo], block)
	}

	// Find max page number
	maxPage := 0
	for pageNo := range pageBlocks {
		if pageNo > maxPage {
			maxPage = pageNo
		}
	}

	for _, pageInfo := range docling.Pages {
		if pageInfo.PageNo > maxPage {
			maxPage = pageInfo.PageNo
		}
	}

	// Create pages
	for i := 1; i <= maxPage; i++ {
		page := types.Page{
			PageNumber: i,
			Blocks:     pageBlocks[i],
		}
		doc.Pages = append(doc.Pages, page)
	}

	return doc
}

// convertTextItem converts a DoclingTextItem to a Block
func (p *DoclingParser) convertTextItem(item *DoclingTextItem) types.Block {
	block := types.Block{
		Text: item.Text,
	}

	// Convert label to block type
	switch item.Label {
	case "title", "section_header":
		block.Type = types.BlockTypeHeading
		block.Level = item.Level
		if block.Level == 0 {
			block.Level = 1
		}
		block.FontWeight = "bold"
	case "paragraph", "text":
		block.Type = types.BlockTypeParagraph
	case "list_item":
		block.Type = types.BlockTypeList
		listType := "unordered"
		if item.Enumerated {
			listType = "ordered"
		}
		block.ListItem = &types.ListItem{
			Marker: item.Marker,
			Type:   listType,
			Level:  1,
		}
	case "caption":
		block.Type = types.BlockTypeParagraph
	case "page_header", "page_footer":
		block.Type = types.BlockTypeParagraph
	case "code":
		block.Type = types.BlockTypeCode
	default:
		block.Type = types.BlockTypeParagraph
	}

	// Add bounding box if available
	if len(item.Prov) > 0 {
		prov := item.Prov[0]
		block.BBox = &types.BBox{
			X1: prov.BBox.L,
			Y1: prov.BBox.T,
			X2: prov.BBox.R,
			Y2: prov.BBox.B,
		}
	}

	return block
}
