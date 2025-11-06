package cmd

import (
	"log"
	"time"

	clibase "github.com/shouni/go-cli-base"
	"github.com/spf13/cobra"
)

// --- グローバル定数 ---

const (
	appName           = "web-text-pipe"
	defaultTimeoutSec = 15 // 秒 (並列スクレイピングを考慮して初期値を15秒に設定)
	defaultMaxRetries = 2  // デフォルトのリトライ回数
)

// --- グローバル変数とフラグ構造体 ---

// AppFlags はこのアプリケーション固有の永続フラグを保持
type AppFlags struct {
	TimeoutSec int // --timeout HTTPリクエストのタイムアウト
	MaxRetries int // --max-retries リトライ回数
}

var Flags AppFlags // アプリケーション固有フラグにアクセスするためのグローバル変数

// --- 初期化とロジック (clibaseへのコールバックとして利用) ---

// addAppPersistentFlags は、アプリケーション固有の永続フラグをルートコマンドに追加します。
// clibase.CustomFlagFunc のシグネチャに一致します。
func addAppPersistentFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().IntVar(
		&Flags.TimeoutSec,
		"timeout",
		defaultTimeoutSec,
		"HTTPリクエストのタイムアウト時間（秒）",
	)
	rootCmd.PersistentFlags().IntVar(
		&Flags.MaxRetries,
		"max-retries",
		defaultMaxRetries,
		"HTTPリクエストのリトライ最大回数",
	)
}

// initAppPreRunE は、アプリケーション固有のPersistentPreRunEです。
// clibaseの共通処理の後に実行されます。
// NOTE: clibase.Flags.Verbose はこの関数実行前に設定済み
func initAppPreRunE(cmd *cobra.Command, args []string) error {
	timeout := time.Duration(Flags.TimeoutSec) * time.Second

	// clibase.Flags の利用
	if clibase.Flags.Verbose {
		log.Printf("HTTPクライアントのタイムアウトを設定しました (Timeout: %s)。", timeout)
		log.Printf("HTTPクライアントのリトライ回数を設定しました (MaxRetries: %d)。", Flags.MaxRetries)
	}

	// WebTextPipeには必須の環境変数チェックはないため、ここでは特別なロジックを追加しません。
	return nil
}

// initCmdFlags は、すべてのサブコマンドのフラグを初期化します。
func initCmdFlags() {
	initScraperFlags()
	initExactFlags()
}

// --- エントリポイント ---

// Execute は、rootCmd を実行するメイン関数です。
func Execute() {
	// initCmdFlags でサブコマンドのフラグを登録
	initCmdFlags()

	// ルートコマンドの構築と実行を clibase に全て委任
	// clibase.Execute を使用して、アプリケーションの初期化、フラグ設定、サブコマンドの登録を一括で行う
	clibase.Execute(
		appName,
		addAppPersistentFlags, // カスタムフラグの追加コールバック
		initAppPreRunE,        // カスタムPersistentPreRunEコールバック
		scraperCmd,
		exactCmd,
	)
}
