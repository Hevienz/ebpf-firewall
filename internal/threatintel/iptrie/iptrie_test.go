package iptrie

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"testing"
)

func TestIPTrie_Insert(t *testing.T) {
	tests := []struct {
		name    string
		inputs  []string
		wantErr bool
	}{
		{
			name:    "Valid IPv4 addresses",
			inputs:  []string{"192.168.1.1", "10.0.0.0/8", "172.16.0.0/12"},
			wantErr: false,
		},
		{
			name:    "Valid IPv6 addresses",
			inputs:  []string{"2001:db8::/32", "::1", "fe80::/10"},
			wantErr: false,
		},
		{
			name:    "Invalid IP addresses",
			inputs:  []string{"invalid", "256.256.256.256", "2001:xyz::/32"},
			wantErr: true,
		},
		{
			name:    "Mixed IPv4 and IPv6",
			inputs:  []string{"192.168.1.1", "2001:db8::/32", "10.0.0.0/8", "fe80::/10"},
			wantErr: false,
		},
		{
			name:    "Special IPv4 addresses",
			inputs:  []string{"0.0.0.0", "255.255.255.255", "127.0.0.1"},
			wantErr: false,
		},
		{
			name:    "Special IPv6 addresses",
			inputs:  []string{"::", "::1", "fe80::1", "ff02::1"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trie := NewIPTrie()
			for _, input := range tt.inputs {
				err := trie.Insert(input)
				if (err != nil) != tt.wantErr {
					t.Errorf("Insert(%s) error = %v, wantErr %v", input, err, tt.wantErr)
				}
			}
		})
	}
}

func TestIPTrie_InsertDuplicate(t *testing.T) {
	trie := NewIPTrie()
	err := trie.Insert("192.168.1.1")
	if err != nil {
		t.Fatalf("Insert(%s) error = %v", "192.168.1.1", err)
	}
	err = trie.Insert("192.168.1.1")
	if err == nil {
		t.Fatalf("Insert(%s) error = %v, wantErr %v", "192.168.1.1", err, true)
	}
	err = trie.Insert("192.168.1.1/32")
	if err == nil {
		t.Fatalf("Insert(%s) error = %v, wantErr %v", "192.168.1.1/32", err, true)
	}
	err = trie.Insert("192.168.1.2")
	if err != nil {
		t.Fatalf("Insert(%s) error = %v", "192.168.1.2", err)
	}
}

func TestIPTrie_Contains(t *testing.T) {
	tests := []struct {
		name     string
		inserts  []string
		queries  []string
		expected []bool
	}{
		{
			name:     "IPv4 exact match",
			inserts:  []string{"192.168.1.1", "10.0.0.1"},
			queries:  []string{"192.168.1.1", "192.168.1.2", "10.0.0.1"},
			expected: []bool{true, false, true},
		},
		{
			name:     "IPv4 CIDR match",
			inserts:  []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
			queries:  []string{"10.1.1.1", "172.16.1.1", "192.168.1.1", "8.8.8.8"},
			expected: []bool{true, true, true, false},
		},
		{
			name:     "IPv6 exact match",
			inserts:  []string{"2001:db8::1", "fe80::1"},
			queries:  []string{"2001:db8::1", "2001:db8::2", "fe80::1"},
			expected: []bool{true, false, true},
		},
		{
			name:     "IPv6 CIDR match",
			inserts:  []string{"2001:db8::/32", "fe80::/10"},
			queries:  []string{"2001:db8::1", "2001:db8:1::1", "fe80::1", "2001:db9::1"},
			expected: []bool{true, true, true, false},
		},
		{
			name:     "Mixed IPv4 and IPv6",
			inserts:  []string{"192.168.0.0/16", "2001:db8::/32"},
			queries:  []string{"192.168.1.1", "2001:db8::1", "192.168.2.1", "2001:db9::1"},
			expected: []bool{true, true, true, false},
		},
		{
			name:     "Special networks",
			inserts:  []string{"127.0.0.0/8", "::1/128", "169.254.0.0/16", "fe80::/10"},
			queries:  []string{"127.0.0.1", "::1", "169.254.1.1", "fe80::1"},
			expected: []bool{true, true, true, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trie := NewIPTrie()
			for _, insert := range tt.inserts {
				err := trie.Insert(insert)
				if err != nil {
					t.Fatalf("Insert(%s) error = %v", insert, err)
				}
			}

			for i, query := range tt.queries {
				got := trie.Contains(query)
				if got != tt.expected[i] {
					t.Errorf("Contains(%s) = %v, want %v", query, got, tt.expected[i])
				}
			}
		})
	}
}

func TestIPTrie_Remove(t *testing.T) {
	tests := []struct {
		name           string
		inserts        []string
		removes        []string
		queriesAfter   []string
		expectedAfter  []bool
		expectedErrors []bool
	}{
		{
			name:           "Remove IPv4 addresses",
			inserts:        []string{"192.168.1.1", "10.0.0.0/8", "172.16.0.0/12"},
			removes:        []string{"192.168.1.1", "10.0.0.0/8"},
			queriesAfter:   []string{"192.168.1.1", "10.1.1.1", "172.16.1.1"},
			expectedAfter:  []bool{false, false, true},
			expectedErrors: []bool{false, false},
		},
		{
			name:           "Remove IPv6 addresses",
			inserts:        []string{"2001:db8::1", "2001:db8::/32", "fe80::/10"},
			removes:        []string{"2001:db8::1", "2001:db8::/32"},
			queriesAfter:   []string{"2001:db8::1", "2001:db8:1::1", "fe80::1"},
			expectedAfter:  []bool{false, false, true},
			expectedErrors: []bool{false, false},
		},
		{
			name:           "Remove invalid addresses",
			inserts:        []string{"192.168.1.1", "2001:db8::1"},
			removes:        []string{"invalid", "256.256.256.256"},
			queriesAfter:   []string{"192.168.1.1", "2001:db8::1"},
			expectedAfter:  []bool{true, true},
			expectedErrors: []bool{true, true},
		},
		{
			name:           "Remove non-existent addresses",
			inserts:        []string{"192.168.1.1", "2001:db8::1"},
			removes:        []string{"192.168.1.2", "2001:db8::2"},
			queriesAfter:   []string{"192.168.1.1", "2001:db8::1"},
			expectedAfter:  []bool{true, true},
			expectedErrors: []bool{false, false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trie := NewIPTrie()

			for _, insert := range tt.inserts {
				err := trie.Insert(insert)
				if err != nil {
					t.Fatalf("Insert(%s) error = %v", insert, err)
				}
			}

			for i, remove := range tt.removes {
				err := trie.Remove(remove)
				if (err != nil) != tt.expectedErrors[i] {
					t.Errorf("Remove(%s) error = %v, wantErr %v", remove, err, tt.expectedErrors[i])
				}
			}

			for i, query := range tt.queriesAfter {
				got := trie.Contains(query)
				if got != tt.expectedAfter[i] {
					t.Errorf("After removal - Contains(%s) = %v, want %v", query, got, tt.expectedAfter[i])
				}
			}
		})
	}
}

func TestIPTrie_Size(t *testing.T) {
	tests := []struct {
		name         string
		inserts      []string
		removes      []string
		expectedSize int
	}{
		{
			name:         "Mixed IPv4 and IPv6",
			inserts:      []string{"192.168.1.1", "10.0.0.0/8", "2001:db8::1", "fe80::/10"},
			removes:      []string{"192.168.1.1"},
			expectedSize: 3,
		},
		{
			name:         "Empty tree",
			inserts:      []string{},
			removes:      []string{},
			expectedSize: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trie := NewIPTrie()

			for _, insert := range tt.inserts {
				err := trie.Insert(insert)
				if err != nil {
					t.Fatalf("Insert(%s) error = %v", insert, err)
				}
			}

			for _, remove := range tt.removes {
				err := trie.Remove(remove)
				if err != nil {
					t.Fatalf("Remove(%s) error = %v", remove, err)
				}
			}

			if got := trie.Size(); int(got) != tt.expectedSize {
				t.Errorf("Size() = %v, want %v", got, tt.expectedSize)
			}
		})
	}
}

func TestParseIPAddrToIPNet(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
	}{
		{
			name:    "Valid IPv4 address",
			input:   "192.168.1.1",
			wantNil: false,
		},
		{
			name:    "Valid IPv4 CIDR",
			input:   "192.168.0.0/16",
			wantNil: false,
		},
		{
			name:    "Valid IPv6 address",
			input:   "2001:db8::1",
			wantNil: false,
		},
		{
			name:    "Valid IPv6 CIDR",
			input:   "2001:db8::/32",
			wantNil: false,
		},
		{
			name:    "Invalid IP address",
			input:   "invalid",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIPAddrToIPNet(tt.input)
			if (result == nil) != tt.wantNil {
				t.Errorf("parseIPAddrToIPNet(%s) = %v, want nil: %v", tt.input, result, tt.wantNil)
			}
		})
	}
}

func TestIPTrie_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		inserts  []string
		queries  []string
		expected []bool
	}{
		{
			name:     "Empty trie",
			inserts:  []string{},
			queries:  []string{"192.168.1.1", "2001:db8::1"},
			expected: []bool{false, false},
		},
		{
			name:     "Single bit difference",
			inserts:  []string{"192.168.1.0/31"},
			queries:  []string{"192.168.1.0", "192.168.1.1", "192.168.1.2"},
			expected: []bool{true, true, false},
		},
		{
			name:     "Overlapping networks",
			inserts:  []string{"10.0.0.0/8", "10.0.0.0/16", "10.0.0.0/24"},
			queries:  []string{"10.0.0.1", "10.0.1.1", "10.1.0.1"},
			expected: []bool{true, true, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trie := NewIPTrie()
			for _, insert := range tt.inserts {
				_ = trie.Insert(insert)
			}

			for i, query := range tt.queries {
				got := trie.Contains(query)
				if got != tt.expected[i] {
					t.Errorf("Contains(%s) = %v, want %v", query, got, tt.expected[i])
				}
			}
		})
	}
}
func TestIPTrie_Concurrency(t *testing.T) {
	trie := NewIPTrie()
	const (
		workers      = 10
		opsPerWorker = 100
	)

	insertData := make([][]string, workers)
	for i := 0; i < workers; i++ {
		insertData[i] = make([]string, opsPerWorker)
		for j := 0; j < opsPerWorker; j++ {
			insertData[i][j] = fmt.Sprintf("10.0.%d.%d", i, j)
		}
	}

	var wg sync.WaitGroup
	wg.Add(workers)
	insertErrors := make(chan error, workers*opsPerWorker)

	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerWorker; j++ {
				if err := trie.Insert(insertData[id][j]); err != nil {
					insertErrors <- fmt.Errorf("worker %d insert error at %d: %v", id, j, err)
				}
			}
		}(i)
	}
	wg.Wait()
	close(insertErrors)

	for err := range insertErrors {
		t.Errorf("Insert error: %v", err)
	}

	expectedSize := workers * opsPerWorker
	if size := trie.Size(); int(size) != expectedSize {
		t.Errorf("Expected size %d after insertions, got %d", expectedSize, size)
	}

	wg.Add(workers)
	queryErrors := make(chan error, workers*opsPerWorker)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerWorker; j++ {
				ip := insertData[id][j]
				if !trie.Contains(ip) {
					queryErrors <- fmt.Errorf("worker %d: IP %s should exist but not found", id, ip)
				}
			}
		}(i)
	}
	wg.Wait()
	close(queryErrors)

	for err := range queryErrors {
		t.Errorf("Query error: %v", err)
	}

	wg.Add(workers)
	removeErrors := make(chan error, workers*opsPerWorker)

	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < opsPerWorker; j++ {
				if err := trie.Remove(insertData[id][j]); err != nil {
					removeErrors <- fmt.Errorf("worker %d remove error at %d: %v", id, j, err)
				}
			}
		}(i)
	}
	wg.Wait()
	close(removeErrors)

	for err := range removeErrors {
		t.Errorf("Remove error: %v", err)
	}

	if size := trie.Size(); size != 0 {
		t.Errorf("Expected empty trie after all operations, got size %d", size)
	}
}

func generateRandomIPv4(r *rand.Rand) string {
	return fmt.Sprintf("%d.%d.%d.%d",
		r.Intn(256), r.Intn(256),
		r.Intn(256), r.Intn(256))
}

func generateRandomIPv6(r *rand.Rand) string {
	bytes := make([]byte, 16)
	r.Read(bytes)
	ip := net.IP(bytes)
	return ip.String()
}

func generateRandomCIDR(isIPv6 bool, r *rand.Rand) string {
	if isIPv6 {
		return fmt.Sprintf("%s/%d", generateRandomIPv6(r), r.Intn(127)+1)
	}
	return fmt.Sprintf("%s/%d", generateRandomIPv4(r), r.Intn(31)+1)
}

func generateTestData(r *rand.Rand, ipCount, cidrCount int) ([]string, []string) {
	ips := make([]string, ipCount)
	cidrs := make([]string, cidrCount)

	for i := 0; i < ipCount; i++ {
		if r.Float32() < 0.5 {
			ips[i] = generateRandomIPv4(r)
		} else {
			ips[i] = generateRandomIPv6(r)
		}
	}

	for i := 0; i < cidrCount; i++ {
		if r.Float32() < 0.5 {
			cidrs[i] = generateRandomCIDR(false, r)
		} else {
			cidrs[i] = generateRandomCIDR(true, r)
		}
	}

	return ips, cidrs
}

func BenchmarkIpTrieInsertIPV4(b *testing.B) {
	r := rand.New(rand.NewSource(1234))

	ips, _ := generateTestData(r, b.N, 0)

	b.ResetTimer()
	b.ReportAllocs()

	trie := NewIPTrie()
	for i := 0; i < len(ips); i++ {
		trie.Insert(ips[i])
	}
}

func BenchmarkIpTrieInsertIPV6(b *testing.B) {
	r := rand.New(rand.NewSource(1234))

	_, cidrs := generateTestData(r, 0, b.N)

	b.ResetTimer()
	b.ReportAllocs()

	trie := NewIPTrie()
	for i := 0; i < len(cidrs); i++ {
		trie.Insert(cidrs[i])
	}
}

func BenchmarkIpTrieQuery(b *testing.B) {
	r := rand.New(rand.NewSource(1234))

	const baseDataSize = 1000000
	trie := NewIPTrie()
	ips, cidrs := generateTestData(r, baseDataSize/2, baseDataSize/2)

	for i := 0; i < baseDataSize; i++ {
		if i < len(ips) {
			trie.Insert(ips[i])
		} else {
			trie.Insert(cidrs[i-len(ips)])
		}
	}

	queryIPs, _ := generateTestData(r, b.N, 0)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		trie.Contains(queryIPs[i])
	}
}
