package builder

import (
	"fmt"
	"time"

	"github.com/shouni/web-text-pipe-go/pkg/runner"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-web-exact/v2/pkg/extract"
	"github.com/shouni/go-web-exact/v2/pkg/feed"
	"github.com/shouni/go-web-exact/v2/pkg/scraper"
)

// BuildReliableScraperExecutor は、必要な依存関係をすべて構築し、
// リトライ戦略を持つ ScraperExecutor (ReliableScraper) のインスタンスを返します。
func BuildReliableScraperExecutor(clientTimeout time.Duration, concurrency int) (*runner.ReliableScraper, error) {
	// HTTP クライアントを初期化
	fetcher := httpkit.New(clientTimeout)

	// コアな抽出エンジンを初期化
	extractor, err := extract.NewExtractor(fetcher)
	if err != nil {
		return nil, fmt.Errorf("Extractorの初期化エラー: %w", err)
	}

	// 並列実行とレート制限を担当するコアスクレイパーを初期化
	coreScraper := scraper.NewParallelScraper(extractor, concurrency, scraper.DefaultScrapeRateLimit)

	// リトライ戦略と遅延処理を担当する ReliableScraper を構築
	return runner.NewReliableScraper(coreScraper, extractor), nil
}

// BuildScraperRunner は、必要な設定値に基づいて、runner.Runnerの依存関係をすべて構築し、
// Runnerインスタンスを返します。
func BuildScraperRunner(clientTimeout time.Duration, concurrency int) (*runner.Runner, error) {
	// HTTP クライアントを初期化
	fetcher := httpkit.New(clientTimeout)

	// FeedParser を初期化
	parser := feed.NewParser(fetcher)

	// ReliableScraperExecutor を構築
	reliableScraperExecutor, err := BuildReliableScraperExecutor(clientTimeout, concurrency)
	if err != nil {
		return nil, err
	}

	// Runner を初期化
	return runner.NewRunner(parser, reliableScraperExecutor), nil
}
