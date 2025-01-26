package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/takak2166/scrapbox2notion/internal/models"
)

func TestParseFile(t *testing.T) {
	// Create a temporary test file
	content := `{
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
	}`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
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

	if pages[0].Title != "Test Page" {
		t.Errorf("Expected page title 'Test Page', got '%s'", pages[0].Title)
	}
}

func TestConvertToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		page     models.Page
		expected string
	}{
		{
			name: "Basic page with title and content",
			page: models.Page{
				Title: "Test Page",
				Lines: []models.Line{
					{Text: "Test Page"},
					{Text: "This is a test"},
				},
			},
			expected: "# Test Page\n\nThis is a test\n",
		},
		{
			name: "Page with indentation",
			page: models.Page{
				Title: "Indented List",
				Lines: []models.Line{
					{Text: "Indented List"},
					{Text: " Item 1"},
					{Text: "  Item 1.1"},
					{Text: " Item 2"},
				},
			},
			expected: "# Indented List\n\n- Item 1\n  - Item 1.1\n- Item 2\n",
		},
		{
			name: "Page with formatting",
			page: models.Page{
				Title: "Formatted Text",
				Lines: []models.Line{
					{Text: "Formatted Text"},
					{Text: "[* Bold text]"},
					{Text: "[/ Italic text]"},
					{Text: "[- Strikethrough]"},
					{Text: "[$ E = mc^2]"},
					{Text: "code:test.js"},
					{Text: " console.log('hello')"},
					{Text: "[** h4 text]"},
					{Text: "[*** h3 text]"},
					{Text: "[**** h2 text]"},
				},
			},
			expected: "# Formatted Text\n\n**Bold text**\n_Italic text_\n~~Strikethrough~~\n$E = mc^2$\n```test.js\nconsole.log('hello')\n```\n#### h4 text\n### h3 text\n## h2 text\n",
		},
		{
			name: "Page with links",
			page: models.Page{
				Title: "Links",
				Lines: []models.Line{
					{Text: "Links"},
					{Text: "http://example.com"},
					{Text: "http://example.com/image.jpg"},
					{Text: "[Another Page]"},
				},
				LinksLc: []string{"another_page"},
			},
			expected: "# Links\n\nhttp://example.com\n![image](http://example.com/image.jpg)\n[Another Page](./another_page.md)\n",
		},
	}

	p := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.ConvertToMarkdown(&tt.page)
			if result != tt.expected {
				t.Errorf("ConvertToMarkdown() = %v, want %v", result, tt.expected)
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
