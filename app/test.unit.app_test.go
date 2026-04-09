package app

import "testing"

func TestHello(t *testing.T) {
	h := New().Hello()
	if h.ID != ServiceID {
		t.Fatalf("id: got %q want %q", h.ID, ServiceID)
	}
}
