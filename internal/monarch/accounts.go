package monarch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/monarchmoney-cli/monarch/internal/errors"
	"github.com/monarchmoney-cli/monarch/internal/graphql"
	"github.com/monarchmoney-cli/monarch/queries"
)

var GetAccountsQuery = queries.Get("accounts/list.graphql")
var GetAccountQuery = queries.Get("accounts/show.graphql")
var GetAccountHoldingsQuery = queries.Get("accounts/holdings.graphql")
var GetAccountHistoryQuery = queries.Get("accounts/history.graphql")
var GetAccountTypesQuery = queries.Get("accounts/types.graphql")
var RefreshAccountsMutation = queries.Get("accounts/refresh.graphql")
var GetAccountsRefreshStatusQuery = queries.Get("accounts/refresh_status.graphql")
var GetAccountRecentBalancesQuery = queries.Get("accounts/recent_balances.graphql")
var GetSnapshotsByAccountTypeQuery = queries.Get("accounts/snapshots_by_type.graphql")
var GetAggregateSnapshotsQuery = queries.Get("accounts/aggregate_snapshots.graphql")
var UpdateAccountMutation = queries.Get("accounts/update.graphql")
var DeleteAccountMutation = queries.Get("accounts/delete.graphql")
var CreateManualAccountMutation = queries.Get("accounts/create_manual.graphql")

type Account struct {
	ID             string  `json:"id"`
	DisplayName    string  `json:"display_name"`
	AccountType    string  `json:"account_type"`
	DisplayBalance float64 `json:"display_balance"`
	UpdatedAt      string  `json:"updated_at"`
}

type Holding struct {
	ID       string  `json:"id"`
	Security string  `json:"security"`
	Symbol   string  `json:"symbol"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
	Value    float64 `json:"value"`
}

type HistoryRecord struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

func (s *Service) GetAccountHoldings(ctx context.Context, accountID string) ([]Holding, error) {
	var resp struct {
		Account struct {
			Holdings []struct {
				ID       string `json:"id"`
				Security struct {
					Name   string `json:"name"`
					Symbol string `json:"symbol"`
				} `json:"security"`
				Quantity float64 `json:"quantity"`
				Price    float64 `json:"price"`
				Value    float64 `json:"value"`
			} `json:"holdings"`
		} `json:"account"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetAccountHoldings",
		Query:         GetAccountHoldingsQuery,
		Variables:     map[string]interface{}{"accountId": accountID},
	}, &resp)

	if err != nil {
		return nil, err
	}

	holdings := make([]Holding, len(resp.Account.Holdings))
	for i, h := range resp.Account.Holdings {
		holdings[i] = Holding{
			ID:       h.ID,
			Security: h.Security.Name,
			Symbol:   h.Security.Symbol,
			Quantity: h.Quantity,
			Price:    h.Price,
			Value:    h.Value,
		}
	}

	return holdings, nil
}

func (s *Service) GetAccountHistory(ctx context.Context, accountID string, startDate, endDate string) ([]HistoryRecord, error) {
	var resp struct {
		Account struct {
			BalanceHistory []struct {
				Date   string  `json:"date"`
				Amount float64 `json:"amount"`
			} `json:"balanceHistory"`
		} `json:"account"`
	}

	variables := map[string]interface{}{"accountId": accountID}
	if startDate != "" {
		variables["startDate"] = startDate
	}
	if endDate != "" {
		variables["endDate"] = endDate
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetAccountHistory",
		Query:         GetAccountHistoryQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	history := make([]HistoryRecord, len(resp.Account.BalanceHistory))
	for i, r := range resp.Account.BalanceHistory {
		history[i] = HistoryRecord{
			Date:   r.Date,
			Amount: r.Amount,
		}
	}

	return history, nil
}

func (s *Service) GetAccount(ctx context.Context, id string) (*Account, error) {
	var resp struct {
		Account struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			AccountType struct {
				Name string `json:"name"`
			} `json:"accountType"`
			DisplayBalance float64 `json:"displayBalance"`
			UpdatedAt      string  `json:"updatedAt"`
		} `json:"account"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetAccount",
		Query:         GetAccountQuery,
		Variables:     map[string]interface{}{"id": id},
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &Account{
		ID:             resp.Account.ID,
		DisplayName:    resp.Account.DisplayName,
		AccountType:    resp.Account.AccountType.Name,
		DisplayBalance: resp.Account.DisplayBalance,
		UpdatedAt:      resp.Account.UpdatedAt,
	}, nil
}

func (s *Service) GetAccountRecentBalances(ctx context.Context, startDate string) (interface{}, error) {
	var resp struct {
		Accounts []struct {
			ID             string      `json:"id"`
			RecentBalances interface{} `json:"recentBalances"`
		} `json:"accounts"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetAccountRecentBalances",
		Query:         GetAccountRecentBalancesQuery,
		Variables:     map[string]interface{}{"startDate": startDate},
	}, &resp)

	if err != nil {
		return nil, err
	}

	return resp.Accounts, nil
}

func (s *Service) GetSnapshotsByAccountType(ctx context.Context, startDate, timeframe string) (interface{}, error) {
	var resp struct {
		SnapshotsByAccountType []struct {
			AccountType string  `json:"accountType"`
			Month       string  `json:"month"`
			Balance     float64 `json:"balance"`
		} `json:"snapshotsByAccountType"`
		AccountTypes []struct {
			Name  string `json:"name"`
			Group string `json:"group"`
		} `json:"accountTypes"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetSnapshotsByAccountType",
		Query:         GetSnapshotsByAccountTypeQuery,
		Variables: map[string]interface{}{
			"startDate": startDate,
			"timeframe": timeframe,
		},
	}, &resp)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *Service) GetAggregateSnapshots(ctx context.Context, startDate, endDate, accountType string) (interface{}, error) {
	var resp struct {
		AggregateSnapshots []struct {
			Date    string  `json:"date"`
			Balance float64 `json:"balance"`
		} `json:"aggregateSnapshots"`
	}

	filters := map[string]interface{}{
		"startDate": startDate,
	}
	if endDate != "" {
		filters["endDate"] = endDate
	}
	if accountType != "" {
		filters["accountType"] = accountType
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetAggregateSnapshots",
		Query:         GetAggregateSnapshotsQuery,
		Variables:     map[string]interface{}{"filters": filters},
	}, &resp)

	if err != nil {
		return nil, err
	}

	return resp.AggregateSnapshots, nil
}

func (s *Service) GetAccountTypes(ctx context.Context) ([]string, error) {
	var resp struct {
		AccountTypeOptions []struct {
			Name string `json:"name"`
		} `json:"accountTypeOptions"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetAccountTypeOptions",
		Query:         GetAccountTypesQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	types := make([]string, len(resp.AccountTypeOptions))
	for i, t := range resp.AccountTypeOptions {
		types[i] = t.Name
	}

	return types, nil
}

func (s *Service) GetAccountsRefreshStatus(ctx context.Context) (map[string]interface{}, error) {
	var resp struct {
		AccountRefreshProgress struct {
			IsComplete bool   `json:"isComplete"`
			Status     string `json:"status"`
			StartTime  string `json:"startTime"`
			EndTime    string `json:"endTime"`
		} `json:"accountRefreshProgress"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetAccountsRefreshStatus",
		Query:         GetAccountsRefreshStatusQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"is_complete": resp.AccountRefreshProgress.IsComplete,
		"status":      resp.AccountRefreshProgress.Status,
		"start_time":  resp.AccountRefreshProgress.StartTime,
		"end_time":    resp.AccountRefreshProgress.EndTime,
	}, nil
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

func (s *Service) CreateManualAccount(ctx context.Context, name, accType string, balance float64) (*Account, error) {
	var resp struct {
		CreateManualAccount struct {
			Account struct {
				ID             string  `json:"id"`
				DisplayName    string  `json:"displayName"`
				DisplayBalance float64 `json:"displayBalance"`
			} `json:"account"`
		} `json:"createManualAccount"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "CreateManualAccount",
		Query:         CreateManualAccountMutation,
		Variables: map[string]interface{}{
			"name":    name,
			"type":    accType,
			"balance": balance,
		},
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &Account{
		ID:             resp.CreateManualAccount.Account.ID,
		DisplayName:    resp.CreateManualAccount.Account.DisplayName,
		DisplayBalance: resp.CreateManualAccount.Account.DisplayBalance,
	}, nil
}

func (s *Service) RefreshAccounts(ctx context.Context, accountIDs []string) error {
	var resp struct {
		RequestAccountsRefresh struct {
			OK bool `json:"ok"`
		} `json:"requestAccountsRefresh"`
	}

	variables := make(map[string]interface{})
	if len(accountIDs) > 0 {
		variables["accountIds"] = accountIDs
	}

	return s.Client.Do(ctx, &graphql.Request{
		OperationName: "RefreshAccounts",
		Query:         RefreshAccountsMutation,
		Variables:     variables,
	}, &resp)
}

func (s *Service) UpdateAccount(ctx context.Context, id string, name *string, balance *float64) (*Account, error) {
	var resp struct {
		UpdateAccount struct {
			Account struct {
				ID             string  `json:"id"`
				DisplayName    string  `json:"displayName"`
				DisplayBalance float64 `json:"displayBalance"`
			} `json:"account"`
		} `json:"updateAccount"`
	}

	variables := map[string]interface{}{"id": id}
	if name != nil {
		variables["displayName"] = *name
	}
	if balance != nil {
		variables["balance"] = *balance
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "UpdateAccount",
		Query:         UpdateAccountMutation,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &Account{
		ID:             resp.UpdateAccount.Account.ID,
		DisplayName:    resp.UpdateAccount.Account.DisplayName,
		DisplayBalance: resp.UpdateAccount.Account.DisplayBalance,
	}, nil
}

func (s *Service) DeleteAccount(ctx context.Context, id string) error {
	var resp struct {
		DeleteAccount struct {
			OK bool `json:"ok"`
		} `json:"deleteAccount"`
	}

	return s.Client.Do(ctx, &graphql.Request{
		OperationName: "DeleteAccount",
		Query:         DeleteAccountMutation,
		Variables:     map[string]interface{}{"id": id},
	}, &resp)
}

func (s *Service) UploadAccountBalanceHistory(ctx context.Context, id string, r io.Reader) error {
	// Monarch uses a REST endpoint for file uploads
	url := "https://api.monarch.com/account-balance-history/upload/"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "history.csv")
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, r); err != nil {
		return err
	}
	writer.WriteField("account_id", id)
	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Client-Platform", "web")
	if s.Client.Token != "" {
		req.Header.Set("Authorization", "Token "+s.Client.Token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New(errors.APIError, fmt.Sprintf("upload failed with status %d", resp.StatusCode), errors.CatAPI, false, nil)
	}

	return nil
}
