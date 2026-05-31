package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/neko-server-dev/gomanager/internal/ban"
	"github.com/neko-server-dev/gomanager/internal/config"
	"github.com/neko-server-dev/gomanager/internal/errfile"
	"github.com/neko-server-dev/gomanager/internal/handler"
	"github.com/neko-server-dev/gomanager/internal/nftables"
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

	service, err := openBanService(manager, *configPath)
	if err != nil {
		errfile.Record("ban service init", err)
		log.Fatalf("ban service init failed: %v", err)
	}
	if n, err := service.CleanupExpired(); err != nil {
		errfile.Record("expiration cleanup", err)
	} else if n > 0 {
		log.Printf("removed %d expired blacklist entry(ies) on startup", n)
	}
	service.RunCleanupLoop(time.Minute, func(err error) {
		errfile.Record("expiration cleanup", err)
	})

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
	handler.NewBlacklistHandler(service).Register(api)

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

func openBanService(manager *nftables.Manager, configPath string) (*ban.Service, error) {
	return ban.NewService(manager, ban.NewStore(configPath))
}
