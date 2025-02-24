package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/takak2166/scrapbox2notion/internal/logger"
	"github.com/takak2166/scrapbox2notion/internal/models"
)

// Parser handles the conversion from Scrapbox JSON to markdown
type Parser struct {
	export *models.ScrapboxExport
}

// New creates a new Parser instance
func New() *Parser {
	return &Parser{}
}

// ParseFile reads and parses a Scrapbox JSON export file
func (p *Parser) ParseFile(filepath string) error {
	logger.Debug("Reading Scrapbox export file", map[string]interface{}{
		"filepath": filepath,
	})

	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	p.export = &models.ScrapboxExport{}
	if err := json.Unmarshal(data, p.export); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Extract tags from each page
	for i := range p.export.Pages {
		p.extractTags(&p.export.Pages[i])
	}

	logger.Info("Successfully parsed Scrapbox export file", map[string]interface{}{
		"pages_count": len(p.export.Pages),
	})

	return nil
}

// extractTags extracts tags from page lines and stores them in the Page struct
func (p *Parser) extractTags(page *models.Page) {
	var tags []string
	for _, line := range page.Lines {
		// Split the line into words
		words := strings.Fields(line.Text)
		for _, word := range words {
			// Check if the word starts with #
			if strings.HasPrefix(word, "#") {
				tag := strings.TrimPrefix(word, "#")
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}
	}
	page.Tags = tags
}

// ConvertToMarkdown converts a Scrapbox page to markdown format
func (p *Parser) ConvertToMarkdown(page *models.Page) string {
	logger.Debug("Converting page to markdown", map[string]interface{}{
		"page_title": page.Title,
	})

	var md strings.Builder

	// Add title
	md.WriteString(fmt.Sprintf("# %s\n\n", page.Title))

	// Process lines
	var codeBlock bool
	var codeLanguage string
	var codeContent []string

	for i, line := range page.Lines {
		// Skip the title line as we've already added it
		if i == 0 && line.Text == page.Title {
			continue
		}

		// Skip tag lines as they'll be handled by Notion relations
		if strings.HasPrefix(strings.TrimSpace(line.Text), "#") {
			continue
		}

		// Handle code blocks
		if strings.HasPrefix(strings.TrimSpace(line.Text), "code:") {
			codeBlock = true
			codeLanguage = strings.TrimSpace(strings.TrimPrefix(line.Text, "code:"))
			continue
		}

		if codeBlock {
			if strings.HasPrefix(line.Text, " ") || strings.HasPrefix(line.Text, "\t") {
				codeContent = append(codeContent, strings.TrimLeft(line.Text, " \t"))
				continue
			} else {
				// End of code block
				md.WriteString(fmt.Sprintf("```%s\n%s\n```\n", codeLanguage, strings.Join(codeContent, "\n")))
				codeBlock = false
				codeContent = nil
				codeLanguage = ""
			}
		}

		// Convert line to markdown
		mdLine := p.convertLineToMarkdown(line.Text, page.LinksLc)
		if mdLine != "" {
			md.WriteString(mdLine + "\n")
		}
	}

	// Handle any remaining code block
	if codeBlock && len(codeContent) > 0 {
		md.WriteString(fmt.Sprintf("```%s\n%s\n```\n", codeLanguage, strings.Join(codeContent, "\n")))
	}

	return md.String()
}

// convertLineToMarkdown converts a single line from Scrapbox format to markdown
func (p *Parser) convertLineToMarkdown(line string, links []string) string {
	if line == "" {
		return ""
	}

	// Count leading spaces and tabs for indentation level
	indentLevel := 0
	for _, char := range line {
		if char == ' ' || char == '\t' {
			indentLevel++
		} else {
			break
		}
	}

	// Trim leading whitespace
	line = strings.TrimLeft(line, " \t")

	// Convert Scrapbox syntax to markdown
	line = p.convertSyntax(line, links)

	// Add bullet point if there was indentation
	if indentLevel > 0 {
		indent := strings.Repeat("  ", indentLevel-1)
		return indent + "- " + line
	}

	return line
}

// convertSyntax converts Scrapbox syntax to markdown
func (p *Parser) convertSyntax(text string, links []string) string {
	// Convert headings [** text] to #### text
	if strings.HasPrefix(text, "[**") {
		level := strings.Count(text[:strings.Index(text, " ")], "*")
		heading := strings.TrimPrefix(text, "["+strings.Repeat("*", level)+" ")
		heading = strings.TrimSuffix(heading, "]")

		// Map Scrapbox heading levels to Markdown heading levels
		var mdLevel int
		switch level {
		case 2: // [** text] -> #### text
			mdLevel = 4
		case 3: // [*** text] -> ### text
			mdLevel = 3
		case 4: // [**** text] -> ## text
			mdLevel = 2
		default:
			mdLevel = 4
		}

		return strings.Repeat("#", mdLevel) + " " + heading
	}

	// Convert strikethrough [- text]
	text = p.replaceEnclosed(text, "[- ", "]", "~~", "~~")

	// Convert bold [* text]
	text = p.replaceEnclosed(text, "[* ", "]", "**", "**")

	// Convert italic [/ text]
	text = p.replaceEnclosed(text, "[/ ", "]", "_", "_")

	// Convert math equations [$ text]
	text = p.replaceEnclosed(text, "[$ ", "]", "$", "$")

	// Convert backtick-quoted text
	if strings.HasPrefix(text, "`") && strings.HasSuffix(text, "`") {
		return text
	}

	// Convert page links
	text = p.convertPageLinks(text, links)

	// Convert external links
	text = p.convertExternalLinks(text)

	return text
}

// replaceEnclosed replaces text enclosed in Scrapbox syntax with markdown syntax
func (p *Parser) replaceEnclosed(text, prefix, suffix, mdPrefix, mdSuffix string) string {
	startIdx := strings.Index(text, prefix)
	if startIdx == -1 {
		return text
	}

	endIdx := strings.Index(text[startIdx:], suffix)
	if endIdx == -1 {
		return text
	}
	endIdx += startIdx

	content := text[startIdx+len(prefix) : endIdx]
	// Handle escaped backslashes in LaTeX
	if prefix == "[$ " {
		content = strings.ReplaceAll(content, "\\\\", "\\")
	}
	return text[:startIdx] + mdPrefix + content + mdSuffix + text[endIdx+1:]
}

// convertPageLinks converts Scrapbox page links to markdown links
func (p *Parser) convertPageLinks(text string, links []string) string {
	// First, handle explicit page links in the format [page title]
	startIdx := strings.Index(text, "[")
	if startIdx != -1 && !strings.HasPrefix(text[startIdx:], "[- ") &&
		!strings.HasPrefix(text[startIdx:], "[* ") &&
		!strings.HasPrefix(text[startIdx:], "[$ ") &&
		!strings.HasPrefix(text[startIdx:], "[**") &&
		!strings.HasPrefix(text[startIdx:], "[/ ") {
		endIdx := strings.Index(text[startIdx:], "]")
		if endIdx != -1 {
			endIdx += startIdx
			linkText := text[startIdx+1 : endIdx]
			linkId := strings.ToLower(strings.ReplaceAll(linkText, " ", "_"))

			// Check if this is a valid page link
			for _, link := range links {
				if strings.EqualFold(link, linkId) {
					return text[:startIdx] + fmt.Sprintf("[%s](./%s.md)", linkText, link) + text[endIdx+1:]
				}
			}
		}
	}
	return text
}

// convertExternalLinks converts external URLs to markdown links
func (p *Parser) convertExternalLinks(text string) string {
	// Handle image links
	if strings.HasPrefix(text, "http") &&
		(strings.HasSuffix(text, ".jpg") || strings.HasSuffix(text, ".png") ||
			strings.HasSuffix(text, ".gif") || strings.HasSuffix(text, ".jpeg")) {
		return fmt.Sprintf("![image](%s)", text)
	}

	return text
}

// GetPages returns all pages from the parsed export
func (p *Parser) GetPages() []models.Page {
	if p.export == nil {
		return nil
	}
	return p.export.Pages
}
