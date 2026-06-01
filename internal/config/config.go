package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultPath = "gomanager.yaml"

type Config struct {
	Host             string   `yaml:"host"`
	Port             int      `yaml:"port"`
	Addr             string   `yaml:"addr,omitempty"`
	TableName        string   `yaml:"table_name"`
	SetName          string   `yaml:"set_name"`
	ChainName        string   `yaml:"chain_name"`
	ForwardChainName string   `yaml:"forward_chain_name"`
	NICs             []string `yaml:"nics"`
}

func Default() Config {
	return Config{
		Host:             "0.0.0.0",
		Port:             8080,
		TableName:        "gomanager",
		SetName:          "blacklist",
		ChainName:        "input",
		ForwardChainName: "forward",
		NICs:             nil,
	}
}

// ListenAddr returns the HTTP listen address for Gin.
func (c *Config) ListenAddr() string {
	if c.Addr != "" {
		return c.Addr
	}
	host := c.Host
	if host == "" {
		host = Default().Host
	}
	port := c.Port
	if port == 0 {
		port = Default().Port
	}
	return netJoinHostPort(host, port)
}

func netJoinHostPort(host string, port int) string {
	if host == "" {
		return fmt.Sprintf(":%d", port)
	}
	if host[0] == '[' {
		return fmt.Sprintf("%s:%d", host, port)
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultPath
	}

	var cfg Config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg = Default()
		if err := Save(path, cfg); err != nil {
			return Config{}, fmt.Errorf("generate config: %w", err)
		}
	} else if err != nil {
		return Config{}, fmt.Errorf("stat config: %w", err)
	} else {
		data, err := os.ReadFile(path)
		if err != nil {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
		cfg = Default()
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("parse config: %w", err)
		}
		cfg.applyDefaults()
	}

	if err := EnsureServiceFile(path); err != nil {
		return Config{}, fmt.Errorf("generate service: %w", err)
	}

	return cfg, nil
}

func Save(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (c *Config) applyDefaults() {
	def := Default()
	if c.Host == "" && c.Addr == "" {
		c.Host = def.Host
	}
	if c.Port == 0 && c.Addr == "" {
		c.Port = def.Port
	}
	if c.TableName == "" {
		c.TableName = def.TableName
	}
	if c.SetName == "" {
		c.SetName = def.SetName
	}
	if c.ChainName == "" {
		c.ChainName = def.ChainName
	}
	if c.ForwardChainName == "" {
		c.ForwardChainName = def.ForwardChainName
	}
}
