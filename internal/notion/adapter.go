package notion

import (
	"github.com/jomei/notionapi"
)

type notionClientAdapter struct {
	client *notionapi.Client
}

func newNotionClientAdapter(client *notionapi.Client) NotionClient {
	return &notionClientAdapter{client: client}
}

func (a *notionClientAdapter) Page() notionapi.PageService {
	return a.client.Page
}

func (a *notionClientAdapter) Search() notionapi.SearchService {
	return a.client.Search
}

func (a *notionClientAdapter) Block() notionapi.BlockService {
	return a.client.Block
}

func (a *notionClientAdapter) Database() notionapi.DatabaseService {
	return a.client.Database
}

func (a *notionClientAdapter) User() notionapi.UserService {
	return a.client.User
}
