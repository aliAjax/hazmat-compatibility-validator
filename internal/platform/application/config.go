package application

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr      string
	AdminToken      string
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
	BatchWorkers    int
	MaxBatchLine    int
}

func DefaultConfig() Config {
	return Config{ListenAddr: ":32710", AdminToken: "hazmat-admin-token", RequestTimeout: 30 * time.Second, ShutdownTimeout: 10 * time.Second, BatchWorkers: 4, MaxBatchLine: 2 << 20}
}
func Load(path string) (Config, error) {
	var c Config
	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return c, err
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				return c, fmt.Errorf("invalid config line")
			}
			if err := apply(&c, strings.TrimSpace(parts[0]), strings.Trim(strings.TrimSpace(parts[1]), "\"'")); err != nil {
				return c, err
			}
		}
		if err := scanner.Err(); err != nil {
			return c, err
		}
	}
	env := map[string]string{"listen_addr": os.Getenv("HAZMAT_LISTEN_ADDR"), "admin_token": os.Getenv("HAZMAT_ADMIN_TOKEN"), "request_timeout": os.Getenv("HAZMAT_REQUEST_TIMEOUT"), "shutdown_timeout": os.Getenv("HAZMAT_SHUTDOWN_TIMEOUT"), "batch_workers": os.Getenv("HAZMAT_BATCH_WORKERS"), "max_batch_line": os.Getenv("HAZMAT_MAX_BATCH_LINE")}
	for k, v := range env {
		if v != "" {
			if err := apply(&c, k, v); err != nil {
				return c, err
			}
		}
	}
	return c, c.Validate()
}
func apply(c *Config, key, value string) error {
	switch key {
	case "listen_addr":
		c.ListenAddr = value
	case "admin_token":
		c.AdminToken = value
	case "request_timeout":
		v, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		c.RequestTimeout = v
	case "shutdown_timeout":
		v, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		c.ShutdownTimeout = v
	case "batch_workers":
		v, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.BatchWorkers = v
	case "max_batch_line":
		v, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		c.MaxBatchLine = v
	default:
		return fmt.Errorf("unknown configuration key %q", key)
	}
	return nil
}
func (c Config) Validate() error {
	if c.ListenAddr == "" || c.AdminToken == "" || c.RequestTimeout <= 0 || c.ShutdownTimeout <= 0 || c.BatchWorkers < 1 || c.BatchWorkers > 64 || c.MaxBatchLine < 1024 {
		return fmt.Errorf("invalid configuration")
	}
	return nil
}
