package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFile(t *testing.T) {
	testCases := map[string]struct {
		content       string
		expectedTitle string
		expectedTags  []string
	}{
		"Page without tags": {
			content: `{
				"name": "test",
				"displayName": "Test Project",
				"exported": 1681398816,
				"pages": [
					{
						"title": "Test Page",
						"created": 1543523476,
						"updated": 1681397964,
						"lines": [
							{
								"text": "Test Page",
								"created": 1543523476,
								"updated": 1543523682,
								"userId": "user1"
							},
							{
								"text": "This is a test",
								"created": 1543523697,
								"updated": 1651583814,
								"userId": "user1"
							}
						],
						"linksLc": ["test"]
					}
				]
			}`,
			expectedTitle: "Test Page",
			expectedTags:  []string{},
		},
		"Page with one tag": {
			content: `{
				"name": "test",
				"displayName": "Test Project",
				"exported": 1681398816,
				"pages": [
					{
						"title": "Test Page with One Tag",
						"created": 1543523476,
						"updated": 1681397964,
						"lines": [
							{
								"text": "Test Page with One Tag",
								"created": 1543523476,
								"updated": 1543523682,
								"userId": "user1"
							},
							{
								"text": "#tag1",
								"created": 1543523697,
								"updated": 1651583814,
								"userId": "user1"
							}
						],
						"linksLc": ["test"]
					}
				]
			}`,
			expectedTitle: "Test Page with One Tag",
			expectedTags:  []string{"tag1"},
		},
		"Page with two tags": {
			content: `{
				"name": "test",
				"displayName": "Test Project",
				"exported": 1681398816,
				"pages": [
					{
						"title": "Test Page with Two Tags",
						"created": 1543523476,
						"updated": 1681397964,
						"lines": [
							{
								"text": "Test Page with Two Tags",
								"created": 1543523476,
								"updated": 1543523682,
								"userId": "user1"
							},
							{
								"text": "#tag1 #tag2",
								"created": 1543523697,
								"updated": 1651583814,
								"userId": "user1"
							}
						],
						"linksLc": ["test"]
					}
				]
			}`,
			expectedTitle: "Test Page with Two Tags",
			expectedTags:  []string{"tag1", "tag2"},
		},
		"Page with two tags and extra spaces": {
			content: `{
				"name": "test",
				"displayName": "Test Project",
				"exported": 1681398816,
				"pages": [
					{
						"title": "Test Page with Two Tags and Extra Spaces",
						"created": 1543523476,
						"updated": 1681397964,
						"lines": [
							{
								"text": "Test Page with Two Tags and Extra Spaces",
								"created": 1543523476,
								"updated": 1543523682,
								"userId": "user1"
							},
							{
								"text": "#tag1 text",
								"created": 1543523697,
								"updated": 1651583814,
								"userId": "user1"
							},
							{
								"text": "text #tag2",
								"created": 1543523697,
								"updated": 1651583814,
								"userId": "user1"
							}
						],
						"linksLc": ["test"]
					}
				]
			}`,
			expectedTitle: "Test Page with Two Tags and Extra Spaces",
			expectedTags:  []string{"tag1", "tag2"},
		},
	}

	for name, tt := range testCases {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.json")
			if err := os.WriteFile(tmpFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			p := New()
			if err := p.ParseFile(tmpFile); err != nil {
				t.Errorf("ParseFile() error = %v", err)
			}

			pages := p.GetPages()
			if len(pages) != 1 {
				t.Errorf("Expected 1 page, got %d", len(pages))
			}

			if pages[0].Title != tt.expectedTitle {
				t.Errorf("Expected page title '%s', got '%s'", tt.expectedTitle, pages[0].Title)
			}

			if len(pages[0].Tags) != len(tt.expectedTags) {
				t.Errorf("Expected tags %v, got %v", tt.expectedTags, pages[0].Tags)
			} else {
				for i, tag := range tt.expectedTags {
					if pages[0].Tags[i] != tag {
						t.Errorf("Expected tag '%s', got '%s'", tag, pages[0].Tags[i])
					}
				}
			}
		})
	}
}

func TestConvertToMarkdown(t *testing.T) {
	// Load test cases from testfiles/output directory
	testCases := []struct {
		inputFile   string
		outputFiles []string
	}{
		{
			"takak_20250125_051047.json",
			[]string{"Test Page1.md", "Test Page2.md"},
		},
	}

	p := New()
	for _, tc := range testCases {
		t.Run(tc.inputFile, func(t *testing.T) {
			// Get test file path
			filePath := filepath.Join("..", "..", "testfiles", "input", tc.inputFile)

			// Parse JSON file
			if err := p.ParseFile(filePath); err != nil {
				t.Fatalf("Failed to parse file: %v", err)
			}

			// Read and combine expected markdown files
			var expected strings.Builder
			for _, outputFile := range tc.outputFiles {
				expectedFilePath := filepath.Join("..", "..", "testfiles", "output", outputFile)
				content, err := os.ReadFile(expectedFilePath)
				if err != nil {
					t.Fatalf("Failed to read expected file: %v", err)
				}
				expected.WriteString(string(content))
				expected.WriteString("\n\n")
			}

			// Convert all pages to markdown and combine
			pages := p.GetPages()
			if len(pages) == 0 {
				t.Fatal("No pages found in parsed export")
			}

			var result strings.Builder
			for _, page := range pages {
				result.WriteString(p.ConvertToMarkdown(&page))
				result.WriteString("\n\n")
			}

			// Compare results
			if result.String() != expected.String() {
				t.Errorf("Expected and actual results do not match\nExpected:\n%s\n\nActual:\n%s", expected.String(), result.String())
			}
		})
	}
}

func TestConvertLineToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		links    []string
		expected string
	}{
		{
			name:     "Empty line",
			line:     "",
			expected: "",
		},
		{
			name:     "Simple text",
			line:     "Hello world",
			expected: "Hello world",
		},
		{
			name:     "Indented text",
			line:     " Indented",
			expected: "- Indented",
		},
		{
			name:     "Double indented text",
			line:     "  Double indented",
			expected: "  - Double indented",
		},
		{
			name:     "Bold text",
			line:     "[* Bold text]",
			expected: "**Bold text**",
		},
		{
			name:     "Italic text",
			line:     "[/ Italic text]",
			expected: "_Italic text_",
		},
		{
			name:     "h4 text",
			line:     "[** h4 text]",
			expected: "#### h4 text",
		},
		{
			name:     "h3 text",
			line:     "[*** h3 text]",
			expected: "### h3 text",
		},
		{
			name:     "h2 text",
			line:     "[**** h2 text]",
			expected: "## h2 text",
		},
		{
			name:     "Page link",
			line:     "[Test Page]",
			links:    []string{"test_page"},
			expected: "[Test Page](./test_page.md)",
		},
	}

	p := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.convertLineToMarkdown(tt.line, tt.links)
			if result != tt.expected {
				t.Errorf("convertLineToMarkdown() = %v, want %v", result, tt.expected)
			}
		})
	}
}
