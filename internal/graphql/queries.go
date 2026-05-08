package graphql

import (
	_ "embed"
)

//go:embed queries/GetIdentity.graphql
var GetIdentityQuery string
