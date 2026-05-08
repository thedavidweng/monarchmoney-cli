package monarch

import (
	"github.com/monarchmoney-cli/monarch/internal/graphql"
)

// Service provides access to Monarch Money data.
type Service struct {
	Client *graphql.Client
}

// NewService returns a new Service.
func NewService(client *graphql.Client) *Service {
	return &Service{Client: client}
}
