package collector

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultr/govultr/v2"
)

// AccountCollector represents Account
type AccountCollector struct {
	System System
	Client *govultr.Client
	Log    logr.Logger

	Balance        *prometheus.Desc
	Bandwidth      *prometheus.Desc
	PendingCharges *prometheus.Desc
}

// NewAccountCollector create a new AccountCollector
func NewAccountCollector(s System, client *govultr.Client, log logr.Logger) *AccountCollector {
	subsystem := "account"
	return &AccountCollector{
		System: s,
		Client: client,
		Log:    log,

		Balance: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, subsystem, "balance"),
			"Account Balance",
			[]string{
				"name",
				"email",
			},
			nil,
		),
		// Bandwidth: prometheus.NewDesc(
		// 	prometheus.BuildFQName(s.Namespace, subsystem, "bandwidth"),
		// 	"Bandwidth",
		// 	[]string{
		// 		"name",
		// 		"email",
		// 	},
		// 	nil,
		// ),
		PendingCharges: prometheus.NewDesc(
			prometheus.BuildFQName(s.Namespace, subsystem, "pending_charges"),
			"Pending Charges",
			[]string{
				"name",
				"email",
			},
			nil,
		),
	}
}

// Collect implements Prometheus' Collector interface and is used to collect metrics
func (c *AccountCollector) Collect(ch chan<- prometheus.Metric) {
	log := c.Log.WithName("Collect")
	ctx := context.Background()
	account, err := c.Client.Account.Get(ctx)
	if err != nil {
		log.Info("Unable to get account details")
		return
	}

	log.Info("Response",
		"account", account,
	)

	ch <- prometheus.MustNewConstMetric(
		c.Balance,
		prometheus.GaugeValue,
		float64(account.Balance),
		[]string{
			account.Name,
			account.Email,
		}...,
	)
	// ch <- prometheus.MustNewConstMetric(
	// 	c.Bandwidth,
	// 	prometheus.GaugeValue,
	// 	float64(account.Bandwidth),
	// []string{
	// 	account.Name,
	// 	account.Email,
	// }...,
	// )
	ch <- prometheus.MustNewConstMetric(
		c.PendingCharges,
		prometheus.GaugeValue,
		float64(account.PendingCharges),
		[]string{
			account.Name,
			account.Email,
		}...,
	)
}

// Describe implements Prometheus' Collector interface and is used to describe metrics
func (c *AccountCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Balance
	// ch <- c.Bandwidth
	ch <- c.PendingCharges
}
