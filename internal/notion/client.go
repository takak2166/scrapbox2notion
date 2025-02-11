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

	// If no tags, use default parent
	parent := notionapi.Parent{
		Type:   c.parentType,
		PageID: c.parentID,
	}

	// If tags exist, create/get tag database and use as parent
	if len(tags) > 0 {
		// Use first tag as parent
		tagDB, err := c.createDatabase(ctx, tags[0], map[string]notionapi.PropertyConfig{
			"Name": notionapi.TitlePropertyConfig{
				Type:  "title",
				Title: struct{}{},
			},
			"Created": notionapi.DatePropertyConfig{
				Type: "date",
				Date: struct{}{},
			},
			"Content": notionapi.RichTextPropertyConfig{
				Type:     "rich_text",
				RichText: struct{}{},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create tag database: %w", err)
		}

		parent = notionapi.Parent{
			Type:       "database_id",
			DatabaseID: notionapi.DatabaseID(tagDB.ID),
		}
	}

	// Create page with determined parent
	pageParams := &notionapi.PageCreateRequest{
		Parent: parent,
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
			"Content": notionapi.RichTextProperty{
				RichText: []notionapi.RichText{
					{
						Text: &notionapi.Text{
							Content: content,
						},
					},
				},
			},
		},
		Children: c.convertMarkdownToBlocks(content),
	}

	// Retry page creation up to 3 times with 1 second delay
	var page *notionapi.Page
	var err error

	for i := 0; i < 3; i++ {
		page, err = c.client.Page().Create(ctx, pageParams)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("failed to create page after 3 attempts: %w", err)
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

// addPageToTagGallery creates a database for the tag if it doesn't exist and adds the page to it
func (c *Client) addPageToTagGallery(ctx context.Context, pageID notionapi.ObjectID, tag string) error {
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

	var tagDatabase *notionapi.Database
	for _, result := range results.Results {
		if db, ok := result.(*notionapi.Database); ok {
			if len(db.Title) > 0 && db.Title[0].Text != nil && db.Title[0].Text.Content == tag {
				tagDatabase = db
				break
			}
		}
	}

	// Create database if it doesn't exist
	if tagDatabase == nil {
		// Define database properties
		properties := notionapi.PropertyConfigs{
			"Name": notionapi.TitlePropertyConfig{
				Type:  "title",
				Title: struct{}{},
			},
			"Created": notionapi.DatePropertyConfig{
				Type: "date",
				Date: struct{}{},
			},
		}

		dbParams := &notionapi.DatabaseCreateRequest{
			Parent: notionapi.Parent{
				Type:   c.parentType,
				PageID: c.parentID,
			},
			Title: []notionapi.RichText{
				{
					Text: &notionapi.Text{
						Content: tag,
					},
				},
			},
			Properties: properties,
			IsInline:   true,
		}

		tagDatabase, err = c.client.Database().Create(ctx, dbParams)
		if err != nil {
			return fmt.Errorf("failed to create tag database: %w", err)
		}
	}

	// Get page details to add to database
	page, err := c.client.Page().Get(ctx, notionapi.PageID(pageID))
	if err != nil {
		return fmt.Errorf("failed to get page details: %w", err)
	}

	// Create database entry
	pageTitle := ""
	if titleProp, ok := page.Properties["title"].(notionapi.TitleProperty); ok && len(titleProp.Title) > 0 {
		pageTitle = titleProp.Title[0].Text.Content
	}

	createdTime := notionapi.Date(page.CreatedTime)
	pageParams := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:       "database_id",
			DatabaseID: notionapi.DatabaseID(tagDatabase.ID),
		},
		Properties: notionapi.Properties{
			"Name": notionapi.TitleProperty{
				Title: []notionapi.RichText{
					{
						Text: &notionapi.Text{
							Content: pageTitle,
						},
					},
				},
			},
			"Created": notionapi.DateProperty{
				Date: &notionapi.DateObject{
					Start: &createdTime,
				},
			},
		},
	}

	_, err = c.client.Page().Create(ctx, pageParams)
	if err != nil {
		return fmt.Errorf("failed to create database entry: %w", err)
	}

	return nil
}

// createDatabase creates a new database with the given name and properties if it doesn't already exist
func (c *Client) createDatabase(ctx context.Context, name string, properties notionapi.PropertyConfigs) (*notionapi.Database, error) {
	// Search for existing database with this name
	query := &notionapi.SearchRequest{
		Query: name,
		Filter: notionapi.SearchFilter{
			Property: "object",
			Value:    "database",
		},
	}

	results, err := c.client.Search().Do(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search for existing database: %w", err)
	}

	// Check if a database with the same name already exists
	for _, result := range results.Results {
		if db, ok := result.(*notionapi.Database); ok {
			if len(db.Title) > 0 && db.Title[0].Text != nil && db.Title[0].Text.Content == name {
				return db, nil
			}
		}
	}

	// Create new database if it doesn't exist
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
