package runner

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/shouni/go-web-exact/v2/pkg/scraper"
	"github.com/shouni/go-web-exact/v2/pkg/types"
)

// ----------------------------------------------------------------
// 信頼性エグゼキュータ (ReliableScraper) - リトライ戦略の実装
// ----------------------------------------------------------------

const (
	InitialScrapeDelay = 5 * time.Second
	RetryScrapeDelay   = 3 * time.Second
	PhaseContent       = "ContentExtraction"
)

// Extractor はコンテンツ抽出ロジックの抽象化です。リトライ時の単体抽出に使用します。
type Extractor interface {
	FetchAndExtractText(ctx context.Context, url string) (string, bool, error)
}

// ReliableScraper は ScraperExecutor インターフェースを実装し、
// リトライと遅延のロジックを重ねて信頼性を高めます。
type ReliableScraper struct {
	baseScraper scraper.Scraper // scraper.ParallelScraper のインターフェース
	extractor   Extractor       // extract.Extractor のインターフェース
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

	// 1. 初回並列実行
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
