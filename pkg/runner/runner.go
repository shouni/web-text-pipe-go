package runner

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/shouni/go-web-exact/v2/pkg/feed"
	"github.com/shouni/go-web-exact/v2/pkg/scraper"
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

// Extractor はコンテンツ抽出ロジックの抽象化です。リトライ時に単体で使用されます。
// (extract.Extractorがこれを実装すると想定)
type Extractor interface {
	FetchAndExtractText(ctx context.Context, url string) (string, bool, error)
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

	// 2. フィードの取得とパースを実行 (r.FeedParser を使用)
	slog.Info(
		"フィードURLを解析中",
		slog.Duration("overall_timeout", overallTimeout),
		slog.String("feed_url", config.FeedURL),
	)

	// rssFeed は *gofeed.Feed 型
	rssFeed, err := r.FeedParser.FetchAndParse(runCtx, config.FeedURL)
	if err != nil {
		slog.Error(
			"フィードの処理エラーが発生しました",
			slog.Any("error", err),
			slog.String("feed_url", config.FeedURL),
		)
		return nil, fmt.Errorf("フィードの処理エラー: %w", err)
	}

	// 3. URLとタイトルの抽出とマップの構築
	adapter := feed.NewFeedAdapter(rssFeed) // feed.FeedAdapter は外部ライブラリにあると仮定
	urls := adapter.GetLinks()
	titlesMap := adapter.GetTitlesMap()

	slog.Info(
		"フィードからURLを抽出",
		slog.Int("extracted_count", len(urls)),
	)

	if len(urls) == 0 {
		return nil, fmt.Errorf("フィード (%s) から処理対象のURLが一つも抽出されませんでした", config.FeedURL)
	}

	// 4. パイプラインの実行 (r.ScraperExecutor を使用)
	// ここで ReliableScraper.ScrapeInParallel が呼び出され、リトライ処理が行われる
	slog.Info(
		"並列スクレイピング実行中",
		slog.Int("total_urls", len(urls)),
	)

	results := r.ScraperExecutor.ScrapeInParallel(runCtx, urls)

	// 5. 結果オブジェクトの作成
	runnerResult := &RunnerResult{
		FeedTitle: rssFeed.Title,
		Results:   results,
		TitlesMap: titlesMap,
	}

	return runnerResult, nil
}

// ----------------------------------------------------------------
// 信頼性エグゼキュータ (ReliableScraper) - リトライ戦略の実装
// ----------------------------------------------------------------

const (
	// InitialScrapeDelay は初回並列スクレイピング後の待機時間 (負荷軽減)
	InitialScrapeDelay = 5 * time.Second
	// RetryScrapeDelay はリトライ前の待機時間 (負荷軽減)
	RetryScrapeDelay = 3 * time.Second
	// PhaseContent はログ用フェーズ名
	PhaseContent = "ContentExtraction"
)

// ReliableScraper は ScraperExecutor インターフェースを実装し、
// 下位のコアスクレイパー (ParallelScraper) の上にリトライと遅延のロジックを重ねて信頼性を高めます。
type ReliableScraper struct {
	baseScraper scraper.Scraper // 例: scraper.ParallelScraper のインターフェース
	extractor   Extractor       // 例: extract.Extractor のインターフェース
}

// NewReliableScraper は ReliableScraper の新しいインスタンスを作成します。
func NewReliableScraper(baseScraper scraper.Scraper, extractor Extractor) *ReliableScraper {
	return &ReliableScraper{
		baseScraper: baseScraper,
		extractor:   extractor,
	}
}

// ScrapeInParallel は、URLリストに対して並列スクレイピングと、失敗したURLに対するリトライを実行します。
func (r *ReliableScraper) ScrapeInParallel(ctx context.Context, urls []string) []types.URLResult {
	slog.Info("フェーズ1 - Webコンテンツの並列抽出を開始します。")

	// 1. 初回並列実行 (レート制限機能を持つ下位の ParallelScraper を呼び出す)
	results := r.baseScraper.ScrapeInParallel(ctx, urls)

	// 2. 無条件遅延 (負荷軽減)
	slog.Info("並列抽出が完了しました。次の処理に進む前に待機します。", slog.String("phase", PhaseContent), slog.Duration("delay", InitialScrapeDelay))
	time.Sleep(InitialScrapeDelay)

	// 3. 結果の分類
	successfulResults, failedURLs := classifyResults(results)
	initialSuccessfulCount := len(successfulResults)

	// 4. 失敗URLの上位レベルリトライ処理 (Extractor を使用した順次リトライ)
	if len(failedURLs) > 0 {
		retriedSuccessfulResults, retryErr := r.processFailedURLs(ctx, failedURLs, RetryScrapeDelay)
		if retryErr != nil {
			slog.Warn("失敗URLのリトライ処理中にエラーが発生しました", slog.Any("error", retryErr))
		}
		successfulResults = append(successfulResults, retriedSuccessfulResults...)
	}

	// 5. 最終チェックとログ
	if len(successfulResults) == 0 {
		slog.Error("処理可能なWebコンテンツを一件も取得できませんでした。URLを確認してください。")
		return []types.URLResult{}
	}

	slog.Info("コンテンツ取得結果",
		slog.Int("successful", len(successfulResults)),
		slog.Int("total", len(urls)),
		slog.Int("initial_successful", initialSuccessfulCount),
		slog.Int("retry_successful", len(successfulResults)-initialSuccessfulCount),
		slog.String("phase", PhaseContent),
	)

	return successfulResults
}

// processFailedURLsは、失敗したURLに対し、指定された遅延時間後に順次リトライを実行します。
func (r *ReliableScraper) processFailedURLs(ctx context.Context, failedURLs []string, retryDelay time.Duration) ([]types.URLResult, error) {
	slog.Warn("抽出に失敗したURLがありました。待機後、順次リトライを開始します。", slog.Int("count", len(failedURLs)), slog.Duration("delay", retryDelay))
	time.Sleep(retryDelay)

	var retriedSuccessfulResults []types.URLResult
	slog.Info("失敗URLの順次リトライを開始します。")

	for _, url := range failedURLs {
		slog.Info("リトライ中", slog.String("url", url))

		// 注入された Extractor を使用し、単一URLを再取得
		content, hasBodyFound, err := r.extractor.FetchAndExtractText(ctx, url)

		var extractErr error
		if err != nil {
			extractErr = fmt.Errorf("コンテンツの抽出に失敗しました: %w", err)
		} else if content == "" || !hasBodyFound {
			extractErr = fmt.Errorf("URL %s から有効な本文を抽出できませんでした", url)
		}

		if extractErr != nil {
			formattedErr := formatErrorLog(extractErr)
			slog.Error("リトライでもURLの抽出に失敗しました", slog.String("url", url), slog.String("error", formattedErr))
		} else {
			slog.Info("URLの抽出がリトライで成功しました", slog.String("url", url))
			retriedSuccessfulResults = append(retriedSuccessfulResults, types.URLResult{
				URL:     url,
				Content: content,
				Error:   nil,
			})
		}
	}
	return retriedSuccessfulResults, nil
}

// classifyResultsは並列抽出の結果を成功と失敗に分類します。
func classifyResults(results []types.URLResult) (successfulResults []types.URLResult, failedURLs []string) {
	for _, res := range results {
		if res.Error != nil || res.Content == "" {
			failedURLs = append(failedURLs, res.URL)
		} else {
			successfulResults = append(successfulResults, res)
		}
	}
	return successfulResults, failedURLs
}

// formatErrorLogは、冗長なエラーメッセージを短縮します。
func formatErrorLog(err error) string {
	errMsg := err.Error()
	if idx := strings.Index(errMsg, ", ボディ: <!"); idx != -1 {
		errMsg = errMsg[:idx]
	}
	if idx := strings.LastIndex(errMsg, "最終エラー:"); idx != -1 {
		return strings.TrimSpace(errMsg[idx:])
	}
	return errMsg
}
