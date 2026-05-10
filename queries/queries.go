package queries

import (
	"embed"
)

//go:embed accounts/*.graphql budgets/*.graphql cashflow/*.graphql categories/*.graphql credit/*.graphql goals/*.graphql institutions/*.graphql investments/*.graphql recurring/*.graphql rules/*.graphql subscription/*.graphql tags/*.graphql transactions/*.graphql GetIdentity.graphql
var FS embed.FS

// Get returns the content of a query file.
func Get(path string) string {
	data, err := FS.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
