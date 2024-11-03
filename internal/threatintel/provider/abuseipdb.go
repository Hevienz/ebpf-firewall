package provider

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	// source: https://github.com/borestad/blocklist-abuseipdb
	// s100 means ~100% confidence lists.
	defaultBaseURL = "https://raw.githubusercontent.com/borestad/blocklist-abuseipdb/refs/heads/main"
	sourceMap      = map[string]string{
		"s1001d":   "abuseipdb-s100-1d.ipv4",
		"s1003d":   "abuseipdb-s100-3d.ipv4",
		"s1007d":   "abuseipdb-s100-7d.ipv4",
		"s10014d":  "abuseipdb-s100-14d.ipv4",
		"s10030d":  "abuseipdb-s100-30d.ipv4",
		"s10060d":  "abuseipdb-s100-60d.ipv4",
		"s10090d":  "abuseipdb-s100-90d.ipv4",
		"s100120d": "abuseipdb-s100-120d.ipv4",
		"s991d":    "abuseipdb-s99-hall-of-shame-1d.ipv4",
		"s993d":    "abuseipdb-s99-hall-of-shame-3d.ipv4",
		"s997d":    "abuseipdb-s99-hall-of-shame-7d.ipv4",
		"s9914d":   "abuseipdb-s99-hall-of-shame-14d.ipv4",
		"s9930d":   "abuseipdb-s99-hall-of-shame-30d.ipv4",
		"s9960d":   "abuseipdb-s99-hall-of-shame-60d.ipv4",
		"s9990d":   "abuseipdb-s99-hall-of-shame-90d.ipv4",
		"s99120d":  "abuseipdb-s99-hall-of-shame-120d.ipv4",
	}
)

type AbuseIPDB struct {
}

func (a *AbuseIPDB) Name() string {
	return "abuseipdb"
}

func (a *AbuseIPDB) Description() string {
	return "AbuseIPDB is a platform that provides information about IP addresses that are known to be involved in malicious activities."
}

func (a *AbuseIPDB) Schedule() string {
	return "30 2,18 * * *"
}

func (a *AbuseIPDB) DefaultParams() map[string]string {
	return map[string]string{
		"baseURL": defaultBaseURL,
		// default use s100-30d and s99-hall-of-shame-30d
		"source": "s10030d,s9930d",
	}
}

func (a *AbuseIPDB) Fetch(params map[string]string) ([]string, error) {
	baseURL := params["baseURL"]
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	sources := strings.Split(params["source"], ",")
	if len(sources) == 0 {
		return nil, fmt.Errorf("source is required")
	}
	results := make([]string, 0)
	errs := make([]error, 0)
	for _, source := range sources {
		requestURL := ""
		if url, ok := sourceMap[source]; ok {
			requestURL = url
		} else {
			errs = append(errs, fmt.Errorf("invalid source: %s", source))
			continue
		}
		url := fmt.Sprintf("%s/%s", baseURL, requestURL)
		resp, err := http.Get(url)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to fetch %s: %v", source, err))
			continue
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to read body of %s: %v", source, err))
			continue
		}
		blocklist := strings.Split(string(body), "\n")
		for _, line := range blocklist {
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			results = append(results, line)
		}
	}
	if len(results) > 0 {
		return results, nil
	}
	return nil, errors.Join(errs...)
}
