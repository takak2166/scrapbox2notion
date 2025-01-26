package models

// ScrapboxExport represents the root structure of the Scrapbox export JSON
type ScrapboxExport struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Exported    int64  `json:"exported"`
	Pages       []Page `json:"pages"`
}

// Page represents a Scrapbox page
type Page struct {
	Title   string   `json:"title"`
	Created int64    `json:"created"`
	Updated int64    `json:"updated"`
	ID      string   `json:"id"`
	Views   int      `json:"views"`
	Lines   []Line   `json:"lines"`
	LinksLc []string `json:"linksLc,omitempty"` // Changed to []string to handle direct string values
	Tags    []string // Extracted from lines starting with #
}

// Line represents a line of text in a Scrapbox page
type Line struct {
	Text    string `json:"text"`
	Created int64  `json:"created"`
	Updated int64  `json:"updated"`
	UserID  string `json:"userId"`
}

// NotionIDs holds Notion page and database IDs
type NotionIDs struct {
	TagsDatabaseID string
	PageID         string
}
