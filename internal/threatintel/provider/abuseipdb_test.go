package provider

import "testing"

func TestAbuseIPDB_Fetch(t *testing.T) {
	abuseIPDB := &AbuseIPDB{}
	ips, err := abuseIPDB.Fetch(abuseIPDB.DefaultParams())
	if err != nil {
		t.Errorf("failed to fetch: %v", err)
	}
	if len(ips) == 0 {
		t.Errorf("no ips fetched")
	} else {
		t.Logf("fetched %d ips", len(ips))
	}
}
