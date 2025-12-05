package parser

import (
	"fmt"

	"github.com/ossf/gemara/layer1/pipeline/types"
)

// Parser is the interface that PDF parsers must implement
type Parser interface {
	// Parse extracts content from a PDF file
	Parse(filePath string) (*types.ParsedDocument, error)
	
	// Name returns the parser name
	Name() string
	
	// Configure sets parser-specific options
	Configure(config types.ParserConfig) error
}

// NewParser creates a parser based on the provider
func NewParser(config types.ParserConfig) (Parser, error) {
	switch config.Provider {
	case "docling":
		return NewDoclingParser(config)
	case "simple":
		return NewSimpleParser(config)
	default:
		return nil, fmt.Errorf("unsupported parser provider: %s", config.Provider)
	}
}

// ParserBase provides common functionality for all parsers
type ParserBase struct {
	config types.ParserConfig
}

// Configure sets the parser configuration
func (p *ParserBase) Configure(config types.ParserConfig) error {
	p.config = config
	return nil
}

// GetConfig returns the parser configuration
func (p *ParserBase) GetConfig() types.ParserConfig {
	return p.config
}

