# Scrapbox to Notion Migration Tool

[English](#english) | [日本語](#japanese)

<a id="english"></a>
## English

A Go tool to migrate Scrapbox pages to Notion. This tool converts Scrapbox JSON exports to markdown format and uploads them to Notion using the Notion API.

### Features

- Converts Scrapbox JSON exports to markdown format
- Creates pages in Notion with proper formatting
- Saves markdown files locally for reference
- Detailed logging of the migration process

### Prerequisites

- Go 1.16 or later
- A Notion API key
- A Notion parent page ID where the migrated pages will be created

### Installation

```bash
go install github.com/takak2166/scrapbox2notion@latest
```

Or clone and build manually:

```bash
git clone https://github.com/takak2166/scrapbox2notion.git
cd scrapbox2notion
go build -o bin/scrapbox2notion cmd/main.go
```

### Configuration

Create a `.env` file in the project root with the following variables:

```env
# Logging
LOG_LEVEL=debug # debug, info, warn, error

# Notion API
NOTION_API_KEY=your_notion_api_key
NOTION_PARENT_PAGE_ID=your_notion_parent_page_id

# Application Settings
OUTPUT_DIR=output # Directory for markdown files
```

### Usage

1. Export your Scrapbox pages as JSON
2. Run the migration tool:

```bash
scrapbox2notion -input path/to/scrapbox_export.json [-output path/to/markdown/output]
```

Options:
- `-input`: Path to the Scrapbox JSON export file (required)
- `-output`: Directory to save markdown files (optional, defaults to OUTPUT_DIR in .env or output)

---

<a id="japanese"></a>
## 日本語

ScrapboxのページをNotionに移行するためのGoツールです。ScrapboxのJSONエクスポートをMarkdown形式に変換し、Notion APIを使用してNotionにアップロードします。

### 機能

- ScrapboxのJSONエクスポートをMarkdown形式に変換し、Notionにページを作成
- 参照用にMarkdownファイルをローカルに保存

### 必要条件

- Go 1.16以降
- Notion APIキー
- 移行先となるNotionの親ページID

### インストール

```bash
go install github.com/takak2166/scrapbox2notion@latest
```

または、クローンしてビルド：

```bash
git clone https://github.com/takak2166/scrapbox2notion.git
cd scrapbox2notion
go build -o bin/scrapbox2notion cmd/main.go
```

### 設定

プロジェクトのルートディレクトリに以下の変数を含む`.env`ファイルを作成してください：

```env
# ログレベル
LOG_LEVEL=debug # debug, info, warn, error

# Notion API
NOTION_API_KEY=your_notion_api_key
NOTION_PARENT_PAGE_ID=your_notion_parent_page_id

# アプリケーション設定
OUTPUT_DIR=output # Markdownファイルの出力ディレクトリ
```

### 使用方法

1. ScrapboxのページをJSONとしてエクスポート
2. 移行ツールを実行：

```bash
scrapbox2notion -input path/to/scrapbox_export.json [-output path/to/markdown/output]
```

オプション：
- `-input`: ScrapboxのJSONエクスポートファイルのパス（必須）
- `-output`: Markdownファイルを保存するディレクトリ（オプション、デフォルトは.envのOUTPUT_DIRまたはoutput）

## License

MIT License