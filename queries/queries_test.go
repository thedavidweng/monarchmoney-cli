package queries

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestGetPanicsForMissingFile(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Get() did not panic for a missing embedded query")
		}
	}()
	Get("does-not-exist.graphql")
}

func TestNoOrphanQueries(t *testing.T) {
	var embedded []string
	err := fs.WalkDir(FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".graphql") {
			embedded = append(embedded, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk embedded FS: %v", err)
	}

	referenced := make(map[string]bool)
	for _, root := range []string{"../internal", "../cmd"} {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			source, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for _, match := range graphqlFilePattern.FindAllString(string(source), -1) {
				referenced[strings.Trim(match, `"`)] = true
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}

	var orphans []string
	for _, path := range embedded {
		if !referenced[path] {
			orphans = append(orphans, path)
		}
	}
	if len(orphans) > 0 {
		t.Errorf("%d embedded queries are never loaded via queries.Get(): %v. Delete the files or wire them into a service method.", len(orphans), orphans)
	}
}

var graphqlFilePattern = regexp.MustCompile(`"[A-Za-z0-9_/.-]+\.graphql"`)
