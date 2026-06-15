package monarch

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetAccountsQuery = queries.Get("accounts/list.graphql")
var GetAccountQuery = queries.Get("accounts/show.graphql")
var GetAccountHoldingsQuery = queries.Get("accounts/holdings.graphql")
var GetAccountHistoryQuery = queries.Get("accounts/history.graphql")
var GetAccountTypesQuery = queries.Get("accounts/types.graphql")
var GetAccountBalancesAtQuery = queries.Get("accounts/balance_at.graphql")
var RefreshAccountsMutation = queries.Get("accounts/refresh.graphql")
var GetAccountsRefreshStatusQuery = queries.Get("accounts/refresh_status.graphql")
var GetAccountRecentBalancesQuery = queries.Get("accounts/recent_balances.graphql")
var GetSnapshotsByAccountTypeQuery = queries.Get("accounts/snapshots_by_type.graphql")
var GetAggregateSnapshotsQuery = queries.Get("accounts/aggregate_snapshots.graphql")
var UpdateAccountMutation = queries.Get("accounts/update.graphql")
var DeleteAccountMutation = queries.Get("accounts/delete.graphql")
var CreateManualAccountMutation = queries.Get("accounts/create_manual.graphql")
var newBalanceHistoryRequest = http.NewRequestWithContext
var createBalanceHistoryFormFile = func(w *multipart.Writer, field, filename string) (io.Writer, error) {
	return w.CreateFormFile(field, filename)
}

type Account struct {
	ID                              string  `json:"id"`
	DisplayName                     string  `json:"display_name"`
	AccountType                     string  `json:"account_type"`
	AccountSubtype                  string  `json:"account_subtype"`
	DisplayBalance                  float64 `json:"display_balance"`
	CurrentBalance                  float64 `json:"current_balance"`
	Limit                           float64 `json:"limit"`
	DataProviderCreditLimit         float64 `json:"data_provider_credit_limit"`
	UpdatedAt                       string  `json:"updated_at"`
	DisplayLastUpdatedAt            string  `json:"display_last_updated_at"`
	DeactivatedAt                   string  `json:"deactivated_at"`
	IsHidden                        bool    `json:"is_hidden"`
	IsAsset                         bool    `json:"is_asset"`
	Mask                            string  `json:"mask"`
	CreatedAt                       string  `json:"created_at"`
	IncludeInNetWorth               bool    `json:"include_in_net_worth"`
	HideFromList                    bool    `json:"hide_from_list"`
	HideTransactionsFromReports     bool    `json:"hide_transactions_from_reports"`
	IncludeBalanceInNetWorth        bool    `json:"include_balance_in_net_worth"`
	IncludeInGoalBalance            bool    `json:"include_in_goal_balance"`
	DataProvider                    string  `json:"data_provider"`
	DataProviderAccountID           string  `json:"data_provider_account_id"`
	IsManual                        bool    `json:"is_manual"`
	TransactionsCount               int     `json:"transactions_count"`
	HoldingsCount                   int     `json:"holdings_count"`
	ManualInvestmentsTrackingMethod string  `json:"manual_investments_tracking_method"`
	Order                           int     `json:"order"`
	Icon                            string  `json:"icon"`
	LogoURL                         string  `json:"logo_url"`
	IsClosed                        bool    `json:"is_closed"`
}

type AccountRecentBalance struct {
	ID               string `json:"id"`
	DisplayName      string `json:"display_name"`
	AccountTypeGroup string `json:"account_type_group"`
	RecentBalances   any    `json:"recent_balances"`
}

type AccountBalanceAt struct {
	ID               string  `json:"id"`
	DisplayName      string  `json:"display_name"`
	DisplayBalance   float64 `json:"display_balance"`
	AccountType      string  `json:"account_type"`
	AccountTypeGroup string  `json:"account_type_group"`
}

type Holding struct {
	ID         string  `json:"id"`
	Quantity   float64 `json:"quantity"`
	Basis      float64 `json:"basis"`
	TotalValue float64 `json:"total_value"`
}

type HistoryRecord struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"`
}

func (s *Service) GetAccountHoldings(ctx context.Context, accountID string) ([]Holding, error) {
	var resp struct {
		Portfolio struct {
			AggregateHoldings struct {
				Edges []struct {
					Node struct {
						ID         string  `json:"id"`
						Quantity   float64 `json:"quantity"`
						Basis      float64 `json:"basis"`
						TotalValue float64 `json:"totalValue"`
						Holdings   []struct {
							ID       string  `json:"id"`
							Quantity float64 `json:"quantity"`
							Name     string  `json:"name"`
							Ticker   string  `json:"ticker"`
							Account  struct {
								ID string `json:"id"`
							} `json:"account"`
						} `json:"holdings"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"aggregateHoldings"`
		} `json:"portfolio"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Web_GetHoldings",
		Query:         GetAccountHoldingsQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	var holdings []Holding
	for _, edge := range resp.Portfolio.AggregateHoldings.Edges {
		if accountID != "" {
			matched := false
			for _, h := range edge.Node.Holdings {
				if h.Account.ID == accountID {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		node := edge.Node
		holdings = append(holdings, Holding{
			ID:         node.ID,
			Quantity:   node.Quantity,
			Basis:      node.Basis,
			TotalValue: node.TotalValue,
		})
	}

	return holdings, nil
}

func (s *Service) GetAccountHistory(ctx context.Context, accountID string, startDate, endDate string) ([]HistoryRecord, error) {
	var resp struct {
		AggregateSnapshots []struct {
			Date    string  `json:"date"`
			Balance float64 `json:"balance"`
		} `json:"aggregateSnapshots"`
	}

	variables := map[string]any{
		"filters": map[string]any{},
	}
	if startDate != "" {
		variables["filters"].(map[string]any)["startDate"] = startDate
	}
	if endDate != "" {
		variables["filters"].(map[string]any)["endDate"] = endDate
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetAccountHistory",
		Query:         GetAccountHistoryQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	history := make([]HistoryRecord, len(resp.AggregateSnapshots))
	for i, r := range resp.AggregateSnapshots {
		history[i] = HistoryRecord{
			Date:   r.Date,
			Amount: r.Balance,
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
				Name    string `json:"name"`
				Display string `json:"display"`
			} `json:"type"`
			Subtype struct {
				Name    string `json:"name"`
				Display string `json:"display"`
			} `json:"subtype"`
			DisplayBalance                  float64 `json:"displayBalance"`
			CurrentBalance                  float64 `json:"currentBalance"`
			Limit                           float64 `json:"limit"`
			DataProviderCreditLimit         float64 `json:"dataProviderCreditLimit"`
			UpdatedAt                       string  `json:"updatedAt"`
			DisplayLastUpdatedAt            string  `json:"displayLastUpdatedAt"`
			DeactivatedAt                   string  `json:"deactivatedAt"`
			IsHidden                        bool    `json:"isHidden"`
			IsAsset                         bool    `json:"isAsset"`
			Mask                            string  `json:"mask"`
			CreatedAt                       string  `json:"createdAt"`
			IncludeInNetWorth               bool    `json:"includeInNetWorth"`
			HideFromList                    bool    `json:"hideFromList"`
			HideTransactionsFromReports     bool    `json:"hideTransactionsFromReports"`
			IncludeBalanceInNetWorth        bool    `json:"includeBalanceInNetWorth"`
			IncludeInGoalBalance            bool    `json:"includeInGoalBalance"`
			DataProvider                    string  `json:"dataProvider"`
			DataProviderAccountID           string  `json:"dataProviderAccountId"`
			IsManual                        bool    `json:"isManual"`
			TransactionsCount               int     `json:"transactionsCount"`
			HoldingsCount                   int     `json:"holdingsCount"`
			ManualInvestmentsTrackingMethod string  `json:"manualInvestmentsTrackingMethod"`
			Order                           int     `json:"order"`
			Icon                            string  `json:"icon"`
			LogoURL                         string  `json:"logoUrl"`
			IsClosed                        bool    `json:"isClosed"`
		} `json:"account"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetAccount",
		Query:         GetAccountQuery,
		Variables:     map[string]any{"id": id},
	}, &resp)

	if err != nil {
		return nil, err
	}

	return &Account{
		ID:                              resp.Account.ID,
		DisplayName:                     resp.Account.DisplayName,
		AccountType:                     resp.Account.AccountType.Name,
		AccountSubtype:                  resp.Account.Subtype.Name,
		DisplayBalance:                  resp.Account.DisplayBalance,
		CurrentBalance:                  resp.Account.CurrentBalance,
		Limit:                           resp.Account.Limit,
		DataProviderCreditLimit:         resp.Account.DataProviderCreditLimit,
		UpdatedAt:                       resp.Account.UpdatedAt,
		DisplayLastUpdatedAt:            resp.Account.DisplayLastUpdatedAt,
		DeactivatedAt:                   resp.Account.DeactivatedAt,
		IsHidden:                        resp.Account.IsHidden,
		IsAsset:                         resp.Account.IsAsset,
		Mask:                            resp.Account.Mask,
		CreatedAt:                       resp.Account.CreatedAt,
		IncludeInNetWorth:               resp.Account.IncludeInNetWorth,
		HideFromList:                    resp.Account.HideFromList,
		HideTransactionsFromReports:     resp.Account.HideTransactionsFromReports,
		IncludeBalanceInNetWorth:        resp.Account.IncludeBalanceInNetWorth,
		IncludeInGoalBalance:            resp.Account.IncludeInGoalBalance,
		DataProvider:                    resp.Account.DataProvider,
		DataProviderAccountID:           resp.Account.DataProviderAccountID,
		IsManual:                        resp.Account.IsManual,
		TransactionsCount:               resp.Account.TransactionsCount,
		HoldingsCount:                   resp.Account.HoldingsCount,
		ManualInvestmentsTrackingMethod: resp.Account.ManualInvestmentsTrackingMethod,
		Order:                           resp.Account.Order,
		Icon:                            resp.Account.Icon,
		LogoURL:                         resp.Account.LogoURL,
		IsClosed:                        resp.Account.IsClosed,
	}, nil
}

func (s *Service) GetAccountRecentBalances(ctx context.Context, startDate string) ([]AccountRecentBalance, error) {
	var resp struct {
		Accounts []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			Type        struct {
				Group string `json:"group"`
			} `json:"type"`
			RecentBalances any `json:"recentBalances"`
		} `json:"accounts"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetAccountRecentBalances",
		Query:         GetAccountRecentBalancesQuery,
		Variables:     map[string]any{"startDate": startDate},
	}, &resp)

	if err != nil {
		return nil, err
	}

	out := make([]AccountRecentBalance, len(resp.Accounts))
	for i, a := range resp.Accounts {
		out[i] = AccountRecentBalance{
			ID:               a.ID,
			DisplayName:      a.DisplayName,
			AccountTypeGroup: a.Type.Group,
			RecentBalances:   a.RecentBalances,
		}
	}

	return out, nil
}

func (s *Service) GetAccountBalancesAt(ctx context.Context, date string, accountIDs []string) ([]AccountBalanceAt, error) {
	var resp struct {
		Accounts []struct {
			ID             string  `json:"id"`
			DisplayName    string  `json:"displayName"`
			DisplayBalance float64 `json:"displayBalance"`
			Type           struct {
				Name  string `json:"name"`
				Group string `json:"group"`
			} `json:"type"`
		} `json:"accounts"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_GetDisplayBalanceAtDate",
		Query:         GetAccountBalancesAtQuery,
		Variables:     map[string]any{"date": date},
	}, &resp)

	if err != nil {
		return nil, err
	}

	filter := map[string]bool{}
	for _, id := range accountIDs {
		filter[id] = true
	}

	out := make([]AccountBalanceAt, 0, len(resp.Accounts))
	for _, a := range resp.Accounts {
		if len(filter) > 0 && !filter[a.ID] {
			continue
		}
		out = append(out, AccountBalanceAt{
			ID:               a.ID,
			DisplayName:      a.DisplayName,
			DisplayBalance:   a.DisplayBalance,
			AccountType:      a.Type.Name,
			AccountTypeGroup: a.Type.Group,
		})
	}

	return out, nil
}

func (s *Service) GetSnapshotsByAccountType(ctx context.Context, startDate, timeframe string) (any, error) {
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
		Variables: map[string]any{
			"startDate": startDate,
			"timeframe": timeframe,
		},
	}, &resp)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (s *Service) GetAggregateSnapshots(ctx context.Context, startDate, endDate, accountType string) (any, error) {
	var resp struct {
		AggregateSnapshots []struct {
			Date    string  `json:"date"`
			Balance float64 `json:"balance"`
		} `json:"aggregateSnapshots"`
	}

	filters := map[string]any{}
	if startDate != "" {
		filters["startDate"] = startDate
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
		Variables:     map[string]any{"filters": filters},
	}, &resp)

	if err != nil {
		return nil, err
	}

	return resp.AggregateSnapshots, nil
}

func (s *Service) GetAccountTypes(ctx context.Context) ([]string, error) {
	var resp struct {
		AccountTypeOptions []struct {
			Name    string `json:"name"`
			Display string `json:"display"`
		} `json:"accountTypes"`
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

func (s *Service) GetAccountsRefreshStatus(ctx context.Context) (map[string]any, error) {
	var resp struct {
		Accounts []struct {
			ID                string `json:"id"`
			HasSyncInProgress bool   `json:"hasSyncInProgress"`
		} `json:"accounts"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetAccountsRefreshStatus",
		Query:         GetAccountsRefreshStatusQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	accounts := make([]map[string]any, 0, len(resp.Accounts))
	isComplete := true
	for _, account := range resp.Accounts {
		if account.HasSyncInProgress {
			isComplete = false
		}
		accounts = append(accounts, map[string]any{
			"id":                   account.ID,
			"has_sync_in_progress": account.HasSyncInProgress,
		})
	}

	return map[string]any{
		"is_complete": isComplete,
		"status": func() string {
			if isComplete {
				return "complete"
			}
			return "syncing"
		}(),
		"accounts": accounts,
	}, nil
}

func (s *Service) ListAccounts(ctx context.Context) ([]Account, error) {
	var resp struct {
		Accounts []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			AccountType struct {
				Name    string `json:"name"`
				Display string `json:"display"`
			} `json:"type"`
			Subtype struct {
				Name    string `json:"name"`
				Display string `json:"display"`
			} `json:"subtype"`
			DisplayBalance                  float64 `json:"displayBalance"`
			CurrentBalance                  float64 `json:"currentBalance"`
			Limit                           float64 `json:"limit"`
			DataProviderCreditLimit         float64 `json:"dataProviderCreditLimit"`
			UpdatedAt                       string  `json:"updatedAt"`
			DisplayLastUpdatedAt            string  `json:"displayLastUpdatedAt"`
			DeactivatedAt                   string  `json:"deactivatedAt"`
			IsHidden                        bool    `json:"isHidden"`
			IsAsset                         bool    `json:"isAsset"`
			Mask                            string  `json:"mask"`
			CreatedAt                       string  `json:"createdAt"`
			IncludeInNetWorth               bool    `json:"includeInNetWorth"`
			HideFromList                    bool    `json:"hideFromList"`
			HideTransactionsFromReports     bool    `json:"hideTransactionsFromReports"`
			IncludeBalanceInNetWorth        bool    `json:"includeBalanceInNetWorth"`
			IncludeInGoalBalance            bool    `json:"includeInGoalBalance"`
			DataProvider                    string  `json:"dataProvider"`
			DataProviderAccountID           string  `json:"dataProviderAccountId"`
			IsManual                        bool    `json:"isManual"`
			TransactionsCount               int     `json:"transactionsCount"`
			HoldingsCount                   int     `json:"holdingsCount"`
			ManualInvestmentsTrackingMethod string  `json:"manualInvestmentsTrackingMethod"`
			Order                           int     `json:"order"`
			Icon                            string  `json:"icon"`
			LogoURL                         string  `json:"logoUrl"`
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
			ID:                              a.ID,
			DisplayName:                     a.DisplayName,
			AccountType:                     a.AccountType.Name,
			AccountSubtype:                  a.Subtype.Name,
			DisplayBalance:                  a.DisplayBalance,
			CurrentBalance:                  a.CurrentBalance,
			Limit:                           a.Limit,
			DataProviderCreditLimit:         a.DataProviderCreditLimit,
			UpdatedAt:                       a.UpdatedAt,
			DisplayLastUpdatedAt:            a.DisplayLastUpdatedAt,
			DeactivatedAt:                   a.DeactivatedAt,
			IsHidden:                        a.IsHidden,
			IsAsset:                         a.IsAsset,
			Mask:                            a.Mask,
			CreatedAt:                       a.CreatedAt,
			IncludeInNetWorth:               a.IncludeInNetWorth,
			HideFromList:                    a.HideFromList,
			HideTransactionsFromReports:     a.HideTransactionsFromReports,
			IncludeBalanceInNetWorth:        a.IncludeBalanceInNetWorth,
			IncludeInGoalBalance:            a.IncludeInGoalBalance,
			DataProvider:                    a.DataProvider,
			DataProviderAccountID:           a.DataProviderAccountID,
			IsManual:                        a.IsManual,
			TransactionsCount:               a.TransactionsCount,
			HoldingsCount:                   a.HoldingsCount,
			ManualInvestmentsTrackingMethod: a.ManualInvestmentsTrackingMethod,
			Order:                           a.Order,
			Icon:                            a.Icon,
			LogoURL:                         a.LogoURL,
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
		Variables: map[string]any{
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

	// This is the existing account refresh path; it covers Monarch's official
	// requestAccountsRefresh capability without adding a duplicate CLI command.
	variables := make(map[string]any)
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

	variables := map[string]any{"id": id}
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
		Variables:     map[string]any{"id": id},
	}, &resp)
}

func (s *Service) UploadAccountBalanceHistory(ctx context.Context, id string, r io.Reader) error {
	// Monarch exposes balance history upload as a multipart REST endpoint, not GraphQL.
	// Keep the same web headers and token shape as the GraphQL client.
	url := "https://api.monarch.com/account-balance-history/upload/"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := createBalanceHistoryFormFile(writer, "file", "history.csv")
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, r); err != nil {
		return err
	}
	writer.WriteField("account_id", id) //nolint:errcheck // best-effort write
	writer.Close()                      //nolint:errcheck // best-effort close

	req, err := newBalanceHistoryRequest(ctx, "POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Client-Platform", "web")
	req.Header.Set("User-Agent", graphql.UserAgent())
	if token := s.Client.TokenValue(); token != "" {
		req.Header.Set("Authorization", "Token "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // response body close

	if resp.StatusCode != 200 {
		return errors.New(errors.APIError, fmt.Sprintf("upload failed with status %d", resp.StatusCode), errors.CatAPI, false, nil)
	}

	return nil
}
