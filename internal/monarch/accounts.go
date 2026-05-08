package monarch

import (
	"context"
	_ "embed"

	"github.com/monarchmoney-cli/monarch/internal/graphql"
)

//go:embed queries/accounts/list.graphql
var GetAccountsQuery string

type Account struct {
	ID             string  `json:"id"`
	DisplayName    string  `json:"display_name"`
	AccountType    string  `json:"account_type"`
	DisplayBalance float64 `json:"display_balance"`
	UpdatedAt      string  `json:"updated_at"`
}

func (s *Service) ListAccounts(ctx context.Context) ([]Account, error) {
	var resp struct {
		Accounts []struct {
			ID             string `json:"id"`
			DisplayName    string `json:"displayName"`
			AccountType    struct {
				Name string `json:"name"`
			} `json:"accountType"`
			DisplayBalance float64 `json:"displayBalance"`
			UpdatedAt      string  `json:"updatedAt"`
		} `json:"accounts"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetAccounts",
		Query:         GetAccountsQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	accounts := make([]Account, len(resp.Accounts))
	for i, a := range resp.Accounts {
		accounts[i] = Account{
			ID:             a.ID,
			DisplayName:    a.DisplayName,
			AccountType:    a.AccountType.Name,
			DisplayBalance: a.DisplayBalance,
			UpdatedAt:      a.UpdatedAt,
		}
	}

	return accounts, nil
}
