// Authored and revised by YOC team, 2018
// License placeholder #1

package multihash

import (
	"bytes"
	"math/rand"
	"testing"
)

// parse multihash, and check that invalid multihashes fail
func TestCheckMultihash(t *testing.T) {
	hashbytes := make([]byte, 32)
	c, err := rand.Read(hashbytes)
	if err != nil {
		t.Fatal(err)
	} else if c < 32 {
		t.Fatal("short read")
	}

	expected := ToMultihash(hashbytes)

	l, hl, _ := GetMultihashLength(expected)
	if l != 32 {
		t.Fatalf("expected length %d, got %d", 32, l)
	} else if hl != 2 {
		t.Fatalf("expected header length %d, got %d", 2, hl)
	}
	if _, _, err := GetMultihashLength(expected[1:]); err == nil {
		t.Fatal("expected failure on corrupt header")
	}
	if _, _, err := GetMultihashLength(expected[:len(expected)-2]); err == nil {
		t.Fatal("expected failure on short content")
	}
	dh, _ := FromMultihash(expected)
	if !bytes.Equal(dh, hashbytes) {
		t.Fatalf("expected content hash %x, got %x", hashbytes, dh)
	}
}
