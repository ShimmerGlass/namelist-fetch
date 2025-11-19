package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	metricListStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dnscrypt_blocklist_list_status",
		Help: "1=Successful reload 0=Failure",
	}, []string{"name"})

	metricListReloadTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dnscrypt_blocklist_list_reload_time_seconds",
		Help: "Time taken to reload a list",
	}, []string{"name"})

	metricListLastFetch = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dnscrypt_blocklist_list_last_fetch_unix",
		Help: "Last unix timestamp when the list was refreshed",
	}, []string{"name"})
)
