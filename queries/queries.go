package queries

import (
	"embed"
	"fmt"
)

//go:embed accounts/*.graphql budgets/*.graphql cashflow/*.graphql categories/*.graphql credit/*.graphql goals/*.graphql household/*.graphql institutions/*.graphql investments/*.graphql merchants/*.graphql recurring/*.graphql receipts/*.graphql reports/*.graphql rules/*.graphql subscription/*.graphql tags/*.graphql transactions/*.graphql GetIdentity.graphql
var FS embed.FS

func Get(path string) string {
	data, err := FS.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("queries: embedded query %q missing: %v", path, err))
	}
	return string(data)
}
