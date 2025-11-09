package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shouni/web-text-pipe-go/pkg/scraper/builder"
	"github.com/shouni/web-text-pipe-go/pkg/scraper/runner"

	"github.com/shouni/go-cli-base"
	"github.com/shouni/go-web-exact/v2/pkg/types"
	"github.com/spf13/cobra"
)

// --- ロジック: 結果の出力 (I/O) ---

// printResults は、runnerから受け取った結果をCLIに出力します。
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
		clientTimeout := time.Duration(Flags.TimeoutSec) * time.Second

		// 2. Runnerを取得
		runnerInstance, err := builder.BuildScraperRunner(clientTimeout, concurrency)
		if err != nil {
			return err
		}

		// 3. 実行コンテキストと設定の準備
		ctx := context.Background()
		config := runner.RunnerConfig{
			FeedURL:                  feedURL,
			ClientTimeout:            clientTimeout,
			OverallTimeoutMultiplier: 60,
		}

		// 4. ScrapeAndRun の呼び出し
		// 修正: 戻り値の型を *runner.RunnerResult に変更
		runnerResult, err := runnerInstance.ScrapeAndRun(ctx, config)
		if err != nil {
			return err
		}

		// 抽出結果の確認
		if len(runnerResult.Results) == 0 {
			// runner.ScrapeAndRun が既にエラーチェックをしているはずだが、念のため
			log.Printf("エラー: スクレピング結果が一つもありませんでした。フィードタイトル: %s\n", runnerResult.FeedTitle)
		}

		// 5. 結果の出力
		// 修正: RunnerResult から Results スライスを取り出して渡す
		printResults(runnerResult.Results, clibase.Flags.Verbose)

		return nil
	},
}

// --- フラグ初期化 ---

func initScraperFlags() {
	// runner.DefaultMaxConcurrency は以前記憶した runner パッケージでエクスポートされています
	scraperCmd.Flags().StringP("url", "u", "https://news.yahoo.co.jp/rss/categories/it.xml", "解析対象のフィードURL (RSS/Atom)")
	scraperCmd.Flags().IntP("concurrency", "c", runner.DefaultMaxConcurrency, "最大並列実行数 (デフォルト: 6)")
}
