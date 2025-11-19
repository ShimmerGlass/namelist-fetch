package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// config env vars
const (
	envListenAddr = "NLF_LISTEN_ADDR"
	envTargetFile = "NLF_TARGET_FILE"
	envTempDir    = "NLF_TEMP_DIR"
	envListPrefix = "NLF_LIST_" // prefix, eg: NLF_LIST_
	envInterval   = "NLF_INTERVAL"
)

// config vars
var (
	cfgListenAddr string
	cfgTargetFile string
	cfgTempDir    string
	cfgBlocklists []listConfig
	cfgInterval   time.Duration
)

type listConfig struct {
	Name string
	URL  string
}

func loadConfig() error {
	var err error

	cfgListenAddr = os.Getenv(envListenAddr)

	if v := os.Getenv(envInterval); v != "" {
		cfgInterval, err = time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("cannot parse %s: %w", envInterval, err)
		}
	} else {
		cfgInterval = 4 * time.Hour
	}

	cfgBlocklists, err = getLists()
	if err != nil {
		return err
	}

	if v := os.Getenv(envTempDir); v != "" {
		cfgTempDir = os.Getenv(envTempDir)
	} else {
		cfgTempDir = "/tmp"
	}

	if v := os.Getenv(envTargetFile); v != "" {
		cfgTargetFile = os.Getenv(envTargetFile)
	} else {
		return fmt.Errorf("%s not set", envTargetFile)
	}

	http.DefaultClient.Timeout = 30 * time.Second

	return nil
}

func getLists() ([]listConfig, error) {
	lists := []listConfig{}

	for _, ev := range os.Environ() {
		k, listURL, _ := strings.Cut(ev, "=")
		_, name, ok := strings.Cut(k, envListPrefix)
		if !ok {
			continue
		}

		_, err := url.Parse(listURL)
		if err != nil {
			return nil, fmt.Errorf("blocklist %q: invalid url: %w", name, err)
		}

		lists = append(lists, listConfig{
			Name: name,
			URL:  listURL,
		})
	}

	if len(lists) == 0 {
		return nil, fmt.Errorf("no blocklist configured, set %sXXX", envListPrefix)
	}

	return lists, nil
}
