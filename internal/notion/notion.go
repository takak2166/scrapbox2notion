package notion

import "github.com/jomei/notionapi"

//go:generate mockgen -source=notion.go -destination=mock_notion/mock_notion.go -package=mock_notion
type NotionClient interface {
	Page() notionapi.PageService
	Search() notionapi.SearchService
	Block() notionapi.BlockService
	Database() notionapi.DatabaseService
	User() notionapi.UserService
}
