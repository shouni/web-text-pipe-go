package scraperrunner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-web-exact/v2/pkg/extract"
	"github.com/shouni/go-web-exact/v2/pkg/feed"
	"github.com/shouni/go-web-exact/v2/pkg/scraper"
	"github.com/shouni/go-web-exact/v2/pkg/types"
)

const DefaultMaxConcurrency = scraper.DefaultMaxConcurrency

// --- 構造体 ---

// RunnerConfig は実行に必要な設定を保持します。
type RunnerConfig struct {
	FeedURL                  string
	Concurrency              int
	ClientTimeout            time.Duration
	OverallTimeoutMultiplier int // 全体タイムアウト倍率 (例: 2)
}

// ScrapeAndRun は、フィードの解析から並列スクレイピングまでの一連の処理を実行し、
// 結果データとエラーを返します。
func ScrapeAndRun(ctx context.Context, config RunnerConfig) ([]types.URLResult, error) {
	// 1. HTTPクライアントの初期化 (extract.Fetcher を満たす)
	fetcher := httpkit.New(config.ClientTimeout)

	// 2. フィード解析器の初期化
	parser := feed.NewParser(fetcher)

	// 3. 全体実行コンテキストの設定 (RunEから渡されたコンテキストを使用)
	overallTimeout := config.ClientTimeout * time.Duration(config.OverallTimeoutMultiplier)

	runCtx, cancel := context.WithTimeout(ctx, overallTimeout)
	defer cancel()

	// 4. フィードの取得とパースを実行
	slog.Info(
		"フィードURLを解析中",
		slog.Duration("overall_timeout", overallTimeout),
		slog.String("feed_url", config.FeedURL),
	)
	rssFeed, err := parser.FetchAndParse(runCtx, config.FeedURL)
	if err != nil {
		// ログは呼び出し元 (cmd/scraper.go) で処理されるため、ここではエラーを返します。
		return nil, fmt.Errorf("フィードの処理エラー: %w", err)
	}

	// 5. URLを抽出
	adapter := feed.NewFeedAdapter(rssFeed)
	urls := adapter.GetLinks()

	slog.Info(
		"フィードからURLを抽出",
		slog.Int("extracted_count", len(urls)),
	)

	if len(urls) == 0 {
		return nil, fmt.Errorf("フィード (%s) から処理対象のURLが一つも抽出されませんでした", config.FeedURL)
	}

	// 6. パイプラインの構築と実行
	return runPipeline(runCtx, urls, fetcher, config.Concurrency)
}

// runPipeline は、Webスクレイピングの並列実行ロジックのみを担当します。
func runPipeline(ctx context.Context, urls []string, fetcher *httpkit.Client, concurrency int) ([]types.URLResult, error) {
	// 1. Extractor の初期化
	extractor, err := extract.NewExtractor(fetcher)
	if err != nil {
		return nil, fmt.Errorf("Extractorの初期化エラー: %w", err)
	}

	// 2. Scraper の初期化
	parallelScraper := scraper.NewParallelScraper(extractor, concurrency)

	slog.Info(
		"並列スクレイピング実行中",
		slog.Int("concurrency", concurrency),
		slog.Int("total_urls", len(urls)),
	)

	// 3. メインロジックの実行
	results := parallelScraper.ScrapeInParallel(ctx, urls)

	return results, nil
}
