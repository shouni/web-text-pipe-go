package runner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/shouni/go-web-exact/v2/pkg/feed"
	"github.com/shouni/go-web-exact/v2/pkg/types"
)

// ----------------------------------------------------------------
// インターフェース定義 (DI対象)
// ----------------------------------------------------------------

// FeedParser はフィードの取得とパース機能を提供します。
type FeedParser interface {
	FetchAndParse(ctx context.Context, feedURL string) (*gofeed.Feed, error)
}

// ScraperExecutor はスクレイピングの実行機能を提供します。
// ReliableScraper がこのインターフェースを実装します。
type ScraperExecutor interface {
	ScrapeInParallel(ctx context.Context, urls []string) []types.URLResult
}

// ----------------------------------------------------------------
// ワークフロー管理者 (Runner)
// ----------------------------------------------------------------

// Runner は、フィードの取得、URLの抽出、スクレイピング実行という一連の処理フローを管理します。
type Runner struct {
	FeedParser      FeedParser
	ScraperExecutor ScraperExecutor // リトライ機能を持つ ReliableScraper が注入される
}

// NewRunner は依存関係を注入して Runner を初期化する関数
func NewRunner(parser FeedParser, scraperExecutor ScraperExecutor) *Runner {
	return &Runner{
		FeedParser:      parser,
		ScraperExecutor: scraperExecutor,
	}
}

// RunnerConfig は実行に必要な設定を保持します。
type RunnerConfig struct {
	FeedURL                  string
	ClientTimeout            time.Duration
	OverallTimeoutMultiplier int
}

// RunnerResult は ScrapeAndRun の実行結果とメタデータを保持します。
type RunnerResult struct {
	FeedTitle string
	Results   []types.URLResult
	TitlesMap map[string]string // URLをキー、記事タイトルを値とするマップ
}

// ScrapeAndRun は、フィードの解析から並列スクレイピングまでの一連の処理を実行し、
// 結果データとメタデータを RunnerResult として返します。
func (r *Runner) ScrapeAndRun(ctx context.Context, config RunnerConfig) (*RunnerResult, error) {

	overallTimeout := config.ClientTimeout * time.Duration(config.OverallTimeoutMultiplier)

	runCtx, cancel := context.WithTimeout(ctx, overallTimeout)
	defer cancel()

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

	adapter := feed.NewFeedAdapter(rssFeed)
	urls := adapter.GetLinks()
	titlesMap := adapter.GetTitlesMap()

	slog.Info(
		"フィードからURLを抽出",
		slog.Int("extracted_count", len(urls)),
	)

	if len(urls) == 0 {
		return nil, fmt.Errorf("フィード (%s) から処理対象のURLが一つも抽出されませんでした", config.FeedURL)
	}

	slog.Info(
		"並列スクレイピング実行中",
		slog.Int("total_urls", len(urls)),
	)

	// ScraperExecutor (ReliableScraper) を呼び出し
	results := r.ScraperExecutor.ScrapeInParallel(runCtx, urls)

	runnerResult := &RunnerResult{
		FeedTitle: rssFeed.Title,
		Results:   results,
		TitlesMap: titlesMap,
	}

	return runnerResult, nil
}
