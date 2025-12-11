# Layer 1 Pipeline Quick Start

This guide shows how to convert PDF compliance documents (like PCI DSS, NIST, GDPR) into machine-readable Layer 1 Gemara format.

## Prerequisites

- Go 1.21+
- PDF document to convert

## Build the Pipeline

```bash
go build -o pipeline ./layer1/pipeline/cmd/pipeline
```

## Run Complete Pipeline

The fastest way to convert a PDF is using `run-all`:

```bash
./pipeline run-all --input path/to/your.pdf --document-id my-doc-id --segmenter generic
```

### For PCI DSS Documents

```bash
./pipeline run-all --input PCI_DSS_v3-2-1.pdf --document-id pci-dss-3.2.1 --segmenter pci-dss
```

## Step-by-Step Conversion

### 1. Parse PDF

Extract text and structure from the PDF:

```bash
./pipeline parse --input path/to/your.pdf --document-id my-doc-id
```

**Options:**
- `--parser simple` (default) - Built-in Go parser
- `--parser docling` - Python-based docling parser (requires Python)

### 2. Segment

Organize parsed content into categories and guidelines:

```bash
./pipeline segment --document-id my-doc-id --segmenter generic
```

**Available segmenters:**
| Segmenter | Use For |
|-----------|---------|
| `generic` | General compliance documents |
| `pci-dss` | PCI DSS standards |
| `nist-800-53` | NIST 800-53 controls |

### 3. Convert to Layer-1

Generate the final Layer 1 YAML/JSON output:

```bash
./pipeline convert --document-id my-doc-id --output my-document.yaml
```

**Options:**
- `--format yaml` (default) or `--format json`
- `--strict` - Enable strict schema validation (default: true)

## Optional: LLM Enhancement

Improve extraction quality using an LLM:

```bash
# Using OpenAI
export LLM_API_KEY=your-openai-key
./pipeline enhance --document-id my-doc-id --llm-provider openai

# Using Anthropic
./pipeline enhance --document-id my-doc-id --llm-provider anthropic --llm-api-key your-key
```

## Validation & Analysis

### Validate Output

Check that the output conforms to Layer 1 schema:

```bash
# Validate from storage
./pipeline validate --document-id my-doc-id

# Validate an external file
./pipeline validate --validate-file ./my-document.yaml
```

### Check Schema Coverage

Analyze what information was captured vs. what couldn't be mapped:

```bash
./pipeline coverage --document-id my-doc-id
```

## List Document Versions

View all stored versions of a processed document:

```bash
./pipeline list --document-id my-doc-id
```

## Storage Structure

By default, data is stored in `./layer1/pipeline/test-data/`:

```
test-data/
├── intermediate/
│   └── {document-id}/
│       └── v{n}/
│           ├── parsed.json          # Raw parsed output
│           ├── metadata-parsed.json
│           ├── segmented.json       # Segmented output
│           └── metadata-segmented.json
├── final/
│   └── {document-id}.yaml           # Final Layer 1 output
├── validation-reports/
│   └── {document-id}/
│       └── convert-{timestamp}.json # Validation reports
└── coverage-reports/
    └── {document-id}-{timestamp}.json
```

## Global Options

| Option | Default | Description |
|--------|---------|-------------|
| `--base-dir` | `./layer1/pipeline/test-data` | Storage directory |
| `--verbose` | false | Enable detailed output |
| `--source-version` | 0 (latest) | Use specific version |

## Example: Full Workflow

```bash
# 1. Build
go build -o pipeline ./layer1/pipeline/cmd/pipeline

# 2. Parse a PDF
./pipeline parse --input CRA_Regulation.pdf --document-id cra-2024

# 3. Segment with generic segmenter
./pipeline segment --document-id cra-2024 --segmenter generic

# 4. Convert to Layer 1 YAML
./pipeline convert --document-id cra-2024 --output cra-2024.yaml

# 5. Validate the output
./pipeline validate --document-id cra-2024

# 6. Check what was captured
./pipeline coverage --document-id cra-2024
```

## Troubleshooting

**"Parser failed"**: Ensure the PDF is text-based, not scanned images. Use `--parser docling` for better OCR support.

**"Segmentation empty"**: Try a different segmenter or use `--verbose` to see what's being parsed.

**"Validation failed"**: Run `./pipeline validate --document-id X --verbose` to see specific schema errors.