package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	clibase "github.com/shouni/go-cli-base"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-web-exact/v2/pkg/extract"
	"github.com/shouni/go-web-exact/v2/pkg/feed"
	"github.com/shouni/go-web-exact/v2/pkg/scraper"
	"github.com/spf13/cobra"
)

// --- コマンドラインフラグ変数 ---

var (
	feedURL     string // --url (-u) で受け取るフィードURL
	concurrency int    // --concurrency (-c) で受け取る並列実行数
)

// --- メインロジック ---

// runScrapePipeline は、並列スクレイピングを実行するメインロジックです。
func runScrapePipeline(ctx context.Context, urls []string, fetcher *httpkit.Client) error {
	// 1. Extractor の初期化
	// fetcher が httpkit.Client であり、extract.NewExtractor がそのラッパーまたはインターフェースを受け入れると仮定
	extractor, err := extract.NewExtractor(fetcher)
	if err != nil {
		return fmt.Errorf("Extractorの初期化エラー: %w", err)
	}

	// 2. Scraper の初期化
	parallelScraper := scraper.NewParallelScraper(extractor, concurrency)

	log.Printf("並列スクレイピング開始 (対象URL数: %d, 最大同時実行数: %d)\n", len(urls), concurrency)

	// 3. メインロジックの実行
	results := parallelScraper.ScrapeInParallel(ctx, urls)

	// 4. 結果の出力
	fmt.Println("\n--- 並列スクレイピング結果 ---")
	successCount := 0
	errorCount := 0

	for i, res := range results {
		if res.Error != nil {
			errorCount++
			log.Printf("❌ [%d] %s\n     エラー: %v\n", i+1, res.URL, res.Error)
		} else {
			successCount++
			// 詳細ログが有効な場合のみコンテンツの一部を出力
			if clibase.Flags.Verbose {
				fmt.Printf("✅ [%d] %s\n     抽出コンテンツの長さ: %d 文字\n     プレビュー: %s...\n",
					i+1, res.URL, len(res.Content), res.Content[:min(len(res.Content), 50)])
			} else {
				fmt.Printf("✅ [%d] %s\n     抽出コンテンツの長さ: %d 文字\n", i+1, res.URL, len(res.Content))
			}
		}
	}

	fmt.Println("-------------------------------")
	log.Printf("完了: 成功 %d 件, 失敗 %d 件\n", successCount, errorCount)

	return nil
}

// --- サブコマンド定義 ---

var scraperCmd = &cobra.Command{
	Use:   "scraper",
	Short: "RSSフィードからURLを抽出し、Webコンテンツを並列で取得・整形します",
	Long: `--url フラグで指定されたRSS/Atomフィードを解析し、含まれる記事のURLを抽出し、
指定された最大同時実行数で並列にコンテンツ抽出を実行します。`,
	Args: cobra.NoArgs,

	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. HTTPクライアントの初期化 (root.go のグローバルフラグを使用)
		clientTimeout := time.Duration(Flags.TimeoutSec) * time.Second
		// 修正: httpkit.New が time.Duration を引数に取るように修正 (ビルドエラーの解消)
		fetcher := httpkit.New(clientTimeout)
		if fetcher == nil {
			return fmt.Errorf("HTTPクライアントの初期化に失敗しました")
		}

		// 2. フィード解析器の初期化
		parser := feed.NewParser(fetcher)

		// 3. 全体実行コンテキストの設定
		overallTimeout := clientTimeout * 2

		ctx, cancel := context.WithTimeout(context.Background(), overallTimeout)
		defer cancel()

		// 4. フィードの取得とパースを実行
		log.Printf("フィードURLを解析中 (全体タイムアウト: %s): %s\n", overallTimeout, feedURL)
		rssFeed, err := parser.FetchAndParse(ctx, feedURL)
		if err != nil {
			return fmt.Errorf("フィードの処理エラー: %w", err)
		}

		// 5. FeedAdapter を使用して URL を抽象的に抽出
		// 提供された feed パッケージの FeedAdapter を利用
		adapter := feed.NewFeedAdapter(rssFeed)
		urls := adapter.GetLinks() // LinkSource インターフェース経由でリンクを取得

		log.Printf("フィードから %d 件のURLを抽出しました。\n", len(urls))

		if len(urls) == 0 {
			return fmt.Errorf("フィード (%s) から処理対象のURLが一つも抽出されませんでした", feedURL)
		}

		// 6. メインロジックの実行
		return runScrapePipeline(ctx, urls, fetcher)
	},
}

// --- フラグ初期化 ---

// initScraperFlags は、scraperCmdのフラグを設定し、root.goから呼び出されます。
func initScraperFlags() {
	scraperCmd.Flags().StringVarP(&feedURL, "url", "u", "https://news.yahoo.co.jp/rss/categories/it.xml", "解析対象のフィードURL (RSS/Atom)")
	scraperCmd.Flags().IntVarP(&concurrency, "concurrency", "c", scraper.DefaultMaxConcurrency, fmt.Sprintf("最大並列実行数"))
}

// ユーティリティ関数（Go 1.21未満の互換性のため）
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
