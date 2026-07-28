package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
)

type graphQLClient interface {
	Do(ctx context.Context, reqBody *graphql.Request, result any) error
	DoMutation(ctx context.Context, reqBody *graphql.Request, result any) error
	TokenValue() string
}

type Service struct {
	Client graphQLClient
}

func NewService(client graphQLClient) *Service {
	return &Service{Client: client}
}
