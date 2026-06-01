package errfile

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const DefaultFileName = "gomanager.error"

var (
	mu   sync.Mutex
	path string
)

func Init(configPath string) {
	if configPath == "" {
		configPath = "gomanager.yaml"
	}
	path = filepath.Join(filepath.Dir(configPath), DefaultFileName)
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
}

func Record(context string, err error) {
	if err == nil {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	target := path
	if target == "" {
		target = DefaultFileName
	}

	f, openErr := os.OpenFile(target, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if openErr != nil {
		return
	}
	defer f.Close()

	_, _ = fmt.Fprintf(f, "[%s] %s: %v\n", time.Now().Format(time.RFC3339), context, err)
}
