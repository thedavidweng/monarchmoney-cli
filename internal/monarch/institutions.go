package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetInstitutionsQuery = queries.Get("institutions/list.graphql")

type Institution struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (s *Service) ListInstitutions(ctx context.Context) ([]Institution, error) {
	var resp struct {
		Credentials []struct {
			ID                             string `json:"id"`
			UpdateRequired                 bool   `json:"updateRequired"`
			DisconnectedFromDataProviderAt string `json:"disconnectedFromDataProviderAt"`
			DataProvider                   string `json:"dataProvider"`
			Institution                    struct {
				ID                 string `json:"id"`
				PlaidInstitutionID string `json:"plaidInstitutionId"`
				Name               string `json:"name"`
				Status             string `json:"status"`
			} `json:"institution"`
		} `json:"credentials"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetInstitutionSettings",
		Query:         GetInstitutionsQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	// Deduplicate institutions by ID
	seen := make(map[string]bool)
	insts := make([]Institution, 0)
	for _, cred := range resp.Credentials {
		inst := cred.Institution
		if inst.Name != "" && !seen[inst.ID] {
			seen[inst.ID] = true
			insts = append(insts, Institution{
				ID:   inst.ID,
				Name: inst.Name,
				URL:  inst.PlaidInstitutionID,
			})
		}
	}

	return insts, nil
}
