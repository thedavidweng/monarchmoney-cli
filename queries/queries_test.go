package queries

import "testing"

func TestGetReturnsEmbeddedQuery(t *testing.T) {
	got := Get("GetIdentity.graphql")
	if got == "" {
		t.Fatal("Get() returned empty string for embedded query")
	}
}

func TestGetReturnsEmptyStringForMissingFile(t *testing.T) {
	if got := Get("does-not-exist.graphql"); got != "" {
		t.Fatalf("Get() = %q, want empty string", got)
	}
}
