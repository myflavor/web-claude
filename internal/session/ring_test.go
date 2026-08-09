package session

import "testing"

func TestRingOverflow(t *testing.T) {
	r := NewRing(8)
	r.Write([]byte("hello-world")) // 11 bytes → keep last 8
	if got := string(r.Bytes()); got != "lo-world" {
		t.Fatalf("got %q want lo-world", got)
	}
	r2 := NewRing(8)
	r2.Write([]byte("12345"))
	r2.Write([]byte("67890"))
	if got := string(r2.Bytes()); got != "34567890" {
		t.Fatalf("got %q want 34567890", got)
	}
}
