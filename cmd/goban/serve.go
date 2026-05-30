package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/neko-server-dev/goban/internal/config"
	"github.com/neko-server-dev/goban/internal/errfile"
	"github.com/neko-server-dev/goban/internal/handler"
	"github.com/neko-server-dev/goban/internal/nftables"
)

func runServe(args []string) int {
	if wantsHelp(args) {
		printServeHelp()
		return 0
	}

	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = printServeHelp
	configPath := fs.String("config", config.DefaultPath, "path to config file")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	errfile.Init(*configPath)

	cfg, err := config.Load(*configPath)
	if err != nil {
		errfile.Record("config load", err)
		log.Fatalf("config load failed: %v", err)
	}

	manager, err := openManager(cfg)
	if err != nil {
		errfile.Record("nftables init", err)
		log.Fatalf("nftables init failed: %v", err)
	}
	defer manager.Close()

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.SetTrustedProxies(nil)
	r.Use(gin.Logger())
	r.Use(corsMiddleware())
	r.Use(gin.CustomRecovery(func(c *gin.Context, recovered any) {
		errfile.Record("panic", fmt.Errorf("%v", recovered))
		c.AbortWithStatusJSON(500, gin.H{"error": "internal server error"})
	}))

	r.GET("/health", handler.Health)

	api := r.Group("/api/v1")
	handler.NewBlacklistHandler(manager).Register(api)

	addr := cfg.ListenAddr()
	log.Printf("goban listening on %s", addr)
	if err := r.Run(addr); err != nil {
		errfile.Record("server", err)
		log.Fatalf("server failed: %v", err)
	}
	return 0
}

func openManager(cfg config.Config) (*nftables.Manager, error) {
	return nftables.New(nftables.Config{
		TableName:        cfg.TableName,
		SetName:          cfg.SetName,
		ChainName:        cfg.ChainName,
		ForwardChainName: cfg.ForwardChainName,
		NICs:             cfg.NICs,
	})
}
