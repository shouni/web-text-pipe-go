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

// BuildScraperRunner は、必要な設定値に基づいて、runner.Runnerの依存関係をすべて構築し、
// Runnerインスタンスを返します。
func BuildScraperRunner(clientTimeout time.Duration, concurrency int) (*runner.Runner, error) {
	// 1. HTTP クライアント (Fetcher)
	// 最も下層のネットワークアクセスを担うコンポーネント。
	fetcher := httpkit.New(clientTimeout)

	// 2. FeedParser の具体的な実装 (FEED層)
	// Fetcherに依存し、フィードの取得・解析という役割を持つ。
	parser := feed.NewParser(fetcher) // 責務: フィード解析

	// 3. コアな抽出エンジン (EXTRACTOR層)
	// Fetcherに依存し、単一URLの本文抽出という役割を持つ。リトライ時にも利用される。
	extractor, err := extract.NewExtractor(fetcher)
	if err != nil {
		return nil, fmt.Errorf("Extractorの初期化エラー: %w", err)
	}

	// 4. 並列実行とレート制限を担当するコアスクレイパー (SCRAPER層)
	// Extractorに依存し、同時実行制御とレート制限という役割を持つ。リトライロジックは含まない。
	coreScraper := scraper.NewParallelScraper(extractor, concurrency, scraper.DefaultScrapeRateLimit)

	// 5. リトライ戦略と遅延処理を担当する ReliableScraper (RUNNER戦略層)
	// コアスクレイパーとExtractorに依存し、リトライ戦略という上位の役割を持つ。
	reliableScraperExecutor := runner.NewReliableScraper(coreScraper, extractor)

	// 6. Runner の初期化（RUNNERワークフロー層）
	// FeedParserとReliableScraperExecutorという最上位の依存関係を注入し、ワークフロー管理者を構築。
	return runner.NewRunner(parser, reliableScraperExecutor), nil // 責務: ワークフロー管理
}
