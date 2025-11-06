# ✍️ Web Text Pipe

[![Language](https://img.shields.io/badge/Language-Go-blue)](https://golang.org/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/shouni/web-text-pipe-go)](https://golang.org/)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/shouni/web-text-pipe-go)](https://github.com/shouni/web-text-pipe-go/tags)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 💡 概要 (About) — **Web記事からクリーンな本文を高速・堅牢に抽出するパイプラインツール**

**WebTextPipe** は、Go言語の強力な並列処理機能と、高性能なWebコンテンツ抽出ライブラリ **[`shouni/go-web-exact`](https://github.com/shouni/go-web-exact)** を統合した、**RSS/Atomフィード特化型の高速スクレイピングCLIツール**です。

**RSSフィードからの記事一括並列収集**と、**単一URLからの高精度な本文抽出**という2つの主要なワークフローを提供し、ノイズ（広告、ナビゲーションなど）を自動で排除したクリーンなテキストデータ収集パイプラインを構築します。

### 🌟 主な機能

* **高精度な本文抽出 (Core)**: 記事の本文のみを高精度で特定し、**ノイズ（広告、コメントなど）を排除**して整形済みテキストを返します。
* **RSSフィード並列収集 (`scraper`)**: 指定されたフィードURLから記事URLを抽出し、**最大同時実行数を制御しながら並列で**記事本文を一括収集します。
* **単一URL抽出 (`exact`)**: 開発やデバッグのために、**単一のURL**を指定し、その記事本文を直接抽出します。
* **堅牢な処理**: GoのGoroutineとセマフォ制御を利用し、`go-http-kit` を通じた**リトライ**と**タイムアウト制御**をサポートすることで、大規模な収集作業の**堅牢性**を確保します。

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
git clone git@github.com:shouni/web-text-pipe-go.git
cd web-text-pipe-go
go build -o bin/webtextpipe
```

### 2\. コマンド一覧

本ツールは、用途に応じて以下の2つのサブコマンドを提供します。

| コマンド | 説明 | 主な用途 |
| :--- | :--- | :--- |
| **`scraper`** | RSS/AtomフィードからURLを抽出し、記事本文を**並列で一括**取得します。 | 大量の記事データの定期的な収集。 |
| **`exact`** | **単一のURL**から本文を高精度で抽出し、結果を標準出力またはファイルに出力します。 | デバッグ、テスト、または単発の記事抽出。 |

-----

## 🚀 `scraper` コマンド (並列収集)

フィードからの並列抽出を実行します。

#### フラグ一覧 (scraper)

| フラグ | 短縮形 | 説明 |
| :--- | :--- | :--- |
| `--url` | `-u` | **必須**。解析対象のRSS/AtomフィードのURLを指定します。 |
| `--concurrency` | `-c` | 最大並列実行数。同時に処理する記事の数を制御します。**(Default: 5)** |
| `--timeout` | (なし) | **グローバル設定**。HTTPリクエストのタイムアウト時間（秒）。`(Default: 15)` |
| `--max-retries` | (なし) | **グローバル設定**。HTTPリクエストのリトライ最大回数。`(Default: 2)` |

#### 実行例 (scraper)

```bash
# Yahoo!ニュースのITカテゴリのRSSを読み込み、最大8並列、タイムアウト20秒で抽出
./bin/webtextpipe scraper \
    --url "https://news.yahoo.co.jp/rss/categories/it.xml" \
    --concurrency 8 \
    --timeout 20 # タイムアウトを20秒に延長
```

-----

## 🔍 `exact` コマンド (単一URL抽出)

単一のWebページから本文を抽出します。

#### フラグ一覧 (exact)

| フラグ | 短縮形 | 説明 |
| :--- | :--- | :--- |
| `--url` | `-u` | **必須**。抽出対象の単一WebページURLを指定します。 |
| `--output-file` | `-o` | 抽出されたテキストを保存するファイル名。省略時は標準出力に出力。 |
| `--timeout` | (なし) | **グローバル設定**。HTTPリクエストのタイムアウト時間（秒）。`(Default: 15)` |

#### 実行例 (exact)

```bash
# 指定URLから本文を抽出し、結果を output.txt に保存
./bin/webtextpipe exact \
    --url "https://example.com/some-article" \
    --output-file "output.txt"
```

-----

### 📜 ライセンス (License)

このプロジェクトは [MIT License](https://opensource.org/licenses/MIT) の下で公開されています。
