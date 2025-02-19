package collector

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultr/govultr/v3"
)

var (
	_ prometheus.Collector = (*BandwidthCollector)(nil)
)

// BandwidthCollector represents Account Bandwidth
// This comprises repeated (3) occurrences of BandwidthPeriod structs
type BandwidthCollector struct {
	System System
	Client *govultr.Client
	Log    logr.Logger

	Previous  BandwidthPeriodCollector
	Current   BandwidthPeriodCollector
	Projected BandwidthPeriodCollector
}

// NewBandwidthCollector creates a new BandwidthCollector
func NewBandwidthCollector(s System, client *govultr.Client, log logr.Logger) BandwidthCollector {
	// Ensure each bandwidth month's subsystem is unique
	// The namespace and subsystem are used to create uniquely named metrics
	previous := NewBandwidthPeriod(System{
		Namespace: s.Namespace,
		Subsystem: fmt.Sprintf("%s_previous", s.Subsystem),
		Version:   s.Version,
	}, client, log)
	current := NewBandwidthPeriod(System{
		Namespace: s.Namespace,
		Subsystem: fmt.Sprintf("%s_current", s.Subsystem),
		Version:   s.Version,
	}, client, log)
	projected := NewBandwidthPeriod(System{
		Namespace: s.Namespace,
		Subsystem: fmt.Sprintf("%s_projected", s.Subsystem),
		Version:   s.Version,
	}, client, log)

	return BandwidthCollector{
		System: s,
		Client: client,
		Log:    log,

		Previous:  previous,
		Current:   current,
		Projected: projected,
	}
}

// Collect implements Prometheus' Collector interface and is used to collect metrics
func (c BandwidthCollector) Collect(ch chan<- prometheus.Metric) {
	bandwidth, _, err := c.Client.Account.GetBandwidth(context.Background())
	if err != nil {
		c.Log.Error(err, "Account.GetBandwidth")
		return
	}

	// Collect metrics for each BandwidthPeriod
	c.Previous.Collect(ch, bandwidth.PreviousMonth)
	c.Current.Collect(ch, bandwidth.CurrentMonthToDate)
	c.Projected.Collect(ch, bandwidth.CurrentMonthProjected)
}

// Desc	ribe implements Prometheus' Collector interface and is used to describe metrics
func (c BandwidthCollector) Describe(ch chan<- *prometheus.Desc) {
	// Describe metrics for each BandwidthPeriod
	c.Previous.Describe(ch)
	c.Current.Describe(ch)
	c.Projected.Describe(ch)
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
