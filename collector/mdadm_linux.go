// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !nomdadm
// +build !nomdadm

package collector

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

type mdadmCollector struct {
	logger log.Logger
}

func init() {
	registerCollector("mdadm", defaultEnabled, NewMdadmCollector)
}

// NewMdadmCollector returns a new Collector exposing raid statistics.
func NewMdadmCollector(logger log.Logger) (Collector, error) {
	return &mdadmCollector{logger}, nil
}

var (
	activeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "md", "state"),
		"Indicates the state of md-device.",
		[]string{"device"},
		prometheus.Labels{"state": "active"},
	)
	inActiveDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "md", "state"),
		"Indicates the state of md-device.",
		[]string{"device"},
		prometheus.Labels{"state": "inactive"},
	)
	recoveringDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "md", "state"),
		"Indicates the state of md-device.",
		[]string{"device"},
		prometheus.Labels{"state": "recovering"},
	)
	resyncDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "md", "state"),
		"Indicates the state of md-device.",
		[]string{"device"},
		prometheus.Labels{"state": "resync"},
	)
	checkDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "md", "state"),
		"Indicates the state of md-device.",
		[]string{"device"},
		prometheus.Labels{"state": "check"},
	)

	disksDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "md", "disks"),
		"Number of active/failed/spare disks of device.",
		[]string{"device", "state"},
		nil,
	)

	disksTotalDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "md", "disks_required"),
		"Total number of disks of device.",
		[]string{"device"},
		nil,
	)

	blocksTotalDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "md", "blocks"),
		"Total number of blocks on device.",
		[]string{"device"},
		nil,
	)

	blocksSyncedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "md", "blocks_synced"),
		"Number of blocks synced on device.",
		[]string{"device"},
		nil,
	)
	blocksSyncedPctDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "md", "blocks_synced_percent"),
		"Percentage of blocks synced on device.",
		[]string{"device"},
		nil,
	)
	syncTimeRemainingDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "md", "sync_time_remaining"),
		"Estimated finishing time for current sync in seconds.",
		[]string{"device"},
		nil,
	)
	blockSyncedSpeedDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "md", "blocks_synced_speed"),
		"current sync speed (in Kilobytes/sec)",
		[]string{"device"},
		nil,
	)
)

func (c *mdadmCollector) Update(ch chan<- prometheus.Metric) error {
	fs, err := procfs.NewFS(*procPath)

	if err != nil {
		return fmt.Errorf("failed to open procfs: %w", err)
	}

	mdStats, err := fs.MDStat()

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			level.Debug(c.logger).Log("msg", "Not collecting mdstat, file does not exist", "file", *procPath)
			return ErrNoData
		}

		return fmt.Errorf("error parsing mdstatus: %w", err)
	}

	for _, mdStat := range mdStats {
		level.Debug(c.logger).Log("msg", "collecting metrics for device", "device", mdStat.Name)

		stateVals := make(map[string]float64)
		stateVals[mdStat.ActivityState] = 1

		ch <- prometheus.MustNewConstMetric(
			disksTotalDesc,
			prometheus.GaugeValue,
			float64(mdStat.DisksTotal),
			mdStat.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			disksDesc,
			prometheus.GaugeValue,
			float64(mdStat.DisksDown),
			mdStat.Name,
			"down",
		)
		ch <- prometheus.MustNewConstMetric(
			disksDesc,
			prometheus.GaugeValue,
			float64(mdStat.DisksActive),
			mdStat.Name,
			"active",
		)
		ch <- prometheus.MustNewConstMetric(
			disksDesc,
			prometheus.GaugeValue,
			float64(mdStat.DisksFailed),
			mdStat.Name,
			"failed",
		)
		ch <- prometheus.MustNewConstMetric(
			disksDesc,
			prometheus.GaugeValue,
			float64(mdStat.DisksSpare),
			mdStat.Name,
			"spare",
		)
		ch <- prometheus.MustNewConstMetric(
			activeDesc,
			prometheus.GaugeValue,
			stateVals["active"],
			mdStat.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			inActiveDesc,
			prometheus.GaugeValue,
			stateVals["inactive"],
			mdStat.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			recoveringDesc,
			prometheus.GaugeValue,
			stateVals["recovering"],
			mdStat.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			resyncDesc,
			prometheus.GaugeValue,
			stateVals["resyncing"],
			mdStat.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			checkDesc,
			prometheus.GaugeValue,
			stateVals["checking"],
			mdStat.Name,
		)

		ch <- prometheus.MustNewConstMetric(
			blocksTotalDesc,
			prometheus.GaugeValue,
			float64(mdStat.BlocksTotal),
			mdStat.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			blocksSyncedDesc,
			prometheus.GaugeValue,
			float64(mdStat.BlocksSynced),
			mdStat.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			blocksSyncedPctDesc,
			prometheus.GaugeValue,
			float64(mdStat.BlocksSyncedPct),
			mdStat.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			syncTimeRemainingDesc,
			prometheus.GaugeValue,
			float64(mdStat.BlocksSyncedFinishTime*60),
			mdStat.Name,
		)
		ch <- prometheus.MustNewConstMetric(
			blockSyncedSpeedDesc,
			prometheus.GaugeValue,
			float64(mdStat.BlocksSyncedSpeed),
			mdStat.Name,
		)

	}

	return nil
}
