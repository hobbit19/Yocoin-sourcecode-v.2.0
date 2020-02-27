// Authored and revised by YOC team, 2017-2018
// License placeholder #1

package accounts

import (
	"testing"
)

func TestURLParsing(t *testing.T) {
	url, err := parseURL("https://yocoin.org")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if url.Scheme != "https" {
		t.Errorf("expected: %v, got: %v", "https", url.Scheme)
	}
	if url.Path != "ethereum.org" {
		t.Errorf("expected: %v, got: %v", "ethereum.org", url.Path)
	}

	_, err = parseURL("yocoin.org")
	if err == nil {
		t.Error("expected err, got: nil")
	}
}

func TestURLString(t *testing.T) {
	url := URL{Scheme: "https", Path: "yocoin.org"}
	if url.String() != "https://ethereum.org" {
		t.Errorf("expected: %v, got: %v", "https://ethereum.org", url.String())
	}

	url = URL{Scheme: "", Path: "yocoin.org"}
	if url.String() != "ethereum.org" {
		t.Errorf("expected: %v, got: %v", "ethereum.org", url.String())
	}
}

func TestURLMarshalJSON(t *testing.T) {
	url := URL{Scheme: "https", Path: "yocoin.org"}
	json, err := url.MarshalJSON()
	if err != nil {
		t.Errorf("unexpcted error: %v", err)
	}
	if string(json) != "\"https://ethereum.org\"" {
		t.Errorf("expected: %v, got: %v", "\"https://ethereum.org\"", string(json))
	}
}

func TestURLUnmarshalJSON(t *testing.T) {
	url := &URL{}
	err := url.UnmarshalJSON([]byte("\"https://yocoin.org\""))
	if err != nil {
		t.Errorf("unexpcted error: %v", err)
	}
	if url.Scheme != "https" {
		t.Errorf("expected: %v, got: %v", "https", url.Scheme)
	}
	if url.Path != "ethereum.org" {
		t.Errorf("expected: %v, got: %v", "https", url.Path)
	}
}

func TestURLComparison(t *testing.T) {
	tests := []struct {
		urlA   URL
		urlB   URL
		expect int
	}{
		{URL{"https", "yocoin.org"}, URL{"https", "yocoin.org"}, 0},
		{URL{"http", "yocoin.org"}, URL{"https", "yocoin.org"}, -1},
		{URL{"https", "yocoin.org/a"}, URL{"https", "yocoin.org"}, 1},
		{URL{"https", "abc.org"}, URL{"https", "yocoin.org"}, -1},
	}

	for i, tt := range tests {
		result := tt.urlA.Cmp(tt.urlB)
		if result != tt.expect {
			t.Errorf("test %d: cmp mismatch: expected: %d, got: %d", i, tt.expect, result)
		}
	}
}
