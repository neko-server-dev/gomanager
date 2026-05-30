package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/neko-server-dev/goban/internal/config"
	"github.com/neko-server-dev/goban/internal/errfile"
	"github.com/neko-server-dev/goban/internal/nftables"
)

func runAdd(args []string) int {
	if wantsHelp(args) {
		printAddHelp()
		return 0
	}

	fs := flag.NewFlagSet("add", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = printAddHelp
	configPath := fs.String("config", config.DefaultPath, "path to config file")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: IP address is required")
		printAddHelp()
		return 1
	}

	manager, cleanup, err := openManagerFromConfig(*configPath)
	if err != nil {
		return 1
	}
	defer cleanup()

	ip := fs.Arg(0)
	if err := manager.Add(ip); err != nil {
		return writeCLIError("add", err)
	}

	fmt.Printf("added: %s\n", ip)
	return 0
}

func runRemove(args []string) int {
	if wantsHelp(args) {
		printRemoveHelp()
		return 0
	}

	fs := flag.NewFlagSet("remove", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = printRemoveHelp
	configPath := fs.String("config", config.DefaultPath, "path to config file")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: IP address is required")
		printRemoveHelp()
		return 1
	}

	manager, cleanup, err := openManagerFromConfig(*configPath)
	if err != nil {
		return 1
	}
	defer cleanup()

	ip := fs.Arg(0)
	if err := manager.Remove(ip); err != nil {
		return writeCLIError("remove", err)
	}

	fmt.Printf("removed: %s\n", ip)
	return 0
}

func runList(args []string) int {
	if wantsHelp(args) {
		printListHelp()
		return 0
	}

	fs := flag.NewFlagSet("list", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = printListHelp
	configPath := fs.String("config", config.DefaultPath, "path to config file")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "error: unexpected arguments")
		printListHelp()
		return 1
	}

	manager, cleanup, err := openManagerFromConfig(*configPath)
	if err != nil {
		return 1
	}
	defer cleanup()

	ips, err := manager.List()
	if err != nil {
		return writeCLIError("list", err)
	}

	if len(ips) == 0 {
		fmt.Println("no blacklisted IPs")
		return 0
	}

	for _, ip := range ips {
		fmt.Println(ip)
	}
	return 0
}

func openManagerFromConfig(configPath string) (*nftables.Manager, func(), error) {
	errfile.Init(configPath)

	cfg, err := config.Load(configPath)
	if err != nil {
		errfile.Record("config load", err)
		fmt.Fprintf(os.Stderr, "config load failed: %v\n", err)
		return nil, nil, err
	}

	manager, err := openManager(cfg)
	if err != nil {
		errfile.Record("nftables init", err)
		fmt.Fprintf(os.Stderr, "nftables init failed: %v\n", err)
		return nil, nil, err
	}

	return manager, func() { _ = manager.Close() }, nil
}

func writeCLIError(context string, err error) int {
	switch {
	case errors.Is(err, nftables.ErrInvalidIP):
		fmt.Fprintf(os.Stderr, "invalid IP: %v\n", err)
		return 2
	case errors.Is(err, nftables.ErrNotFound):
		fmt.Fprintf(os.Stderr, "not found: %v\n", err)
		return 2
	default:
		errfile.Record(context, err)
		fmt.Fprintf(os.Stderr, "%s failed: %v\n", context, err)
		return 1
	}
}
