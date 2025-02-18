package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/takak2166/scrapbox2notion/internal/logger"
	"github.com/takak2166/scrapbox2notion/internal/notion"
	"github.com/takak2166/scrapbox2notion/internal/parser"
)

func main() {
	// Parse command line flags
	inputFile := flag.String("input", "", "Path to Scrapbox JSON export file")
	outputDir := flag.String("output", "", "Directory to save markdown files (optional)")
	flag.Parse()

	if *inputFile == "" {
		fmt.Println("Error: input file is required")
		flag.Usage()
		os.Exit(1)
	}

	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	if err := logger.Init(logLevel); err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}

	// Get output directory from environment if not specified
	if *outputDir == "" {
		*outputDir = os.Getenv("OUTPUT_DIR")
		if *outputDir == "" {
			*outputDir = "output"
		}
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		logger.Error("Failed to create output directory", err, nil)
		os.Exit(1)
	}

	// Initialize parser
	p := parser.New()

	// Parse Scrapbox JSON file
	if err := p.ParseFile(*inputFile); err != nil {
		logger.Error("Failed to parse input file", err, nil)
		os.Exit(1)
	}

	// Initialize Notion client
	notionClient, err := notion.New()
	if err != nil {
		logger.Error("Failed to initialize Notion client", err, nil)
		os.Exit(1)
	}

	// Process each page
	pages := p.GetPages()
	logger.Info(fmt.Sprintf("Found %d pages to process", len(pages)), nil)

	ctx := context.Background()
	successCount := 0

	for _, page := range pages {
		// Convert to markdown
		markdown := p.ConvertToMarkdown(&page)

		// Save markdown file
		mdFilePath := filepath.Join(*outputDir, page.Title+".md")
		if err := os.WriteFile(mdFilePath, []byte(markdown), 0644); err != nil {
			logger.Error("Failed to save markdown file", err, map[string]interface{}{
				"page":     page.Title,
				"filepath": mdFilePath,
			})
			continue
		}

		// Upload to Notion with tags
		if err := notionClient.CreatePage(ctx, page.Title, markdown, page.Tags); err != nil {
			logger.Error("Failed to create Notion page", err, map[string]interface{}{
				"page": page.Title,
			})
			continue
		}

		successCount++
	}

	logger.Info("Migration completed", map[string]interface{}{
		"total_pages":     len(pages),
		"success_count":   successCount,
		"failure_count":   len(pages) - successCount,
		"markdown_output": *outputDir,
	})
}
