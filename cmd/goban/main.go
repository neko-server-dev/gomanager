package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && !isFlag(os.Args[1]) {
		switch os.Args[1] {
		case "serve":
			os.Exit(runServe(os.Args[2:]))
		case "add":
			os.Exit(runAdd(os.Args[2:]))
		case "remove", "rm", "del", "delete":
			os.Exit(runRemove(os.Args[2:]))
		case "list", "ls":
			os.Exit(runList(os.Args[2:]))
		case "help", "-h", "--help":
			os.Exit(runHelp(os.Args[2:]))
		default:
			fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
			printUsage()
			os.Exit(1)
		}
		return
	}

	os.Exit(runServe(os.Args[1:]))
}

func isFlag(s string) bool {
	return len(s) > 0 && s[0] == '-'
}
