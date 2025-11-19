package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func reloadLists() {
	for _, list := range cfgBlocklists {
		err := loadList(list)
		if err != nil {
			slog.With("name", list.Name, "url", list.URL).Error("reload failed", "err", err.Error())
			metricListStatus.WithLabelValues(list.Name).Set(0)
		} else {
			metricListStatus.WithLabelValues(list.Name).Set(1)
		}
	}

	err := mergeLists()
	if err != nil {
		slog.Error("failed to merge lists", "err", err.Error())
	}
}

func loadList(list blockList) error {
	log := slog.With("name", list.Name, "url", list.URL)
	start := time.Now()

	targetPath := filepath.Join(cfgTempDir, list.Name)
	targetPathTmp := targetPath + ".tmp"

	targetTmp, err := os.OpenFile(targetPathTmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	defer func() {
		_ = targetTmp.Close()
		_ = os.Remove(targetPathTmp)
	}()

	log.Debug("loading list")
	res, err := http.Get(list.URL)
	if err != nil {
		return err
	}

	if res.StatusCode >= 400 {
		return fmt.Errorf("invalid status code %d", res.StatusCode)
	}

	log.Debug("writing list", "target", targetPathTmp)
	err = transform(res.Body, targetTmp)
	if err != nil {
		return err
	}

	err = targetTmp.Close()
	if err != nil {
		return err
	}

	log.Debug("renaming", "from", targetPathTmp, "to", targetPath)
	err = os.Rename(targetPathTmp, targetPath)
	if err != nil {
		return err
	}

	elapsed := time.Since(start)
	log.Info("list reloaded", "time", time.Since(start))
	metricListReloadTime.WithLabelValues(list.Name).Set(elapsed.Seconds())
	metricListLastFetch.WithLabelValues(list.Name).SetToCurrentTime()

	return nil
}

func transform(in io.Reader, out io.Writer) error {
	bout := bufio.NewWriter(out)
	bin := bufio.NewScanner(in)

	for bin.Scan() {
		line := bin.Text()
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		if line[0] == '#' {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		_, err := bout.WriteString(fields[1])
		if err != nil {
			return err
		}

		err = bout.WriteByte('\n')
		if err != nil {
			return err
		}
	}

	err := bout.Flush()
	if err != nil {
		return err
	}

	return nil
}

func mergeLists() error {
	targetPath := cfgTargetFile
	targetPathTmp := targetPath + ".tmp"

	targetTmp, err := os.OpenFile(targetPathTmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	defer func() {
		_ = targetTmp.Close()
		_ = os.Remove(targetPathTmp)
	}()

	bTargetTmp := bufio.NewWriter(targetTmp)
	seen := map[string]struct{}{}

	for _, list := range cfgBlocklists {
		err := mergeList(list, bTargetTmp, seen)
		if err != nil {
			return err
		}
	}

	err = bTargetTmp.Flush()
	if err != nil {
		return err
	}

	err = targetTmp.Close()
	if err != nil {
		return err
	}

	err = os.Rename(targetPathTmp, targetPath)
	if err != nil {
		return err
	}

	slog.Info("lists merged")

	return nil
}

func mergeList(list blockList, to *bufio.Writer, seen map[string]struct{}) error {
	inPath := filepath.Join(cfgTempDir, list.Name)
	in, err := os.Open(inPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	defer func() { _ = in.Close() }()

	bin := bufio.NewScanner(in)

	for bin.Scan() {
		line := bin.Text()

		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}

		_, err := to.WriteString(line)
		if err != nil {
			return err
		}

		err = to.WriteByte('\n')
		if err != nil {
			return err
		}
	}

	return nil
}
