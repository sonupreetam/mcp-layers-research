#!/usr/bin/env python3
"""
Direct docling PDF conversion script.
Called by the Go pipeline to convert PDFs without needing docling-serve.
Outputs JSON to stdout.
"""

import json
import sys
from pathlib import Path


def convert_pdf(input_path: str) -> dict:
    """Convert a PDF file using docling and return structured data."""
    from docling.document_converter import DocumentConverter

    converter = DocumentConverter()
    result = converter.convert(input_path)
    doc = result.document

    # Build output structure matching what the Go parser expects
    output = {
        "status": "success",
        "document": {
            "name": Path(input_path).stem,
            "texts": [],
            "tables": [],
            "pages": {},
        },
        "errors": [],
    }

    # Extract text items
    for item in doc.texts:
        text_item = {
            "self_ref": item.self_ref if hasattr(item, "self_ref") else "",
            "label": item.label if hasattr(item, "label") else "paragraph",
            "text": item.text if hasattr(item, "text") else str(item),
            "prov": [],
        }

        # Add provenance (page/bbox info) if available
        if hasattr(item, "prov") and item.prov:
            for p in item.prov:
                prov_item = {"page_no": p.page_no if hasattr(p, "page_no") else 1}
                if hasattr(p, "bbox"):
                    prov_item["bbox"] = {
                        "l": p.bbox.l if hasattr(p.bbox, "l") else 0,
                        "t": p.bbox.t if hasattr(p.bbox, "t") else 0,
                        "r": p.bbox.r if hasattr(p.bbox, "r") else 0,
                        "b": p.bbox.b if hasattr(p.bbox, "b") else 0,
                    }
                text_item["prov"].append(prov_item)

        # Add level for headings
        if hasattr(item, "level"):
            text_item["level"] = item.level

        # Add marker for list items
        if hasattr(item, "marker"):
            text_item["marker"] = item.marker
        if hasattr(item, "enumerated"):
            text_item["enumerated"] = item.enumerated

        output["document"]["texts"].append(text_item)

    # Extract tables
    for table in doc.tables:
        table_item = {
            "self_ref": table.self_ref if hasattr(table, "self_ref") else "",
            "label": "table",
            "prov": [],
        }

        if hasattr(table, "prov") and table.prov:
            for p in table.prov:
                prov_item = {"page_no": p.page_no if hasattr(p, "page_no") else 1}
                if hasattr(p, "bbox"):
                    prov_item["bbox"] = {
                        "l": p.bbox.l if hasattr(p.bbox, "l") else 0,
                        "t": p.bbox.t if hasattr(p.bbox, "t") else 0,
                        "r": p.bbox.r if hasattr(p.bbox, "r") else 0,
                        "b": p.bbox.b if hasattr(p.bbox, "b") else 0,
                    }
                table_item["prov"].append(prov_item)

        output["document"]["tables"].append(table_item)

    # Extract page info
    if hasattr(doc, "pages"):
        for page_key, page in doc.pages.items():
            output["document"]["pages"][str(page_key)] = {
                "page_no": page.page_no if hasattr(page, "page_no") else int(page_key),
                "size": {
                    "width": page.size.width if hasattr(page, "size") else 0,
                    "height": page.size.height if hasattr(page, "size") else 0,
                },
            }

    return output


def main():
    if len(sys.argv) != 2:
        print(json.dumps({"status": "error", "errors": [{"error_message": "Usage: docling_convert.py <pdf_path>"}]}))
        sys.exit(1)

    input_path = sys.argv[1]

    if not Path(input_path).exists():
        print(json.dumps({"status": "error", "errors": [{"error_message": f"File not found: {input_path}"}]}))
        sys.exit(1)

    try:
        result = convert_pdf(input_path)
        print(json.dumps(result))
    except Exception as e:
        print(json.dumps({"status": "error", "errors": [{"error_message": str(e)}]}))
        sys.exit(1)


if __name__ == "__main__":
    main()


