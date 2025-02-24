package notion

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/jomei/notionapi"
	"github.com/takak2166/scrapbox2notion/internal/notion/mock_notion"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		expectError bool
	}{
		{
			name: "Valid configuration",
			envVars: map[string]string{
				"NOTION_API_KEY":        "test_key",
				"NOTION_PARENT_PAGE_ID": "test_page_id",
			},
			expectError: false,
		},
		{
			name: "Missing API key",
			envVars: map[string]string{
				"NOTION_PARENT_PAGE_ID": "test_page_id",
			},
			expectError: true,
		},
		{
			name: "Missing parent page ID",
			envVars: map[string]string{
				"NOTION_API_KEY": "test_key",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables
			os.Clearenv()

			// Set test environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			client, err := New()
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Error("Expected client, got nil")
				}
			}
		})
	}
}

func TestCreatePage(t *testing.T) {
	// Set up test environment
	os.Setenv("NOTION_API_KEY", "test_key")
	os.Setenv("NOTION_PARENT_PAGE_ID", "test_page_id")

	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	tests := map[string]struct {
		title      string
		content    string
		tags       []string
		setupMocks func(mockClient *mock_notion.MockNotionClient, mockPage *mock_notion.MockPageService, mockSearch *mock_notion.MockSearchService, mockDatabase *mock_notion.MockDatabaseService)
	}{
		"Success - With Tags": {
			title: "Test Page",
			content: `# Test Page

This is a test page.`,
			tags: []string{"Test"},
			setupMocks: func(mockClient *mock_notion.MockNotionClient, mockPage *mock_notion.MockPageService, mockSearch *mock_notion.MockSearchService, mockDatabase *mock_notion.MockDatabaseService) {
				// Set up service returns
				mockClient.EXPECT().Search().Return(mockSearch).AnyTimes()
				mockClient.EXPECT().Database().Return(mockDatabase).AnyTimes()
				mockClient.EXPECT().Page().Return(mockPage).AnyTimes()

				// Initial search for database
				mockSearch.EXPECT().Do(ctx, gomock.Any()).Return(&notionapi.SearchResponse{}, nil)

				// Create database
				mockDatabase.EXPECT().Create(ctx, gomock.Any()).Return(&notionapi.Database{
					Object: "database",
					ID:     "test_db_id",
					Title: []notionapi.RichText{
						{
							Text: &notionapi.Text{
								Content: "test",
							},
						},
					},
				}, nil)

				// Validation searches for database creation
				mockSearch.EXPECT().Do(ctx, gomock.Any()).Return(&notionapi.SearchResponse{
					Results: []notionapi.Object{
						&notionapi.Database{
							Object: "database",
							ID:     "test_db_id",
							Title: []notionapi.RichText{
								{
									Text: &notionapi.Text{
										Content: "Test",
									},
								},
							},
						},
					},
				}, nil)

				// Query database for existing pages
				mockDatabase.EXPECT().Query(ctx, notionapi.DatabaseID("test_db_id"), gomock.Any()).Return(&notionapi.DatabaseQueryResponse{
					Results: []notionapi.Page{},
				}, nil)

				// Create page
				mockPage.EXPECT().Create(ctx, gomock.Any()).Return(&notionapi.Page{
					Object: "page",
					ID:     "test_page_id",
					Properties: notionapi.Properties{
						"title": notionapi.TitleProperty{
							Title: []notionapi.RichText{
								{
									Text: &notionapi.Text{
										Content: "Test Page",
									},
								},
							},
						},
					},
				}, nil)

				// Get page to confirm creation
				mockPage.EXPECT().Get(ctx, notionapi.PageID("test_page_id")).Return(&notionapi.Page{
					Object: "page",
					ID:     "test_page_id",
					Properties: notionapi.Properties{
						"title": notionapi.TitleProperty{
							Title: []notionapi.RichText{
								{
									Text: &notionapi.Text{
										Content: "Test Page",
									},
								},
							},
						},
					},
				}, nil)
			},
		},

		"Success - Without Tags": {
			title: "Test Page 2",
			content: `# Test Page 2

This is another test page.`,
			tags: []string{},
			setupMocks: func(mockClient *mock_notion.MockNotionClient, mockPage *mock_notion.MockPageService, mockSearch *mock_notion.MockSearchService, mockDatabase *mock_notion.MockDatabaseService) {
				// Set up service returns
				mockClient.EXPECT().Search().Return(mockSearch).AnyTimes()
				mockClient.EXPECT().Page().Return(mockPage).AnyTimes()

				// Search for existing page
				mockSearch.EXPECT().Do(ctx, gomock.Any()).Return(&notionapi.SearchResponse{
					Results: []notionapi.Object{},
				}, nil)

				// Create page
				mockPage.EXPECT().Create(ctx, gomock.Any()).Return(&notionapi.Page{
					Object: "page",
					ID:     "test_page_id_2",
					Properties: notionapi.Properties{
						"Name": notionapi.TitleProperty{
							Title: []notionapi.RichText{
								{
									Text: &notionapi.Text{
										Content: "Test Page 2",
									},
								},
							},
						},
					},
				}, nil)
			},
		},

		"Failure - Empty Title": {
			title: "",
			content: `# Empty Page

This page has no title.`,
			tags: []string{"error"},
			setupMocks: func(mockClient *mock_notion.MockNotionClient, mockPage *mock_notion.MockPageService, mockSearch *mock_notion.MockSearchService, mockDatabase *mock_notion.MockDatabaseService) {
				// Set up service returns
				mockClient.EXPECT().Search().Return(mockSearch).AnyTimes()
				mockClient.EXPECT().Database().Return(mockDatabase).AnyTimes()

				// Initial search for database
				mockSearch.EXPECT().Do(ctx, gomock.Any()).Return(&notionapi.SearchResponse{}, nil)

				// Database creation fails
				mockDatabase.EXPECT().Create(ctx, gomock.Any()).Return(nil, errors.New("title cannot be empty"))
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Reinitialize mocks for each test case
			mockClient := mock_notion.NewMockNotionClient(ctrl)
			mockPage := mock_notion.NewMockPageService(ctrl)
			mockSearch := mock_notion.NewMockSearchService(ctrl)
			mockDatabase := mock_notion.NewMockDatabaseService(ctrl)

			client.client = mockClient
			tt.setupMocks(mockClient, mockPage, mockSearch, mockDatabase)

			err := client.CreatePage(context.Background(), tt.title, tt.content, tt.tags)
			if name == "Failure - Empty Title" {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
