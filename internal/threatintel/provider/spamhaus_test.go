package provider

import (
	"encoding/json"
	"testing"
)

func TestSpamhaus_Fetch(t *testing.T) {
	spamhaus := &Spamhaus{}
	ips, err := spamhaus.Fetch(spamhaus.DefaultParams())
	data, _ := json.MarshalIndent(ips, "", "  ")
	t.Logf("ips: %s", string(data))
	if err != nil {
		t.Errorf("failed to fetch: %v", err)
	}
	if len(ips) < 50 {
		t.Errorf("no ips fetched")
	} else {
		t.Logf("fetched %d ips", len(ips))
	}
}
