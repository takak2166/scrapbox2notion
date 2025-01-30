package notion

import (
	"context"

	"github.com/jomei/notionapi"
)

//go:generate mockgen -source=notionapi_interfaces.go -destination=mock_notion/mock_notionapi.go -package=mock_notion
type (
	PageService interface {
		Create(context.Context, *notionapi.PageCreateRequest) (*notionapi.Page, error)
		Update(context.Context, notionapi.PageID, *notionapi.PageUpdateRequest) (*notionapi.Page, error)
		Get(context.Context, notionapi.PageID) (*notionapi.Page, error)
	}

	SearchService interface {
		Do(context.Context, *notionapi.SearchRequest) (*notionapi.SearchResponse, error)
	}

	BlockService interface {
		AppendChildren(context.Context, notionapi.BlockID, *notionapi.AppendBlockChildrenRequest) (*notionapi.AppendBlockChildrenResponse, error)
		Get(context.Context, notionapi.BlockID) (notionapi.Block, error)
		GetChildren(context.Context, notionapi.BlockID, *notionapi.Pagination) (*notionapi.GetChildrenResponse, error)
		Update(ctx context.Context, id notionapi.BlockID, request *notionapi.BlockUpdateRequest) (notionapi.Block, error)
		Delete(context.Context, notionapi.BlockID) (notionapi.Block, error)
	}
)
