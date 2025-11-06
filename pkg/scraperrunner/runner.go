package scraperrunner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/shouni/go-web-exact/v2/pkg/feed"
	"github.com/shouni/go-web-exact/v2/pkg/types"
)

// DefaultMaxConcurrency は、pkg/scraper の定数を公開するために使用
const DefaultMaxConcurrency = 6

// FeedParser はフィードを取得し、パースする責務を持つインターフェース
type FeedParser interface {
	FetchAndParse(ctx context.Context, feedURL string) (*gofeed.Feed, error)
}

// ScraperExecutor は並列スクレイピングを実行する責務を持つインターフェース
type ScraperExecutor interface {
	ScrapeInParallel(ctx context.Context, urls []string) []types.URLResult
}

// Runner 構造体は、具体的な実装（依存関係）を保持する
type Runner struct {
	FeedParser      FeedParser      // feed.Parser のインスタンス
	ScraperExecutor ScraperExecutor // scraper.ParallelScraper のインスタンス
}

// NewRunner は依存関係を注入して Runner を初期化する関数
func NewRunner(parser FeedParser, scraperExecutor ScraperExecutor) *Runner {
	return &Runner{
		FeedParser:      parser,
		ScraperExecutor: scraperExecutor,
	}
}

// --- 実行設定とメインロジック ---

// RunnerConfig は実行に必要な設定を保持します。
type RunnerConfig struct {
	FeedURL                  string
	ClientTimeout            time.Duration
	OverallTimeoutMultiplier int // 全体タイムアウト倍率 (例: 2)
}

// ScrapeAndRun は、フィードの解析から並列スクレイピングまでの一連の処理を実行し、
// 結果データとエラーを返します。
func (r *Runner) ScrapeAndRun(ctx context.Context, config RunnerConfig) ([]types.URLResult, error) {

	overallTimeout := config.ClientTimeout * time.Duration(config.OverallTimeoutMultiplier)

	runCtx, cancel := context.WithTimeout(ctx, overallTimeout)
	defer cancel()

	// 2. フィードの取得とパースを実行 (r.FeedParser を使用)
	slog.Info(
		"フィードURLを解析中",
		slog.Duration("overall_timeout", overallTimeout),
		slog.String("feed_url", config.FeedURL),
	)

	rssFeed, err := r.FeedParser.FetchAndParse(runCtx, config.FeedURL)
	if err != nil {
		slog.Error(
			"フィードの処理エラーが発生しました",
			slog.Any("error", err),
			slog.String("feed_url", config.FeedURL),
		)
		return nil, fmt.Errorf("フィードの処理エラー: %w", err)
	}

	// 3. URLを抽出
	adapter := feed.NewFeedAdapter(rssFeed)
	urls := adapter.GetLinks()

	slog.Info(
		"フィードからURLを抽出",
		slog.Int("extracted_count", len(urls)),
	)

	if len(urls) == 0 {
		return nil, fmt.Errorf("フィード (%s) から処理対象のURLが一つも抽出されませんでした", config.FeedURL)
	}

	// 4. パイプラインの実行 (r.ScraperExecutor を使用)
	slog.Info(
		"並列スクレイピング実行中",
		slog.Int("total_urls", len(urls)),
	)

	return r.ScraperExecutor.ScrapeInParallel(runCtx, urls), nil
}
