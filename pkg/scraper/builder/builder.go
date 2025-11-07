package builder

import (
	"fmt"
	"time"

	"web-text-pipe-go/pkg/scraper/runner"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-web-exact/v2/pkg/extract"
	"github.com/shouni/go-web-exact/v2/pkg/feed"
	"github.com/shouni/go-web-exact/v2/pkg/scraper"
)

// BuildScraperRunner は、必要な設定値に基づいて、runner.Runnerの依存関係をすべて構築し、
// Runnerインスタンスを返します。
func BuildScraperRunner(clientTimeout time.Duration, concurrency int) (*runner.Runner, error) {
	// 1. HTTP クライアント (Fetcher)
	fetcher := httpkit.New(clientTimeout)

	// 2. FeedParser の具体的な実装
	parser := feed.NewParser(fetcher)

	// 3. ScraperExecutor の具体的な実装
	extractor, err := extract.NewExtractor(fetcher)
	if err != nil {
		return nil, fmt.Errorf("Extractorの初期化エラー: %w", err)
	}
	scraperExecutor := scraper.NewParallelScraper(extractor, concurrency)

	// 4. Runner の初期化（依存関係を注入）
	return runner.NewRunner(parser, scraperExecutor), nil
}
