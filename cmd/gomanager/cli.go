package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/neko-server-dev/gomanager/internal/ban"
	"github.com/neko-server-dev/gomanager/internal/config"
	"github.com/neko-server-dev/gomanager/internal/errfile"
	"github.com/neko-server-dev/gomanager/internal/nftables"
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
	ttl := fs.String("ttl", "", "block duration (e.g. 1h, 30m)")
	expiresAt := fs.String("expires-at", "", "block until RFC3339 timestamp")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: IP address is required")
		printAddHelp()
		return 1
	}

	if *ttl != "" && *expiresAt != "" {
		fmt.Fprintln(os.Stderr, "error: specify either -ttl or -expires-at, not both")
		return 2
	}

	service, cleanup, err := openServiceFromConfig(*configPath)
	if err != nil {
		return 1
	}
	defer cleanup()

	var expiration *time.Time
	if *ttl != "" {
		t, err := ban.ParseTTL(*ttl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid ttl: %v\n", err)
			return 2
		}
		expiration = &t
	} else if *expiresAt != "" {
		t, err := ban.ParseExpiresAt(*expiresAt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid expires-at: %v\n", err)
			return 2
		}
		expiration = &t
	}

	ip := fs.Arg(0)
	if err := service.Add(ip, expiration); err != nil {
		return writeCLIError("add", err)
	}

	if expiration != nil {
		fmt.Printf("added: %s (expires %s)\n", ip, expiration.Format(time.RFC3339))
	} else {
		fmt.Printf("added: %s\n", ip)
	}
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

	service, cleanup, err := openServiceFromConfig(*configPath)
	if err != nil {
		return 1
	}
	defer cleanup()

	ip := fs.Arg(0)
	if err := service.Remove(ip); err != nil {
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

	service, cleanup, err := openServiceFromConfig(*configPath)
	if err != nil {
		return 1
	}
	defer cleanup()

	entries, err := service.List()
	if err != nil {
		return writeCLIError("list", err)
	}

	if len(entries) == 0 {
		fmt.Println("no blacklisted IPs")
		return 0
	}

	for _, entry := range entries {
		if entry.ExpiresAt != nil {
			fmt.Printf("%s\texpires %s\n", entry.IP, entry.ExpiresAt.Format(time.RFC3339))
		} else {
			fmt.Println(entry.IP)
		}
	}
	return 0
}

func openServiceFromConfig(configPath string) (*ban.Service, func(), error) {
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

	service, err := openBanService(manager, configPath)
	if err != nil {
		_ = manager.Close()
		errfile.Record("ban service init", err)
		fmt.Fprintf(os.Stderr, "ban service init failed: %v\n", err)
		return nil, nil, err
	}

	return service, func() { _ = manager.Close() }, nil
}

func writeCLIError(context string, err error) int {
	switch {
	case errors.Is(err, nftables.ErrInvalidIP):
		fmt.Fprintf(os.Stderr, "invalid IP: %v\n", err)
		return 2
	case errors.Is(err, nftables.ErrNotFound):
		fmt.Fprintf(os.Stderr, "not found: %v\n", err)
		return 2
	case errors.Is(err, ban.ErrInvalidTTL), errors.Is(err, ban.ErrInvalidExpiresAt):
		fmt.Fprintf(os.Stderr, "invalid expiration: %v\n", err)
		return 2
	default:
		errfile.Record(context, err)
		fmt.Fprintf(os.Stderr, "%s failed: %v\n", context, err)
		return 1
	}
}
