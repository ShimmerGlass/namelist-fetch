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
		start := time.Now()
		err := loadList(list)
		elapsed := time.Since(start)
		metricListReloadTime.WithLabelValues(list.Name).Set(elapsed.Seconds())

		if err != nil {
			slog.With("name", list.Name, "url", list.URL).Error("reload failed", "err", err.Error())
			metricListStatus.WithLabelValues(list.Name).Set(0)
		} else {
			metricListStatus.WithLabelValues(list.Name).Set(1)
			metricListLastFetch.WithLabelValues(list.Name).SetToCurrentTime()
		}
	}

	err := mergeLists()
	if err != nil {
		slog.Error("failed to merge lists", "err", err.Error())
	}
}

func loadList(list listConfig) (err error) {
	log := slog.With("name", list.Name, "url", list.URL)

	targetPath := filepath.Join(cfgTempDir, list.Name)
	targetPathTmp := targetPath + ".tmp"
	etagPath := targetPath + ".etag"

	targetTmp, err := os.OpenFile(targetPathTmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	defer func() {
		_ = targetTmp.Close()
		_ = os.Remove(targetPathTmp)
	}()

	etag, err := os.ReadFile(etagPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	log.Debug("loading list")
	req, err := http.NewRequest(http.MethodGet, list.URL, nil)
	if err != nil {
		return err
	}
	if len(etag) > 0 {
		req.Header.Add("If-None-Match", string(etag))
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode == http.StatusNotModified {
		log.Info("list did not change")
		return nil
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

	if etag := res.Header.Get("ETag"); etag != "" && !strings.HasPrefix(etag, "W/") {
		err = os.WriteFile(etagPath, []byte(etag), 0o644)
		if err != nil {
			return err
		}
	}

	log.Info("list updated")

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

		var outLine string
		switch cfgFormat {
		case formatNameOnly:
			outLine = fields[1]
		case formatAddressAndName:
			outLine = fmt.Sprintf("%s %s", fields[0], fields[1])
		}

		_, err := bout.WriteString(outLine)
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

	metricListSize.Reset()
	metricSize.Set(0)

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

func mergeList(list listConfig, to *bufio.Writer, seen map[string]struct{}) error {
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
		metricListSize.WithLabelValues(list.Name).Inc()

		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		metricSize.Inc()

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
