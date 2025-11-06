# Web Text Pipe

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/web-text-pipe-go)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/web-text-pipe-go)](https://github.com/shouni/web-text-pipe-go/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 💡 概要 (About) — **RSS/Atomフィードからクリーンな本文を並列で収集するパイプラインツール**

**WebTextPipe** は、Go言語の強力な並列処理機能と、高性能なWebコンテンツ抽出ライブラリ **[`shouni/go-web-exact`](https://github.com/shouni/go-web-exact)** を統合した、**RSS/Atomフィード特化型の高速スクレイピングCLIツール**です。

指定されたフィードURLから記事URLを抽出し、**最大同時実行数を制御しながら並列で各記事のメインコンテンツ（本文）を収集・整形**します。ノイズ（広告、ナビゲーションなど）を自動で排除し、クリーンなテキストデータ収集パイプラインを構築します。

### 🌟 主な機能

* **フィード解析**: RSSおよびAtomフィードを読み込み、含まれるすべての記事URLを抽出します。
* **高精度な本文抽出**: 記事の本文のみを高精度で特定し、**ノイズ（広告、コメントなど）を排除**して整形済みテキストを返します。
* **堅牢な並列処理**: GoのGoroutineとセマフォ制御を利用し、ネットワーク負荷を管理しながら、複数のURLを**高速かつ安定して同時処理**します。
* **柔軟な通信設定**: 独自のDI対応クライアント（`go-http-kit`）を通じて、リトライとタイムアウト制御をサポートし、大規模な収集作業の**堅牢性**を確保します。

-----

## ✨ 技術スタック (Technology Stack)

| 要素 | 技術 / ライブラリ | 役割 |
| :--- | :--- | :--- |
| **言語** | **Go (Golang)** | ツールの開発言語。並列処理と高速なCLI実行を実現します。 |
| **コア機能** | **`shouni/go-web-exact`** | フィード解析 (`pkg/feed`)、本文抽出 (`pkg/extract`)、並列スクレイピング (`pkg/scraper`) の中核機能。 |
| **通信基盤** | **`shouni/go-http-kit`** | すべてのネットワーク通信における**自動リトライ**と**タイムアウト制御**を提供します。 |
| **並行処理** | **Goroutines / セマフォ制御** | 記事ごとの抽出処理を並列化し、最大同時実行数を厳密に管理します。 |
| **CLI** | **Cobra** | コマンドラインインターフェースの構築。 |

-----

## 📦 使い方 (CLI Command)

### 1\. ビルドとインストール

```bash
# Go環境が整っていることを前提とします
git clone git@github.com:your-account/WebTextPipe.git
cd WebTextPipe
go build -o bin/webtextpipe
```

### 2\. 実行コマンド

`webtextpipe scraper` コマンドを使用して、フィードからの並列抽出を実行します。

```bash
./bin/webtextpipe scraper [flags]
```

#### フラグ一覧

| フラグ | 短縮形 | 説明 |
| :--- | :--- | :--- |
| `--url` | `-u` | **必須**。解析対象のRSS/AtomフィードのURLを指定します。 |
| `--concurrency` | `-c` | 最大並列実行数。同時に処理する記事の数を制御します。**(Default: 5)** |
| `--timeout` | (なし) | HTTPリクエストのタイムアウト時間（秒）。`(Default: 10)` |
| `--max-retries` | (なし) | HTTPリクエストのリトライ最大回数。`(Default: 2)` |

-----

## 🔊 実行例

### 例: 特定のRSSフィードから記事を並列抽出し、並列数を制御

Yahoo!ニュースのITカテゴリのRSSを読み込み、最大8並列でコンテンツ抽出を実行します。

```bash
./bin/webtextpipe scraper \
    --url "https://news.yahoo.co.jp/rss/categories/it.xml" \
    --concurrency 8 \
    --timeout 20 # タイムアウトを20秒に延長
```

### 期待される出力例

```
2025/11/07 04:00:00 フィードURLを解析中: https://news.yahoo.co.jp/rss/categories/it.xml
2025/11/07 04:00:01 20 件のURLを抽出しました。
2025/11/07 04:00:01 並列スクレイピング開始 (対象URL数: 20, 最大同時実行数: 8, 全体タイムアウト: 40s)
--- 並列スクレイピング結果 ---
✅ [1] https://news.yahoo.co.jp/articles/xxxxxxxxxxxxxxxxxxxxxx
     抽出コンテンツの長さ: 4520 文字
     プレビュー: 【記事タイトル】〇〇〇の市場動向が急変...
❌ [2] https://news.yahoo.co.jp/articles/yyyyyyyyyyyyyyyyyyyyyy
     エラー: コンテンツ抽出エラー (URL: ...): GET ...に失敗しました: 404 Not Found
...
-------------------------------
完了: 成功 18 件, 失敗 2 件
```

-----

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
