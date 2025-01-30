package notion

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jomei/notionapi"
	"github.com/takak2166/scrapbox2notion/internal/logger"
)

// Client wraps the Notion API client
type Client struct {
	client     NotionClient
	parentID   notionapi.PageID
	parentType notionapi.ParentType
}

// New creates a new Notion client
func New() (*Client, error) {
	apiKey := os.Getenv("NOTION_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("NOTION_API_KEY is not set")
	}

	parentID := os.Getenv("NOTION_PARENT_PAGE_ID")
	if parentID == "" {
		return nil, fmt.Errorf("NOTION_PARENT_PAGE_ID is not set")
	}

	notionClient := notionapi.NewClient(notionapi.Token(apiKey))
	return &Client{
		client:     newNotionClientAdapter(notionClient),
		parentID:   notionapi.PageID(parentID),
		parentType: "page_id",
	}, nil
}

// CreatePage creates a new page in Notion with the given title and markdown content
func (c *Client) CreatePage(ctx context.Context, title string, content string, tags []string) error {
	logger.Debug("Creating Notion page", map[string]interface{}{
		"title": title,
		"tags":  tags,
	})

	// Create page properties (only title)
	properties := notionapi.Properties{
		"title": notionapi.TitleProperty{
			Title: []notionapi.RichText{
				{
					Text: &notionapi.Text{
						Content: title,
					},
				},
			},
		},
	}

	// Create page
	pageParams := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:   c.parentType,
			PageID: c.parentID,
		},
		Properties: properties,
		Children:   c.convertMarkdownToBlocks(content),
	}

	page, err := c.client.Page().Create(ctx, pageParams)
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	// Create or update gallery views for each tag
	for _, tag := range tags {
		if err := c.addPageToTagGallery(ctx, page.ID, tag); err != nil {
			logger.Error("Failed to add page to tag gallery", err, map[string]interface{}{
				"tag":  tag,
				"page": title,
			})
		}
	}

	logger.Info("Successfully created Notion page", map[string]interface{}{
		"title": title,
		"tags":  tags,
	})

	return nil
}

// addPageToTagGallery creates a gallery view for the tag if it doesn't exist and adds the page to it
func (c *Client) addPageToTagGallery(ctx context.Context, pageID notionapi.ObjectID, tag string) error {
	// Search for existing gallery page with this tag name
	query := &notionapi.SearchRequest{
		Query: tag,
		Filter: notionapi.SearchFilter{
			Property: "object",
			Value:    "page",
		},
	}

	results, err := c.client.Search().Do(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to search for tag gallery: %w", err)
	}

	var galleryPage *notionapi.Page
	for _, result := range results.Results {
		if page, ok := result.(*notionapi.Page); ok {
			if title, ok := page.Properties["title"].(notionapi.TitleProperty); ok {
				if len(title.Title) > 0 && title.Title[0].Text != nil && title.Title[0].Text.Content == tag {
					galleryPage = page
					break
				}
			}
		}
	}

	// Create gallery page if it doesn't exist
	if galleryPage == nil {
		pageParams := &notionapi.PageCreateRequest{
			Parent: notionapi.Parent{
				Type:   c.parentType,
				PageID: c.parentID,
			},
			Properties: notionapi.Properties{
				"title": notionapi.TitleProperty{
					Title: []notionapi.RichText{
						{
							Text: &notionapi.Text{
								Content: tag,
							},
						},
					},
				},
			},
			Children: []notionapi.Block{
				&notionapi.ParagraphBlock{
					BasicBlock: notionapi.BasicBlock{
						Object: "block",
						Type:   notionapi.BlockTypeParagraph,
					},
					Paragraph: notionapi.Paragraph{
						RichText: []notionapi.RichText{
							{
								Text: &notionapi.Text{
									Content: fmt.Sprintf("Pages tagged with #%s", tag),
								},
							},
						},
					},
				},
			},
		}

		var err error
		galleryPage, err = c.client.Page().Create(ctx, pageParams)
		if err != nil {
			return fmt.Errorf("failed to create tag gallery page: %w", err)
		}
	}

	// Add link to the page in the gallery
	linkBlock := &notionapi.ParagraphBlock{
		BasicBlock: notionapi.BasicBlock{
			Object: "block",
			Type:   notionapi.BlockTypeParagraph,
		},
		Paragraph: notionapi.Paragraph{
			RichText: []notionapi.RichText{
				{
					Text: &notionapi.Text{
						Content: "📄 ",
					},
				},
				{
					Text: &notionapi.Text{
						Content: pageID.String(),
					},
					Href: fmt.Sprintf("https://notion.so/%s", pageID),
				},
			},
		},
	}

	// Check if the link already exists to maintain idempotency
	blocks, err := c.client.Block().GetChildren(ctx, notionapi.BlockID(galleryPage.ID), nil)
	if err != nil {
		return fmt.Errorf("failed to get gallery blocks: %w", err)
	}

	for _, block := range blocks.Results {
		if paragraph, ok := block.(*notionapi.ParagraphBlock); ok {
			if len(paragraph.Paragraph.RichText) > 1 && paragraph.Paragraph.RichText[1].Text.Content == pageID.String() {
				// Link already exists
				return nil
			}
		}
	}

	// Append the link block
	_, err = c.client.Block().AppendChildren(ctx, notionapi.BlockID(galleryPage.ID), &notionapi.AppendBlockChildrenRequest{
		Children: []notionapi.Block{linkBlock},
	})
	if err != nil {
		return fmt.Errorf("failed to append page link to gallery: %w", err)
	}

	return nil
}

// convertMarkdownToBlocks converts markdown content to Notion blocks
func (c *Client) convertMarkdownToBlocks(content string) []notionapi.Block {
	var blocks []notionapi.Block
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Handle headings
		if strings.HasPrefix(line, "# ") {
			blocks = append(blocks, c.createHeadingBlock(line[2:], 1))
			continue
		}
		if strings.HasPrefix(line, "## ") {
			blocks = append(blocks, c.createHeadingBlock(line[3:], 2))
			continue
		}

		// Handle code blocks
		if strings.HasPrefix(line, "```") {
			codeContent := []string{}
			i++
			for i < len(lines) && !strings.HasPrefix(lines[i], "```") {
				codeContent = append(codeContent, lines[i])
				i++
			}
			blocks = append(blocks, c.createCodeBlock(strings.Join(codeContent, "\n")))
			continue
		}

		// Handle bullet points
		if strings.HasPrefix(line, "- ") {
			blocks = append(blocks, c.createBulletedListBlock(line[2:]))
			continue
		}

		// Handle regular text
		blocks = append(blocks, c.createParagraphBlock(line))
	}

	return blocks
}

// createHeadingBlock creates a heading block with the specified level
func (c *Client) createHeadingBlock(text string, level int) notionapi.Block {
	richText := []notionapi.RichText{
		{
			Text: &notionapi.Text{
				Content: text,
			},
		},
	}

	switch level {
	case 1:
		return &notionapi.Heading1Block{
			BasicBlock: notionapi.BasicBlock{
				Object: "block",
				Type:   notionapi.BlockTypeHeading1,
			},
			Heading1: notionapi.Heading{
				RichText: richText,
			},
		}
	case 2:
		return &notionapi.Heading2Block{
			BasicBlock: notionapi.BasicBlock{
				Object: "block",
				Type:   notionapi.BlockTypeHeading2,
			},
			Heading2: notionapi.Heading{
				RichText: richText,
			},
		}
	default:
		return &notionapi.Heading3Block{
			BasicBlock: notionapi.BasicBlock{
				Object: "block",
				Type:   notionapi.BlockTypeHeading3,
			},
			Heading3: notionapi.Heading{
				RichText: richText,
			},
		}
	}
}

// createCodeBlock creates a code block
func (c *Client) createCodeBlock(content string) notionapi.Block {
	return &notionapi.CodeBlock{
		BasicBlock: notionapi.BasicBlock{
			Object: "block",
			Type:   notionapi.BlockTypeCode,
		},
		Code: notionapi.Code{
			RichText: []notionapi.RichText{
				{
					Text: &notionapi.Text{
						Content: content,
					},
				},
			},
			Language: "plain text",
		},
	}
}

// createBulletedListBlock creates a bulleted list item block
func (c *Client) createBulletedListBlock(text string) notionapi.Block {
	return &notionapi.BulletedListItemBlock{
		BasicBlock: notionapi.BasicBlock{
			Object: "block",
			Type:   notionapi.BlockTypeBulletedListItem,
		},
		BulletedListItem: notionapi.ListItem{
			RichText: []notionapi.RichText{
				{
					Text: &notionapi.Text{
						Content: text,
					},
				},
			},
		},
	}
}

// createParagraphBlock creates a paragraph block
func (c *Client) createParagraphBlock(text string) notionapi.Block {
	return &notionapi.ParagraphBlock{
		BasicBlock: notionapi.BasicBlock{
			Object: "block",
			Type:   notionapi.BlockTypeParagraph,
		},
		Paragraph: notionapi.Paragraph{
			RichText: []notionapi.RichText{
				{
					Text: &notionapi.Text{
						Content: text,
					},
				},
			},
		},
	}
}
