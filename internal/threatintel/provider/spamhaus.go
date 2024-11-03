package provider

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	dropURLs = []string{
		"https://www.spamhaus.org/drop/drop.txt",
		"https://www.spamhaus.org/drop/edrop.txt",
	}
)

type Spamhaus struct{}

func (s *Spamhaus) Name() string {
	return "spamhaus"
}

func (s *Spamhaus) Description() string {
	return "Spamhaus Project is the authority on IP and domain reputation. This intelligence enables us to shine a light on malicious activity, educate and support those who want to change for the better and hold those who don't to account. We do this together with a like-minded community."
}

func (s *Spamhaus) Schedule() string {
	return "30 2 * * *"
}

func (s *Spamhaus) DefaultParams() map[string]string {
	return map[string]string{}
}

func (s *Spamhaus) Fetch(params map[string]string) ([]string, error) {
	results := make([]string, 0)
	errs := make([]error, 0)
	for _, url := range dropURLs {
		resp, err := http.Get(url)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to fetch %s: %v", url, err))
			continue
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to read body of %s: %v", url, err))
			continue
		}
		lines := strings.Split(string(body), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, ";") {
				continue
			}
			ip := strings.Split(line, ";")[0]
			results = append(results, ip)
		}
	}
	if len(results) > 0 {
		return results, nil
	}
	return nil, errors.Join(errs...)
}
