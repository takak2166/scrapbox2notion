package notion

import (
	"context"
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

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock_notion.NewMockNotionClient(ctrl)
	mockPage := mock_notion.NewMockPageService(ctrl)
	mockSearch := mock_notion.NewMockSearchService(ctrl)
	mockBlock := mock_notion.NewMockBlockService(ctrl)

	client.client = mockClient

	tests := []struct {
		name        string
		title       string
		content     string
		tags        []string
		expectError bool
		setupMocks  func()
	}{
		{
			name:  "Basic page",
			title: "Test Page",
			content: `# Test Page

This is a test page.`,
			tags:        []string{"test"},
			expectError: false,
			setupMocks: func() {
				mockClient.EXPECT().Page().Return(mockPage).AnyTimes()
				mockClient.EXPECT().Search().Return(mockSearch).AnyTimes()
				mockClient.EXPECT().Block().Return(mockBlock).AnyTimes()

				mockPage.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&notionapi.Page{
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

				mockSearch.EXPECT().Do(gomock.Any(), gomock.Any()).Return(&notionapi.SearchResponse{
					Results: []notionapi.Object{},
				}, nil)

				// Mock database creation
				mockDatabase := mock_notion.NewMockDatabaseService(ctrl)
				mockClient.EXPECT().Database().Return(mockDatabase).AnyTimes()

				// Mock database creation response
				mockDatabase.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&notionapi.Database{
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

				// Mock page update instead of create
				mockPage.EXPECT().Update(gomock.Any(), notionapi.PageID("test_page_id"), gomock.Any()).Return(&notionapi.Page{
					Object: "page",
					ID:     "test_page_id",
					Parent: notionapi.Parent{
						Type:       "database_id",
						DatabaseID: "test_db_id",
					},
				}, nil)

				// Mock database search
				mockSearch.EXPECT().Do(gomock.Any(), gomock.Any()).Return(&notionapi.SearchResponse{
					Results: []notionapi.Object{
						&notionapi.Database{
							Object: "database",
							ID:     "test_db_id",
							Title: []notionapi.RichText{
								{
									Text: &notionapi.Text{
										Content: "test",
									},
								},
							},
						},
					},
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := client.CreatePage(context.Background(), tt.title, tt.content, tt.tags)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
