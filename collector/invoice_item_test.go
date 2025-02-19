package collector

import (
	"testing"
)

func TestCanonicalize(t *testing.T) {
	// Load Balancer is a valid Product name
	// But I don't know of any others
	for got, want := range map[string]string{
		"Load Balancer":       "load_balancer",
		"Object Storage":      "object_storage",
		"Vultr Cloud Compute": "vultr_cloud_compute",
	} {
		t.Run(got, func(t *testing.T) {
			if got := canonicalize(got); got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}
