package notion

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

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

	// Create database for each tag and add page to it
	for _, tag := range tags {
		// Search for existing database with this tag name
		query := &notionapi.SearchRequest{
			Query: tag,
			Filter: notionapi.SearchFilter{
				Property: "object",
				Value:    "database",
			},
		}

		results, err := c.client.Search().Do(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to search for tag database: %w", err)
		}

		tagDB := validateTagsDatabase(tag, results)

		// Create database if it doesn't exist
		if tagDB == nil {
			tagDB, err = c.createDatabase(ctx, tag, map[string]notionapi.PropertyConfig{
				"Name": notionapi.TitlePropertyConfig{
					Type:  "title",
					Title: struct{}{},
				},
				"Tag": notionapi.SelectPropertyConfig{
					Type: "select",
					Select: notionapi.Select{
						Options: []notionapi.Option{},
					},
				},
				"Created": notionapi.DatePropertyConfig{
					Type: "date",
					Date: struct{}{},
				},
			})
			if err != nil {
				return fmt.Errorf("failed to create tag database: %w", err)
			}
			logger.Info("Successfully created tags database", map[string]interface{}{
				"tags": tags,
			})

			// Confirm database creation
			var exists bool
			for i := 0; i < 10; i++ {
				results, err := c.client.Search().Do(ctx, query)
				if err == nil && len(results.Results) > 0 {
					if validateTagsDatabase(tag, results) != nil {
						exists = true
						break
					}
				}
				time.Sleep(1 * time.Second)
			}
			if !exists {
				return fmt.Errorf("failed to create tag database: %w", err)
			}
		}

		createdAt := notionapi.Date(time.Now())

		// Check if page with same title already exists in the database
		pageQuery := &notionapi.DatabaseQueryRequest{
			Filter: notionapi.PropertyFilter{
				Property: "Name",
				RichText: &notionapi.TextFilterCondition{
					Equals: title,
				},
			},
		}

		existingPages, err := c.client.Database().Query(ctx, notionapi.DatabaseID(tagDB.ID), pageQuery)
		if err != nil {
			return fmt.Errorf("failed to query database for existing pages: %w", err)
		}

		// Only create page if it doesn't already exist
		if len(existingPages.Results) == 0 {
			pageParams := &notionapi.PageCreateRequest{
				Parent: notionapi.Parent{
					Type:       "database_id",
					DatabaseID: notionapi.DatabaseID(tagDB.ID),
				},
				Properties: notionapi.Properties{
					"Name": notionapi.TitleProperty{
						Title: []notionapi.RichText{
							{
								Text: &notionapi.Text{
									Content: title,
								},
							},
						},
					},
					"Tag": notionapi.SelectProperty{
						Type: "select",
						Select: notionapi.Option{
							Name: tag,
						},
					},
					"Created": notionapi.DateProperty{
						Date: &notionapi.DateObject{
							Start: &createdAt,
						},
					},
				},
				Children: c.convertMarkdownToBlocks(content),
			}

			var exists bool
			page, err := c.client.Page().Create(ctx, pageParams)
			if err != nil {
				return fmt.Errorf("failed to create page in tag database: %w", err)
			}
			for i := 0; i < 5; i++ {
				resp, err := c.client.Page().Get(ctx, notionapi.PageID(page.ID))
				if err == nil && resp.ID == page.ID {
					exists = true
					break
				}
				time.Sleep(1 * time.Second)
			}
			if !exists {
				return fmt.Errorf("failed to create page in tag database: %w", err)
			}
			logger.Info("Successfully created Notion page", map[string]interface{}{
				"title": title,
				"tags":  tags,
			})
		} else {
			logger.Info("Notion page has already existed, skip creating", map[string]interface{}{
				"title": title,
				"tags":  tags,
			})
		}
	}

	// If no tags, create page in default parent
	if len(tags) == 0 {
		req := &notionapi.SearchRequest{
			Query: title,
			Filter: notionapi.SearchFilter{
				Property: "object",
				Value:    "page",
			},
		}
		resp, err := c.client.Search().Do(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to search pages, %w", err)
		}
		if len(resp.Results) == 0 {
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
									Content: title,
								},
							},
						},
					},
				},
				Children: c.convertMarkdownToBlocks(content),
			}

			_, err := c.client.Page().Create(ctx, pageParams)
			if err != nil {
				return fmt.Errorf("failed to create page: %w", err)
			}
			logger.Info("Successfully created Notion page", map[string]interface{}{
				"title": title,
				"tags":  tags,
			})
		}
	}

	return nil
}

// createDatabase creates a new database with the given name and properties
func (c *Client) createDatabase(ctx context.Context, name string, properties notionapi.PropertyConfigs) (*notionapi.Database, error) {
	// Create new database
	dbParams := &notionapi.DatabaseCreateRequest{
		Parent: notionapi.Parent{
			Type:   c.parentType,
			PageID: c.parentID,
		},
		Title: []notionapi.RichText{
			{
				Text: &notionapi.Text{
					Content: name,
				},
			},
		},
		Properties: properties,
		IsInline:   true,
	}

	db, err := c.client.Database().Create(ctx, dbParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	return db, nil
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

func validateTagsDatabase(tag string, results *notionapi.SearchResponse) *notionapi.Database {
	for _, result := range results.Results {
		if db, ok := result.(*notionapi.Database); ok {
			if len(db.Title) > 0 && db.Title[0].Text != nil && db.Title[0].Text.Content == tag {
				return db
			}
		}
	}
	return nil
}
