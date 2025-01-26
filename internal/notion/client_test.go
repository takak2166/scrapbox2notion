package notion

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jomei/notionapi"
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

// mockPageService is a mock implementation of the Notion PageService interface
type mockPageService struct {
	pages map[notionapi.PageID]*notionapi.Page
}

func newMockPageService() *mockPageService {
	return &mockPageService{
		pages: make(map[notionapi.PageID]*notionapi.Page),
	}
}

func (m *mockPageService) Create(ctx context.Context, params *notionapi.PageCreateRequest) (*notionapi.Page, error) {
	page := &notionapi.Page{
		Object:     "page",
		ID:         notionapi.ObjectID("test_page_id"),
		Properties: params.Properties,
	}
	m.pages[notionapi.PageID("test_page_id")] = page
	return page, nil
}

func (m *mockPageService) Get(ctx context.Context, id notionapi.PageID) (*notionapi.Page, error) {
	return m.pages[id], nil
}

func (m *mockPageService) Update(ctx context.Context, id notionapi.PageID, params *notionapi.PageUpdateRequest) (*notionapi.Page, error) {
	return nil, nil
}

// mockDatabaseService is a mock implementation of the Notion DatabaseService interface
type mockDatabaseService struct {
	databases map[notionapi.DatabaseID]*notionapi.Database
	pages     map[notionapi.DatabaseID][]*notionapi.Page
}

func newMockDatabaseService() *mockDatabaseService {
	return &mockDatabaseService{
		databases: make(map[notionapi.DatabaseID]*notionapi.Database),
		pages:     make(map[notionapi.DatabaseID][]*notionapi.Page),
	}
}

func (m *mockDatabaseService) Get(ctx context.Context, id notionapi.DatabaseID) (*notionapi.Database, error) {
	return m.databases[id], nil
}

func (m *mockDatabaseService) Create(ctx context.Context, params *notionapi.DatabaseCreateRequest) (*notionapi.Database, error) {
	db := &notionapi.Database{
		ID: notionapi.ObjectID("test_db_id"),
	}
	m.databases[notionapi.DatabaseID("test_db_id")] = db
	return db, nil
}

func (m *mockDatabaseService) Query(ctx context.Context, id notionapi.DatabaseID, params *notionapi.DatabaseQueryRequest) (*notionapi.DatabaseQueryResponse, error) {
	pages := m.pages[id]
	return &notionapi.DatabaseQueryResponse{
		Results: make([]notionapi.Page, len(pages)),
	}, nil
}

func (m *mockDatabaseService) Update(ctx context.Context, id notionapi.DatabaseID, params *notionapi.DatabaseUpdateRequest) (*notionapi.Database, error) {
	db := m.databases[id]
	if db == nil {
		return nil, fmt.Errorf("database not found")
	}
	return db, nil
}

func TestCreatePage(t *testing.T) {
	tests := []struct {
		name              string
		envVars           map[string]string
		title             string
		content           string
		tags              []string
		expectNewDatabase bool
		expectError       bool
	}{
		{
			name: "Create page with existing tags database",
			envVars: map[string]string{
				"NOTION_API_KEY":          "test_key",
				"NOTION_PARENT_PAGE_ID":   "test_page_id",
				"NOTION_TAGS_DATABASE_ID": "existing_db_id",
			},
			title: "Test Page",
			content: `# Test Page

This is a test page.`,
			tags:              []string{"test"},
			expectNewDatabase: false,
			expectError:       false,
		},
		{
			name: "Create page with new tags database",
			envVars: map[string]string{
				"NOTION_API_KEY":        "test_key",
				"NOTION_PARENT_PAGE_ID": "test_page_id",
			},
			title: "Test Page",
			content: `# Test Page

This is a test page.`,
			tags:              []string{"test"},
			expectNewDatabase: true,
			expectError:       false,
		},
		{
			name: "Create page without tags",
			envVars: map[string]string{
				"NOTION_API_KEY":        "test_key",
				"NOTION_PARENT_PAGE_ID": "test_page_id",
			},
			title: "Test Page",
			content: `# Test Page

This is a test page.`,
			tags:              nil,
			expectNewDatabase: true,
			expectError:       false,
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
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Create mock services
			mockPageService := newMockPageService()
			mockDatabaseService := newMockDatabaseService()
			client.client = &notionapi.Client{
				Page:     mockPageService,
				Database: mockDatabaseService,
			}

			ctx := context.Background()
			err = client.CreatePage(ctx, tt.title, tt.content, tt.tags)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify database creation
			if tt.expectNewDatabase {
				if len(mockDatabaseService.databases) == 0 {
					t.Error("Expected new database to be created")
				}
			}

			// Verify the page was created
			page := mockPageService.pages[notionapi.PageID("test_page_id")]
			if page == nil {
				t.Error("Expected page to be created")
				return
			}

			// Verify the title
			titleProp, ok := page.Properties["title"].(notionapi.TitleProperty)
			if !ok {
				t.Error("Expected title property")
				return
			}
			if len(titleProp.Title) == 0 || titleProp.Title[0].Text.Content != tt.title {
				t.Errorf("Expected title %s, got %s", tt.title, titleProp.Title[0].Text.Content)
			}

			// Verify tags if present
			if len(tt.tags) > 0 {
				tagsProp, ok := page.Properties["Tags"].(notionapi.RelationProperty)
				if !ok {
					t.Error("Expected Tags property")
					return
				}
				if len(tagsProp.Relation) != len(tt.tags) {
					t.Errorf("Expected %d tags, got %d", len(tt.tags), len(tagsProp.Relation))
				}
			}
		})
	}
}
