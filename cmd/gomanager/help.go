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
	fmt.Fprintf(os.Stderr, `gomanager - サーバー管理

Usage:
  gomanager [command] [options]

Commands:
  serve              HTTP API サーバーを起動（デフォルト）
  add <ip>           IP をブラックリストに追加
  remove <ip>        IP をブラックリストから削除
  list               ブラックリスト一覧を表示
  help [command]     ヘルプを表示

Global Options:
  -config path       設定ファイルのパス（デフォルト: gomanager.yaml）

Examples:
  gomanager serve
  gomanager add 192.168.1.100
  gomanager remove 192.168.1.100
  gomanager list
  gomanager help add

`)
}

func printServeHelp() {
	fmt.Fprintf(os.Stderr, `Usage:
  gomanager serve [-config gomanager.yaml]
  gomanager [-config gomanager.yaml]

HTTP API サーバーを起動します。
デフォルトでは 0.0.0.0:8080 で全インターフェースに公開されます。

Config (gomanager.yaml):
  host: 0.0.0.0    バインドする IP（外部公開: 0.0.0.0、ローカルのみ: 127.0.0.1）
  port: 8080       リッスンポート

Options:
  -config path    設定ファイルのパス（デフォルト: gomanager.yaml）

API Endpoints:
  GET    /health
  GET    /api/v1/blacklist
  POST   /api/v1/blacklist        body: {"ip": "192.168.1.100"}
                                  optional: "ttl": "24h" or "expires_at": "2026-06-01T00:00:00Z"
  DELETE /api/v1/blacklist/:ip
  GET    /api/v1/network/bandwidth
         インターフェースごとの送受信バイト数・パケット数と転送速度（bytes/sec）
         rx_bytes / tx_bytes は再起動をまたいだ累積量（gomanager.bandwidth.json に永続化）
         session_rx_bytes / session_tx_bytes は直近の OS 起動以降の量
         2 回目以降のリクエストで rx_bytes_per_sec / tx_bytes_per_sec を返します
         config の nics が設定されている場合は対象 NIC のみ（未設定時は lo 以外）

Examples:
  gomanager serve
  gomanager -config /etc/gomanager/gomanager.yaml
  curl -X POST http://localhost:8080/api/v1/blacklist -d '{"ip":"10.0.0.1","ttl":"1h"}'
  curl http://localhost:8080/api/v1/network/bandwidth

`)
}

func printAddHelp() {
	fmt.Fprintf(os.Stderr, `Usage:
  gomanager add <ip> [-config gomanager.yaml] [-ttl duration] [-expires-at RFC3339]

指定した IP をブラックリストに追加します。
IPv4 / IPv6 に対応しています。
期限を省略した場合は無期限でブロックします。

Arguments:
  ip    拒否する IP アドレス

Options:
  -config path       設定ファイルのパス（デフォルト: gomanager.yaml）
  -ttl duration      ブロック期間（例: 1h, 30m）
  -expires-at time   ブロック解除時刻（RFC3339）

Examples:
  gomanager add 192.168.1.100
  gomanager add 192.168.1.100 -ttl 24h
  gomanager add 2001:db8::1 -expires-at 2026-06-01T00:00:00Z
  gomanager add 10.0.0.5 -config /etc/gomanager/gomanager.yaml

`)
}

func printRemoveHelp() {
	fmt.Fprintf(os.Stderr, `Usage:
  gomanager remove <ip> [-config gomanager.yaml]

指定した IP をブラックリストから削除します。

Aliases:
  remove, rm, del, delete

Arguments:
  ip    削除する IP アドレス

Options:
  -config path    設定ファイルのパス（デフォルト: gomanager.yaml）

Examples:
  gomanager remove 192.168.1.100
  gomanager rm 10.0.0.5

`)
}

func printListHelp() {
	fmt.Fprintf(os.Stderr, `Usage:
  gomanager list [-config gomanager.yaml]

ブラックリストに登録されている IP 一覧を表示します。
期限付きエントリは解除予定時刻も表示します。

Aliases:
  list, ls

Options:
  -config path    設定ファイルのパス（デフォルト: gomanager.yaml）

Examples:
  gomanager list
  gomanager ls -config /etc/gomanager/gomanager.yaml

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
