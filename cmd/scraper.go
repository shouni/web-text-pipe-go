package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"web-text-pipe-go/pkg/scraperrunner"

	"github.com/shouni/go-cli-base"
	"github.com/shouni/go-http-kit/pkg/httpkit"
	"github.com/shouni/go-web-exact/v2/pkg/extract"
	"github.com/shouni/go-web-exact/v2/pkg/feed"
	"github.com/shouni/go-web-exact/v2/pkg/scraper"
	"github.com/shouni/go-web-exact/v2/pkg/types"
	"github.com/spf13/cobra"
)

// --- ロジック: 結果の出力 (I/O) ---

// printResults は、scraperrunnerから受け取った結果をCLIに出力します。
func printResults(results []types.URLResult, verbose bool) {
	fmt.Println("\n--- 並列スクレイピング結果 ---")
	successCount := 0
	errorCount := 0

	for i, res := range results {
		if res.Error != nil {
			errorCount++
			log.Printf("❌ [%d] %s\n     エラー: %v\n", i+1, res.URL, res.Error)
		} else {
			successCount++
			if verbose {
				fmt.Printf("✅ [%d] %s\n     抽出コンテンツの長さ: %d 文字\n     プレビュー: %s...\n",
					i+1, res.URL, len(res.Content), res.Content[:min(len(res.Content), 50)])
			} else {
				fmt.Printf("✅ [%d] %s\n     抽出コンテンツの長さ: %d 文字\n", i+1, res.URL, len(res.Content))
			}
		}
	}

	fmt.Println("-------------------------------")
	log.Printf("完了: 成功 %d 件, 失敗 %d 件\n", successCount, errorCount)
}

// --- ファクトリ関数 (依存性構築) ---

// newScraperRunner は、必要な依存関係をすべて構築し、設定済みの scraperrunner.Runner
// インスタンスを返します。この関数が依存性注入(DI)コンポーネントの構築を担当します。
func newScraperRunner(clientTimeout time.Duration, concurrency int) (*scraperrunner.Runner, error) {
	// 1. 依存関係の構築に必要な HTTP クライアント
	fetcher := httpkit.New(clientTimeout)

	// 2. FeedParser の具体的な実装 (依存性を注入)
	parser := feed.NewParser(fetcher)

	// 3. ScraperExecutor の具体的な実装 (Extractor と Concurrency を使用)
	extractor, err := extract.NewExtractor(fetcher)
	if err != nil {
		return nil, fmt.Errorf("Extractorの初期化エラー: %w", err)
	}
	// scraper.NewParallelScraper は並列数(concurrency)を設定
	scraperExecutor := scraper.NewParallelScraper(extractor, concurrency)

	// 4. Runner の初期化（依存関係を注入）
	runner := scraperrunner.NewRunner(parser, scraperExecutor)

	return runner, nil
}

// --- サブコマンド定義 ---

var scraperCmd = &cobra.Command{
	Use:   "scraper",
	Short: "RSSフィードからURLを抽出し、Webコンテンツを並列で取得・整形します",
	Long: `--url フラグで指定されたRSS/Atomフィードを解析し、含まれる記事のURLを抽出し、
指定された最大同時実行数で並列にコンテンツ抽出を実行します。`,
	Args: cobra.NoArgs,

	RunE: func(cmd *cobra.Command, args []string) error {
		// 1. フラグ値の取得と設定の構築
		feedURL, _ := cmd.Flags().GetString("url")
		concurrency, _ := cmd.Flags().GetInt("concurrency")
		// root.go のグローバルフラグからタイムアウトを取得
		clientTimeout := time.Duration(Flags.TimeoutSec) * time.Second

		// 2. 💡 ファクトリ関数を呼び出し、Runnerを取得
		runner, err := newScraperRunner(clientTimeout, concurrency)
		if err != nil {
			return err // エラーの場合は即座に返す
		}

		// 3. 実行コンテキストと設定の準備
		ctx := context.Background()
		config := scraperrunner.RunnerConfig{
			FeedURL:                  feedURL,
			ClientTimeout:            clientTimeout,
			OverallTimeoutMultiplier: 2, // クライアントタイムアウトの2倍を全体のタイムアウトとする
		}

		// 4. ScrapeAndRun の呼び出し
		results, err := runner.ScrapeAndRun(ctx, config)
		if err != nil {
			return err
		}

		// 5. 結果の出力
		printResults(results, clibase.Flags.Verbose)

		return nil
	},
}

// --- フラグ初期化 ---

func initScraperFlags() {
	scraperCmd.Flags().StringP("url", "u", "https://news.yahoo.co.jp/rss/categories/it.xml", "解析対象のフィードURL (RSS/Atom)")

	scraperCmd.Flags().IntP("concurrency", "c",
		scraperrunner.DefaultMaxConcurrency,
		"最大並列実行数 (デフォルト: 5)")
}
