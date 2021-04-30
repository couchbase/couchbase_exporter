//  Copyright (c) 2020 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file
//  except in compliance with the License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing, software distributed under the
//  License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
//  either express or implied. See the License for the specific language governing permissions
//  and limitations under the License.

package collectors

import (
	"time"

	"github.com/couchbase/couchbase-exporter/pkg/log"
	"github.com/couchbase/couchbase-exporter/pkg/objects"
	"github.com/couchbase/couchbase-exporter/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
)

type ftsCollector struct {
	m      MetaCollector
	config *objects.CollectorConfig
}

func NewFTSCollector(client util.Client, config *objects.CollectorConfig) prometheus.Collector {
	if config == nil {
		config = objects.GetSearchCollectorDefaultConfig()
	}

	return &ftsCollector{
		m: MetaCollector{
			client: client,
			up: prometheus.NewDesc(
				prometheus.BuildFQName(config.Namespace, config.Subsystem, objects.DefaultUptimeMetric),
				objects.DefaultUptimeMetricHelp,
				[]string{objects.ClusterLabel},
				nil,
			),
			scrapeDuration: prometheus.NewDesc(
				prometheus.BuildFQName(config.Namespace, config.Subsystem, objects.DefaultScrapeDurationMetric),
				objects.DefaultScrapeDurationMetricHelp,
				[]string{objects.ClusterLabel},
				nil,
			),
		},
		config: config,
	}
}

// Describe all metrics.
func (c *ftsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.m.up
	ch <- c.m.scrapeDuration

	for _, value := range c.config.Metrics {
		if !value.Enabled {
			continue
		}

		ch <- value.GetPrometheusDescription(c.config.Namespace, c.config.Subsystem)
	}
}

// Collect all metrics.
func (c *ftsCollector) Collect(ch chan<- prometheus.Metric) {
	c.m.mutex.Lock()
	defer c.m.mutex.Unlock()

	start := time.Now()

	log.Info("Collecting fts metrics...")

	clusterName, err := c.m.client.ClusterName()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(c.m.up, prometheus.GaugeValue, 0, clusterName)

		log.Error("%s", err)

		return
	}

	ftsStats, err := c.m.client.Fts()
	if err != nil {
		ch <- prometheus.MustNewConstMetric(c.m.up, prometheus.GaugeValue, 0, clusterName)

		log.Error("failed to scrape FTS stats")

		return
	}

	for _, value := range c.config.Metrics {
		if value.Enabled {
			ch <- prometheus.MustNewConstMetric(
				value.GetPrometheusDescription(c.config.Namespace, c.config.Subsystem),
				prometheus.GaugeValue,
				last(ftsStats.Op.Samples[c.config.Subsystem+"_"+value.Name]),
				clusterName,
			)
		}
	}

	ch <- prometheus.MustNewConstMetric(c.m.up, prometheus.GaugeValue, 1, clusterName)
	ch <- prometheus.MustNewConstMetric(c.m.scrapeDuration, prometheus.GaugeValue, time.Since(start).Seconds(), clusterName)
}
