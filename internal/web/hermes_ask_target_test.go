package web

import (
	"testing"

	"mochil/internal/appconfig"
)

// A one-shot question must always have somewhere to go: a dedicated helper when one is configured, otherwise the
// gateway itself so it follows the profile's default model. Getting this wrong is invisible — the tool simply answers
// "Hermes did not answer" — so the resolution is pinned here rather than left to configuration.
func TestHelperTargetPrefersTheHelperThenTheGateway(t *testing.T) {
	gateway := appconfig.Hermes{URL: "http://192.168.50.13:8642/p/companion/", Model: "companion", Key: "gateway-key"}
	cases := []struct {
		name   string
		hermes appconfig.Hermes
		ok     bool
		base   string
		model  string
		key    string
	}{
		{
			name:   "dedicated helper wins and needs no key",
			hermes: appconfig.Hermes{URL: gateway.URL, Model: "companion", Key: "k", HelperURL: "http://box:11434", HelperModel: "small"},
			ok:     true,
			base:   "http://box:11434/v1",
			model:  "small",
			key:    "",
		},
		{
			name:   "helper address already carrying /v1 is not doubled",
			hermes: appconfig.Hermes{HelperURL: "http://box:1234/v1/", HelperModel: "small"},
			ok:     true,
			base:   "http://box:1234/v1",
			model:  "small",
			key:    "",
		},
		{
			name:   "without a helper the gateway answers, with its key",
			hermes: gateway,
			ok:     true,
			base:   "http://192.168.50.13:8642/p/companion/v1",
			model:  "companion",
			key:    "gateway-key",
		},
		{
			name:   "no helper and no gateway model is nothing",
			hermes: appconfig.Hermes{URL: gateway.URL},
			ok:     false,
		},
		{
			name:   "no address at all is nothing",
			hermes: appconfig.Hermes{Model: "companion"},
			ok:     false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base, model, key, ok := helperTarget(tc.hermes)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if !ok {
				return
			}
			if base != tc.base || model != tc.model || key != tc.key {
				t.Fatalf("target = (%q, %q, %q), want (%q, %q, %q)", base, model, key, tc.base, tc.model, tc.key)
			}
		})
	}
}
