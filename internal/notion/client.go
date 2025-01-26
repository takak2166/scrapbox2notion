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
	client     *notionapi.Client
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

	return &Client{
		client:     notionapi.NewClient(notionapi.Token(apiKey)),
		parentID:   notionapi.PageID(parentID),
		parentType: "page_id",
	}, nil
}

// CreateTagsDatabase creates a new database for tags if it doesn't exist
func (c *Client) CreateTagsDatabase(ctx context.Context) (notionapi.DatabaseID, error) {
	// Check if database already exists
	dbID := os.Getenv("NOTION_TAGS_DATABASE_ID")
	if dbID != "" {
		logger.Debug("Using existing tags database", map[string]interface{}{
			"database_id": dbID,
		})
		return notionapi.DatabaseID(dbID), nil
	}

	logger.Info("Creating new tags database", nil)

	// Create new database
	dbReq := &notionapi.DatabaseCreateRequest{
		Parent: notionapi.Parent{
			Type:   c.parentType,
			PageID: c.parentID,
		},
		Title: []notionapi.RichText{
			{
				Text: &notionapi.Text{
					Content: "Tags",
				},
			},
		},
		Properties: notionapi.PropertyConfigs{
			"Name": notionapi.TitlePropertyConfig{
				Type:  "title",
				Title: struct{}{},
			},
		},
		IsInline: false,
	}

	db, err := c.client.Database.Create(ctx, dbReq)
	if err != nil {
		return "", fmt.Errorf("failed to create tags database: %w", err)
	}

	dbID = string(db.ID)
	logger.Info("Successfully created tags database", map[string]interface{}{
		"database_id": dbID,
	})

	return notionapi.DatabaseID(db.ID), nil
}

// CreateOrGetTag creates a new tag in the database or gets existing one
func (c *Client) CreateOrGetTag(ctx context.Context, dbID notionapi.DatabaseID, tagName string) (notionapi.PageID, error) {
	// Search for existing tag
	query := &notionapi.DatabaseQueryRequest{
		Filter: &notionapi.PropertyFilter{
			Property: "Name",
			RichText: &notionapi.TextFilterCondition{
				Equals: tagName,
			},
		},
	}

	result, err := c.client.Database.Query(ctx, dbID, query)
	if err != nil {
		return "", fmt.Errorf("failed to query tags: %w", err)
	}

	if len(result.Results) > 0 {
		logger.Debug("Found existing tag", map[string]interface{}{
			"tag": tagName,
			"id":  result.Results[0].ID,
		})
		return notionapi.PageID(result.Results[0].ID), nil
	}

	// Create new tag
	pageReq := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:       "database_id",
			DatabaseID: dbID,
		},
		Properties: notionapi.Properties{
			"Name": notionapi.TitleProperty{
				Title: []notionapi.RichText{
					{
						Text: &notionapi.Text{
							Content: tagName,
						},
					},
				},
			},
		},
	}

	page, err := c.client.Page.Create(ctx, pageReq)
	if err != nil {
		return "", fmt.Errorf("failed to create tag: %w", err)
	}

	logger.Debug("Created new tag", map[string]interface{}{
		"tag": tagName,
		"id":  page.ID,
	})

	return notionapi.PageID(page.ID), nil
}

// CreatePage creates a new page in Notion with the given title and markdown content
func (c *Client) CreatePage(ctx context.Context, title string, content string, tags []string) error {
	logger.Debug("Creating Notion page", map[string]interface{}{
		"title": title,
		"tags":  tags,
	})

	// Create or get tags database
	dbID, err := c.CreateTagsDatabase(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup tags database: %w", err)
	}

	// Create or get tag pages
	var tagIDs []notionapi.PageID
	for _, tag := range tags {
		tagID, err := c.CreateOrGetTag(ctx, dbID, tag)
		if err != nil {
			logger.Error("Failed to create/get tag", err, map[string]interface{}{
				"tag": tag,
			})
			continue
		}
		tagIDs = append(tagIDs, tagID)
	}

	// Create page properties
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

	// Add tags relation if we have tags
	if len(tagIDs) > 0 {
		properties["Tags"] = notionapi.RelationProperty{
			Relation: make([]notionapi.Relation, len(tagIDs)),
		}
		for i, tagID := range tagIDs {
			properties["Tags"].(notionapi.RelationProperty).Relation[i] = notionapi.Relation{
				ID: tagID,
			}
		}
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

	_, err = c.client.Page.Create(ctx, pageParams)
	if err != nil {
		return fmt.Errorf("failed to create page: %w", err)
	}

	logger.Info("Successfully created Notion page", map[string]interface{}{
		"title": title,
		"tags":  tags,
	})

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
