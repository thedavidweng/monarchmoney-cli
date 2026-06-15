// Package monarch implements the Monarch Money service layer, translating business operations into GraphQL calls.
package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
)

type graphQLClient interface {
	Do(ctx context.Context, reqBody *graphql.Request, result any) error
	TokenValue() string
}

// Service provides access to Monarch Money data.
type Service struct {
	Client graphQLClient
}

// NewService returns a new Service.
func NewService(client graphQLClient) *Service {
	return &Service{Client: client}
}
