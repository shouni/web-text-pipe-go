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

// --- コマンドラインフラグ変数 ---

var (
	rawURL     string // --url (-u) で受け取る抽出対象の単一URL
	outputFile string // --output-file (-o) で受け取る出力ファイル名
)

// --- メインロジック ---

// runExactExtraction は、単一URLからの抽出を実行するロジックです。
func runExactExtraction(ctx context.Context, fetcher *httpkit.Client, url string) (text string, isBodyExtracted bool, err error) {
	// 1. Extractor の初期化
	// fetcher は httpkit.Client であり、extract.Fetcher インターフェースを満たす
	extractor, err := extract.NewExtractor(fetcher)
	if err != nil {
		return "", false, fmt.Errorf("Extractorの初期化エラー: %w", err)
	}

	// 2. 抽出の実行
	// runAudioQuery のようなラッパー関数は省略し、直接 FetchAndExtractText を呼び出す
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
		// 1. URLのバリデーション
		if rawURL == "" {
			return fmt.Errorf("エラー: 抽出対象のURL (--url, -u) は必須です")
		}

		// 2. HTTPクライアントの初期化 (root.go のグローバルフラグを使用)
		clientTimeout := time.Duration(Flags.TimeoutSec) * time.Second
		fetcher := httpkit.New(clientTimeout)

		// 3. 全体実行コンテキストの設定
		// 単一抽出のため、クライアントタイムアウトと同じ値を全体タイムアウトとする
		ctx, cancel := context.WithTimeout(context.Background(), clientTimeout)
		defer cancel()

		log.Printf("抽出処理開始 (URL: %s, タイムアウト: %s)\n", rawURL, clientTimeout)

		// 4. メインロジックの実行
		text, isBodyExtracted, err := runExactExtraction(ctx, fetcher, rawURL)
		if err != nil {
			return fmt.Errorf("コンテンツ抽出パイプラインの実行エラー: %w", err)
		}

		// 5. 結果の出力
		if !isBodyExtracted {
			log.Println("--- 本文抽出失敗 ---")
			log.Printf("本文は見つかりませんでしたが、取得したタイトル/メタ情報は以下です:\n%s\n", text)
			return nil
		}

		// 以前記憶した iohandler パッケージを使用して出力
		return iohandler.WriteOutputString(outputFile, text)
	},
}

// --- フラグ初期化 ---

// initExactFlags は、exactCmdのフラグを設定し、root.goから呼び出されます。
func initExactFlags() {
	// --url フラグ: 抽出対象URL（必須）
	exactCmd.Flags().StringVarP(&rawURL, "url", "u", "", "抽出対象の単一WebページURL (必須)")
	// --output-file フラグ: 出力ファイル名
	exactCmd.Flags().StringVarP(&outputFile, "output-file", "o", "", "抽出されたテキストを保存するファイル名。省略時は標準出力に出力。")

	// URLフラグを必須に設定
	exactCmd.MarkFlagRequired("url")
}
