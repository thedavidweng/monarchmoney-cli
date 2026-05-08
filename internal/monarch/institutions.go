package monarch

import (
	"context"
	_ "embed"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
)

//go:embed queries/institutions/list.graphql
var GetInstitutionsQuery string

type Institution struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (s *Service) ListInstitutions(ctx context.Context) ([]Institution, error) {
	var resp struct {
		Institutions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"institutions"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetInstitutions",
		Query:         GetInstitutionsQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	insts := make([]Institution, len(resp.Institutions))
	for i, inst := range resp.Institutions {
		insts[i] = Institution{
			ID:   inst.ID,
			Name: inst.Name,
			URL:  inst.URL,
		}
	}

	return insts, nil
}
