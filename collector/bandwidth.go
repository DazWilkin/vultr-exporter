package collector

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultr/govultr/v3"
)

var (
	_ prometheus.Collector = (*BandwidthCollector)(nil)
)

// BandwidthCollector represents Account Bandwidth
type BandwidthCollector struct {
	System System
	Client *govultr.Client
	Log    logr.Logger

	// Single metric that captures all bandwidth-related values
	// Using labels for period (current/previous/projected) and metric type (gb_in/gb_out/etc)
	// This allows for easier querying and aggregation across periods and metric types
	Value *prometheus.Desc
}

// NewBandwidthCollector creates a new BandwidthCollector
func NewBandwidthCollector(s System, client *govultr.Client, log logr.Logger) BandwidthCollector {
	return BandwidthCollector{
		System: s,
		Client: client,
		Log:    log,

		Value: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, "account_bandwidth", "value"),
			"Bandwidth metric value",
			[]string{"period", "metric", "unit"},
			nil,
		),
	}
}

// Collect implements Prometheus' Collector interface and is used to collect metrics
func (c BandwidthCollector) Collect(ch chan<- prometheus.Metric) {
	bandwidth, _, err := c.Client.Account.GetBandwidth(context.Background())
	if err != nil {
		c.Log.Error(err, "Account.GetBandwidth")
		return
	}

	// Collect metrics for each period
	c.collectPeriod(ch, bandwidth.PreviousMonth, "previous")
	c.collectPeriod(ch, bandwidth.CurrentMonthToDate, "current")
	c.collectPeriod(ch, bandwidth.CurrentMonthProjected, "projected")
}

// collectPeriod collects metrics for a specific bandwidth period
func (c BandwidthCollector) collectPeriod(ch chan<- prometheus.Metric, p govultr.AccountBandwidthPeriod, period string) {
	// Map of metric name to its value and unit
	metrics := map[string]struct {
		value float64
		unit  string
	}{
		"gb_in":                       {float64(p.GBIn), "GB"},
		"gb_out":                      {float64(p.GBOut), "GB"},
		"total_instance_hours":        {float64(p.TotalInstanceHours), "hours"},
		"total_instance_count":        {float64(p.TotalInstanceCount), "count"},
		"instance_bandwidth_credits":  {float64(p.InstanceBandwidthCredits), "credits"},
		"free_bandwidth_credits":      {float64(p.FreeBandwidthCredits), "credits"},
		"purchased_bandwidth_credits": {float64(p.PurchasedBandwidthCredits), "credits"},
		"overage":                     {float64(p.Overage), "GB"},
		"overage_unit_cost":           {float64(p.OverageUnitCost), "USD"},
		"overage_cost":                {float64(p.OverageCost), "USD"},
	}

	for metricName, data := range metrics {
		ch <- prometheus.MustNewConstMetric(
			c.Value,
			prometheus.GaugeValue,
			data.value,
			[]string{period, metricName, data.unit}...,
		)
	}
}

// Describe implements Prometheus' Collector interface and is used to describe metrics
func (c BandwidthCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Value
}

// BandwidthPeriodCollector represents BandwidthPeriod
// It does not implement prometheus.Collector interface because Collect accepts BandwidthPeriod
// This is necessary because the Account.GetBandwidth returns a Bandwidth struct
// Containing multiple (3) copies of BandwidthPeriod structs
type BandwidthPeriodCollector struct {
	System System
	Client *govultr.Client
	Log    logr.Logger

	GBIn                      *prometheus.Desc
	GBOut                     *prometheus.Desc
	TotalInstanceHours        *prometheus.Desc
	TotalInstanceCount        *prometheus.Desc
	InstanceBandwidthCredits  *prometheus.Desc
	FreeBandwidthCredits      *prometheus.Desc
	PurchasedBandwidthCredits *prometheus.Desc
	Overage                   *prometheus.Desc
	OverageUnitCost           *prometheus.Desc
	OverageCost               *prometheus.Desc
}

// NewBandwidthPeriod creates a new BandwidthPeriodCollector
func NewBandwidthPeriod(s System, client *govultr.Client, log logr.Logger) BandwidthPeriodCollector {
	return BandwidthPeriodCollector{
		System: s,
		Client: client,
		Log:    log,

		GBIn: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "gb_in"),
			"Ingress Bandwidth in GB",
			[]string{},
			nil,
		),
		GBOut: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "gb_out"),
			"Egress Bandwidth in GB",
			[]string{},
			nil,
		),
		TotalInstanceHours: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "total_instance_hours"),
			"Total Instance Hours",
			[]string{},
			nil,
		),
		TotalInstanceCount: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "total_instance_count"),
			"Total Instance Count",
			[]string{},
			nil,
		),
		InstanceBandwidthCredits: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "instance_bandwidth_credits"),
			"Instance Bandwidth Credits",
			[]string{},
			nil,
		),
		FreeBandwidthCredits: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "free_bandwidth_credits"),
			"Free Bandwidth Credits",
			[]string{},
			nil,
		),
		PurchasedBandwidthCredits: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "purchased_bandwidth_credits"),
			"Purchased Bandwidth Credits",
			[]string{},
			nil,
		),
		Overage: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "overage"),
			"Overage",
			[]string{},
			nil,
		),
		OverageUnitCost: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "overage_unit_cost"),
			"Overage Unit Cost",
			[]string{},
			nil,
		),
		OverageCost: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, s.Subsystem, "overage_cost"),
			"Overage Cost",
			[]string{},
			nil,
		),
	}
}

// Collect collects metrics
// The underlying types are either int or float32 and so everything must be converted to float64
func (c *BandwidthPeriodCollector) Collect(ch chan<- prometheus.Metric, p govultr.AccountBandwidthPeriod) {
	ch <- prometheus.MustNewConstMetric(
		c.GBIn,
		prometheus.GaugeValue,
		float64(p.GBIn),
		[]string{}...,
	)
	ch <- prometheus.MustNewConstMetric(
		c.GBOut,
		prometheus.GaugeValue,
		float64(p.GBOut),
		[]string{}...,
	)
	ch <- prometheus.MustNewConstMetric(
		c.TotalInstanceHours,
		prometheus.GaugeValue,
		float64(p.TotalInstanceHours),
		[]string{}...,
	)
	ch <- prometheus.MustNewConstMetric(
		c.TotalInstanceCount,
		prometheus.GaugeValue,
		float64(p.TotalInstanceCount),
		[]string{}...,
	)
	ch <- prometheus.MustNewConstMetric(
		c.InstanceBandwidthCredits,
		prometheus.GaugeValue,
		float64(p.InstanceBandwidthCredits),
		[]string{}...,
	)
	ch <- prometheus.MustNewConstMetric(
		c.FreeBandwidthCredits,
		prometheus.GaugeValue,
		float64(p.FreeBandwidthCredits),
		[]string{}...,
	)
	ch <- prometheus.MustNewConstMetric(
		c.PurchasedBandwidthCredits,
		prometheus.GaugeValue,
		float64(p.PurchasedBandwidthCredits),
		[]string{}...,
	)
	ch <- prometheus.MustNewConstMetric(
		c.Overage,
		prometheus.GaugeValue,
		float64(p.Overage),
		[]string{}...,
	)
	ch <- prometheus.MustNewConstMetric(
		c.OverageUnitCost,
		prometheus.GaugeValue,
		float64(p.OverageUnitCost),
		[]string{}...,
	)
	ch <- prometheus.MustNewConstMetric(
		c.OverageCost,
		prometheus.GaugeValue,
		float64(p.OverageCost),
		[]string{}...,
	)
}

// Describe describes metrics
func (c *BandwidthPeriodCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.GBIn
	ch <- c.GBOut
	ch <- c.TotalInstanceHours
	ch <- c.TotalInstanceCount
	ch <- c.InstanceBandwidthCredits
	ch <- c.FreeBandwidthCredits
	ch <- c.PurchasedBandwidthCredits
	ch <- c.Overage
	ch <- c.OverageUnitCost
	ch <- c.OverageCost
}
