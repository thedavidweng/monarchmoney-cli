package queries

import "testing"

func TestGetPanicsForMissingFile(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Get() did not panic for a missing embedded query")
		}
	}()
	Get("does-not-exist.graphql")
}
