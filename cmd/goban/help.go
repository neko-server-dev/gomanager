package main

import (
	"fmt"
	"os"
)

func runHelp(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 0
	}

	switch args[0] {
	case "serve":
		printServeHelp()
	case "add":
		printAddHelp()
	case "remove", "rm", "del", "delete":
		printRemoveHelp()
	case "list", "ls":
		printListHelp()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printUsage()
		return 1
	}
	return 0
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `goban - IP ブラックリスト管理

Usage:
  goban [command] [options]

Commands:
  serve              HTTP API サーバーを起動（デフォルト）
  add <ip>           IP をブラックリストに追加
  remove <ip>        IP をブラックリストから削除
  list               ブラックリスト一覧を表示
  help [command]     ヘルプを表示

Global Options:
  -config path       設定ファイルのパス（デフォルト: goban.yaml）

Examples:
  goban serve
  goban add 192.168.1.100
  goban remove 192.168.1.100
  goban list
  goban help add

`)
}

func printServeHelp() {
	fmt.Fprintf(os.Stderr, `Usage:
  goban serve [-config goban.yaml]
  goban [-config goban.yaml]

HTTP API サーバーを起動します。
デフォルトでは 0.0.0.0:8080 で全インターフェースに公開されます。

Config (goban.yaml):
  host: 0.0.0.0    バインドする IP（外部公開: 0.0.0.0、ローカルのみ: 127.0.0.1）
  port: 8080       リッスンポート

Options:
  -config path    設定ファイルのパス（デフォルト: goban.yaml）

API Endpoints:
  GET    /health
  GET    /api/v1/blacklist
  POST   /api/v1/blacklist        body: {"ip": "192.168.1.100"}
                                  optional: "ttl": "24h" or "expires_at": "2026-06-01T00:00:00Z"
  DELETE /api/v1/blacklist/:ip

Examples:
  goban serve
  goban -config /etc/goban/goban.yaml
  curl -X POST http://localhost:8080/api/v1/blacklist -d '{"ip":"10.0.0.1","ttl":"1h"}'

`)
}

func printAddHelp() {
	fmt.Fprintf(os.Stderr, `Usage:
  goban add <ip> [-config goban.yaml] [-ttl duration] [-expires-at RFC3339]

指定した IP をブラックリストに追加します。
IPv4 / IPv6 に対応しています。
期限を省略した場合は無期限でブロックします。

Arguments:
  ip    拒否する IP アドレス

Options:
  -config path       設定ファイルのパス（デフォルト: goban.yaml）
  -ttl duration      ブロック期間（例: 1h, 30m）
  -expires-at time   ブロック解除時刻（RFC3339）

Examples:
  goban add 192.168.1.100
  goban add 192.168.1.100 -ttl 24h
  goban add 2001:db8::1 -expires-at 2026-06-01T00:00:00Z
  goban add 10.0.0.5 -config /etc/goban/goban.yaml

`)
}

func printRemoveHelp() {
	fmt.Fprintf(os.Stderr, `Usage:
  goban remove <ip> [-config goban.yaml]

指定した IP をブラックリストから削除します。

Aliases:
  remove, rm, del, delete

Arguments:
  ip    削除する IP アドレス

Options:
  -config path    設定ファイルのパス（デフォルト: goban.yaml）

Examples:
  goban remove 192.168.1.100
  goban rm 10.0.0.5

`)
}

func printListHelp() {
	fmt.Fprintf(os.Stderr, `Usage:
  goban list [-config goban.yaml]

ブラックリストに登録されている IP 一覧を表示します。
期限付きエントリは解除予定時刻も表示します。

Aliases:
  list, ls

Options:
  -config path    設定ファイルのパス（デフォルト: goban.yaml）

Examples:
  goban list
  goban ls -config /etc/goban/goban.yaml

`)
}

func wantsHelp(args []string) bool {
	for _, a := range args {
		switch a {
		case "-h", "--help", "help":
			return true
		}
	}
	return false
}
