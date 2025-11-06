package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shouni/go-http-kit/pkg/httpkit"
	iohandler "github.com/shouni/go-utils/iohandler"
	"github.com/shouni/go-web-exact/v2/pkg/extract"
	"github.com/spf13/cobra"
)

// --- メインロジック ---

// runExactExtraction は、単一URLからの抽出を実行するロジックです。
// 💡 修正: fetcherの型を具体的な型 (*httpkit.Client) からインターフェース (extract.Fetcher) に変更。
func runExactExtraction(ctx context.Context, fetcher extract.Fetcher, url string) (text string, isBodyExtracted bool, err error) {
	// 1. Extractor の初期化
	// Extractor は内部で extract.Fetcher に依存するため、引数として受け取った fetcher をそのまま渡す。
	extractor, err := extract.NewExtractor(fetcher)
	if err != nil {
		return "", false, fmt.Errorf("Extractorの初期化エラー: %w", err)
	}

	// 2. 抽出の実行
	text, isBodyExtracted, err = extractor.FetchAndExtractText(ctx, url)
	if err != nil {
		// エラーのラッピング
		return "", false, fmt.Errorf("コンテンツ抽出エラー (URL: %s): %w", url, err)
	}

	return text, isBodyExtracted, nil
}

// --- サブコマンド定義 ---

var exactCmd = &cobra.Command{
	Use:   "exact",
	Short: "単一のURLからWebコンテンツの本文を高精度で抽出します",
	Long:  `単一のURLを指定し、ノイズを除去したクリーンなメインコンテンツ（本文）を高精度で抽出します。`,

	Args: cobra.NoArgs,

	RunE: func(cmd *cobra.Command, args []string) error {

		var rawURL string
		var outputFile string

		// 実行前にフラグ値を取得（cobraのライフサイクルで設定されている）
		rawURL, _ = cmd.Flags().GetString("url")
		outputFile, _ = cmd.Flags().GetString("output-file")

		// 1. URLのバリデーション
		if rawURL == "" {
			// MarkFlagRequired("url") を使用しているため、通常はcobraが捕捉するが、念のため
			return fmt.Errorf("エラー: 抽出対象のURL (--url, -u) は必須です")
		}

		// 2. HTTPクライアントの初期化 (root.go のグローバルフラグを使用)
		clientTimeout := time.Duration(Flags.TimeoutSec) * time.Second
		// httpkit.New の戻り値は *httpkit.Client であり、これが extract.Fetcher インターフェースを満たす。
		fetcher := httpkit.New(clientTimeout)
		if fetcher == nil {
			return fmt.Errorf("HTTPクライアントの初期化に失敗しました")
		}

		// 3. 全体実行コンテキストの設定
		ctx, cancel := context.WithTimeout(context.Background(), clientTimeout)
		defer cancel()

		log.Printf("抽出処理開始 (URL: %s, タイムアウト: %s)\n", rawURL, clientTimeout)

		// 4. メインロジックの実行
		// fetcher (*httpkit.Client) は runExactExtraction が要求する extract.Fetcher インターフェースを満たすため、型変換なしで渡せる。
		text, isBodyExtracted, err := runExactExtraction(ctx, fetcher, rawURL)
		if err != nil {
			return fmt.Errorf("コンテンツ抽出パイプラインの実行エラー: %w", err)
		}

		// 5. 結果の出力
		if !isBodyExtracted {
			log.Println("--- 本文抽出失敗 ---")
			// 💡 修正: テキストが空でなければ、タイトル/メタ情報として出力
			if text != "" {
				log.Printf("本文は見つかりませんでしたが、以下の情報が抽出されました:\n%s\n", text)
			} else {
				log.Println("本文、タイトル、メタ情報のいずれも抽出されませんでした。")
			}
			return nil
		}

		// iohandler パッケージを使用して出力
		return iohandler.WriteOutputString(outputFile, text)
	},
}

// --- フラグ初期化 ---

// 💡 修正: フラグ変数を削除し、cobraの関数に直接渡すように変更
func initExactFlags() {
	// フラグ変数をパッケージレベルから削除したため、RunEで値を取得できるように、Flags()を直接操作する。
	exactCmd.Flags().StringP("url", "u", "", "抽出対象の単一WebページURL (必須)")
	exactCmd.Flags().StringP("output-file", "o", "", "抽出されたテキストを保存するファイル名。省略時は標準出力に出力。")

	// URLフラグを必須に設定
	exactCmd.MarkFlagRequired("url")
}
