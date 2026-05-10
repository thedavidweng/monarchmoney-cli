package monarch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
)

type mockClient struct {
	token   string
	lastReq *graphql.Request
	handler func(req *graphql.Request, result interface{}) error
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

type fakeCSVWriter struct {
	failOnCall int
	calls      int
	err        error
}

func (w *fakeCSVWriter) Write(record []string) error {
	w.calls++
	if w.calls == w.failOnCall {
		return errors.New("write failed")
	}
	return nil
}

func (*fakeCSVWriter) Flush() {}

func (w *fakeCSVWriter) Error() error { return w.err }

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

type failingReadCloser struct{}

func (failingReadCloser) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func (failingReadCloser) Close() error { return nil }

func (m *mockClient) Do(_ context.Context, req *graphql.Request, result interface{}) error {
	m.lastReq = req
	if m.handler != nil {
		return m.handler(req, result)
	}
	return nil
}

func (m *mockClient) TokenValue() string {
	return m.token
}

func (m *mockClient) respond(result interface{}, payload string) error {
	if result == nil {
		return nil
	}
	return json.Unmarshal([]byte(payload), result)
}

func clientRespond(result interface{}, payload string) error {
	if result == nil {
		return nil
	}
	return json.Unmarshal([]byte(payload), result)
}

func newMockService(token string, handler func(req *graphql.Request, result interface{}) error) (*Service, *mockClient) {
	client := &mockClient{token: token, handler: handler}
	return NewService(client), client
}

func assertReq(t *testing.T, got *graphql.Request, op string) {
	t.Helper()
	if got == nil {
		t.Fatal("request not captured")
	}
	if got.OperationName != op {
		t.Fatalf("OperationName = %q, want %q", got.OperationName, op)
	}
}

func expectVars(t *testing.T, got map[string]interface{}, want map[string]interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Variables = %#v, want %#v", got, want)
	}
}

func runGraphQLCase(t *testing.T, op string, wantVars map[string]interface{}, payload string, call func(*Service) error) {
	t.Helper()

	var client *mockClient
	client = &mockClient{
		token: "token-123",
		handler: func(req *graphql.Request, result interface{}) error {
			assertReq(t, req, op)
			expectVars(t, req.Variables, wantVars)
			return client.respond(result, payload)
		},
	}

	if err := call(NewService(client)); err != nil {
		t.Fatalf("%s() error = %v", op, err)
	}
}

func runGraphQLErrorCase(t *testing.T, op string, wantVars map[string]interface{}, call func(*Service) error) {
	t.Helper()

	var client *mockClient
	client = &mockClient{
		token: "token-123",
		handler: func(req *graphql.Request, result interface{}) error {
			assertReq(t, req, op)
			expectVars(t, req.Variables, wantVars)
			return errors.New("boom")
		},
	}

	if err := call(NewService(client)); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("%s() error = %v, want boom", op, err)
	}
}

func TestServiceTokenValue(t *testing.T) {
	svc, _ := newMockService("abc123", nil)
	if got := svc.Client.TokenValue(); got != "abc123" {
		t.Fatalf("TokenValue() = %q, want %q", got, "abc123")
	}
}

func TestServiceAccountsAndCoreMethods(t *testing.T) {
	t.Run("list accounts", func(t *testing.T) {
		runGraphQLCase(t, "GetAccounts", nil, `{"accounts":[{"id":"a1","displayName":"Checking","type":{"name":"bank"},"subtype":{"name":"checking"},"displayBalance":42.5,"updatedAt":"2026-05-08"}]}`, func(s *Service) error {
			got, err := s.ListAccounts(context.Background())
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "a1", got[0].ID)
			assert.Equal(t, "bank", got[0].AccountType)
			assert.Equal(t, 42.5, got[0].DisplayBalance)
			return nil
		})
	})

	t.Run("get account", func(t *testing.T) {
		runGraphQLCase(t, "GetAccount", map[string]interface{}{"id": "acc-1"}, `{"account":{"id":"acc-1","displayName":"Cash","type":{"name":"cash"},"subtype":{"name":"cash"},"displayBalance":9.5,"updatedAt":"2026-05-08"}}`, func(s *Service) error {
			got, err := s.GetAccount(context.Background(), "acc-1")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "acc-1", got.ID)
			assert.Equal(t, "Cash", got.DisplayName)
			assert.Equal(t, "cash", got.AccountType)
			return nil
		})
	})

	t.Run("account types", func(t *testing.T) {
		runGraphQLCase(t, "GetAccountTypeOptions", nil, `{"accountTypes":[{"name":"bank"},{"name":"credit"}]}`, func(s *Service) error {
			got, err := s.GetAccountTypes(context.Background())
			require.NoError(t, err)
			assert.Equal(t, []string{"bank", "credit"}, got)
			return nil
		})
	})

	t.Run("refresh status", func(t *testing.T) {
		runGraphQLCase(t, "GetAccountsRefreshStatus", nil, `{"accounts":[{"id":"acc-1","hasSyncInProgress":false},{"id":"acc-2","hasSyncInProgress":true}]}`, func(s *Service) error {
			got, err := s.GetAccountsRefreshStatus(context.Background())
			require.NoError(t, err)
			assert.Equal(t, false, got["is_complete"])
			assert.Equal(t, "syncing", got["status"])
			require.Len(t, got["accounts"], 2)
			return nil
		})
	})

	t.Run("create manual account", func(t *testing.T) {
		runGraphQLCase(t, "CreateManualAccount", map[string]interface{}{"name": "Savings", "type": "bank", "balance": 10.0}, `{"createManualAccount":{"account":{"id":"a2","displayName":"Savings","displayBalance":10}}}`, func(s *Service) error {
			got, err := s.CreateManualAccount(context.Background(), "Savings", "bank", 10)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "a2", got.ID)
			assert.Equal(t, "Savings", got.DisplayName)
			assert.Equal(t, 10.0, got.DisplayBalance)
			return nil
		})
	})

	t.Run("update account", func(t *testing.T) {
		name := "New name"
		balance := 11.25
		runGraphQLCase(t, "UpdateAccount", map[string]interface{}{"id": "acc-1", "displayName": name, "balance": balance}, `{"updateAccount":{"account":{"id":"acc-1","displayName":"New name","displayBalance":11.25}}}`, func(s *Service) error {
			got, err := s.UpdateAccount(context.Background(), "acc-1", &name, &balance)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "New name", got.DisplayName)
			assert.Equal(t, 11.25, got.DisplayBalance)
			return nil
		})
	})

	t.Run("refresh accounts", func(t *testing.T) {
		runGraphQLCase(t, "RefreshAccounts", map[string]interface{}{"accountIds": []string{"a1", "a2"}}, `{"requestAccountsRefresh":{"ok":true}}`, func(s *Service) error {
			return s.RefreshAccounts(context.Background(), []string{"a1", "a2"})
		})
	})

	t.Run("delete account", func(t *testing.T) {
		runGraphQLCase(t, "DeleteAccount", map[string]interface{}{"id": "acc-1"}, `{"deleteAccount":{"ok":true}}`, func(s *Service) error {
			return s.DeleteAccount(context.Background(), "acc-1")
		})
	})

	t.Run("account holdings", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetHoldings", nil, `{"portfolio":{"aggregateHoldings":{"edges":[{"node":{"id":"h1","quantity":2,"basis":3,"totalValue":6,"holdings":[{"id":"h1-1","quantity":2,"name":"Alphabet","ticker":"GOOGL","account":{"id":"acc-1"}}]}},{"node":{"id":"h2","quantity":4,"basis":5,"totalValue":20,"holdings":[{"id":"h2-1","quantity":4,"name":"Other","ticker":"OTR","account":{"id":"acc-2"}}]}}]}}}`, func(s *Service) error {
			got, err := s.GetAccountHoldings(context.Background(), "acc-1")
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "h1", got[0].ID)
			assert.Equal(t, 3.0, got[0].Basis)
			assert.Equal(t, 6.0, got[0].TotalValue)
			return nil
		})
	})

	t.Run("account history", func(t *testing.T) {
		runGraphQLCase(t, "GetAccountHistory", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-05-01", "endDate": "2026-05-31"}}, `{"aggregateSnapshots":[{"date":"2026-05-01","balance":10}]}`, func(s *Service) error {
			got, err := s.GetAccountHistory(context.Background(), "acc-1", "2026-05-01", "2026-05-31")
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, 10.0, got[0].Amount)
			return nil
		})
	})

	t.Run("recent balances", func(t *testing.T) {
		runGraphQLCase(t, "GetAccountRecentBalances", map[string]interface{}{"startDate": "2026-05-01"}, `{"accounts":[{"id":"acc-1","displayName":"Checking","type":{"group":"asset"},"recentBalances":[1,2,3]}]}`, func(s *Service) error {
			got, err := s.GetAccountRecentBalances(context.Background(), "2026-05-01")
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "Checking", got[0].DisplayName)
			assert.Equal(t, "asset", got[0].AccountTypeGroup)
			assert.NotNil(t, got[0].RecentBalances)
			return nil
		})
	})

	t.Run("balance at date", func(t *testing.T) {
		runGraphQLCase(t, "Common_GetDisplayBalanceAtDate", map[string]interface{}{"date": "2026-05-10"}, `{"accounts":[{"id":"acc-1","displayName":"Checking","displayBalance":42.25,"type":{"name":"cash","group":"asset"}}]}`, func(s *Service) error {
			got, err := s.GetAccountBalancesAt(context.Background(), "2026-05-10", []string{"acc-1"})
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "acc-1", got[0].ID)
			assert.Equal(t, "Checking", got[0].DisplayName)
			assert.Equal(t, 42.25, got[0].DisplayBalance)
			assert.Equal(t, "cash", got[0].AccountType)
			assert.Equal(t, "asset", got[0].AccountTypeGroup)
			return nil
		})
	})

	t.Run("snapshots by type", func(t *testing.T) {
		runGraphQLCase(t, "GetSnapshotsByAccountType", map[string]interface{}{"startDate": "2026-05-01", "timeframe": "month"}, `{"snapshotsByAccountType":[{"accountType":"bank","month":"2026-05","balance":1}],"accountTypes":[{"name":"bank","group":"asset"}]}`, func(s *Service) error {
			got, err := s.GetSnapshotsByAccountType(context.Background(), "2026-05-01", "month")
			require.NoError(t, err)
			b, _ := json.Marshal(got)
			assert.Contains(t, string(b), "snapshotsByAccountType")
			return nil
		})
	})

	t.Run("aggregate snapshots", func(t *testing.T) {
		runGraphQLCase(t, "GetAggregateSnapshots", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-05-01", "endDate": "2026-05-31", "accountType": "bank"}}, `{"aggregateSnapshots":[{"date":"2026-05-01","balance":1}]}`, func(s *Service) error {
			got, err := s.GetAggregateSnapshots(context.Background(), "2026-05-01", "2026-05-31", "bank")
			require.NoError(t, err)
			require.Len(t, got, 1)
			return nil
		})
	})

	t.Run("aggregate snapshots default filters", func(t *testing.T) {
		runGraphQLCase(t, "GetAggregateSnapshots", map[string]interface{}{"filters": map[string]interface{}{}}, `{"aggregateSnapshots":[{"date":"2026-05-01","balance":1}]}`, func(s *Service) error {
			got, err := s.GetAggregateSnapshots(context.Background(), "", "", "")
			require.NoError(t, err)
			require.Len(t, got, 1)
			return nil
		})
	})
}

func TestServiceBudgetCashflowAndReferenceMethods(t *testing.T) {
	t.Run("get budget", func(t *testing.T) {
		runGraphQLCase(t, "GetJointPlanningData", map[string]interface{}{"startDate": "2026-05-01", "endDate": "2026-05-31"}, `{"budgetData":{"monthlyAmountsByCategory":[{"category":{"id":"cat-1","name":"Food"},"monthlyAmounts":[{"month":"2026-05","plannedCashFlowAmount":100,"actualAmount":80}]}]}}`, func(s *Service) error {
			got, err := s.GetBudget(context.Background(), "cat-1", "2026-05-01", "2026-05-31")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "cat-1", got.CategoryID)
			assert.Equal(t, "Food", got.CategoryName)
			assert.Equal(t, 100.0, got.Planned)
			assert.Equal(t, 80.0, got.Actual)
			return nil
		})
	})

	t.Run("list budgets", func(t *testing.T) {
		runGraphQLCase(t, "GetJointPlanningData", map[string]interface{}{"startDate": "2026-05-01", "endDate": "2026-05-31"}, `{"budgetData":{"monthlyAmountsByCategory":[{"category":{"id":"cat-1","name":"Food"},"monthlyAmounts":[{"month":"2026-05","plannedCashFlowAmount":100,"actualAmount":80}]}]}}`, func(s *Service) error {
			got, err := s.ListBudgets(context.Background(), ListBudgetsOptions{StartDate: "2026-05-01", EndDate: "2026-05-31"})
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "Food", got[0].CategoryName)
			return nil
		})
	})

	t.Run("set budget", func(t *testing.T) {
		runGraphQLCase(t, "SetBudget", map[string]interface{}{"input": map[string]interface{}{"categoryId": "cat-1", "amount": 75.5, "month": "2026-05-01"}}, `{"setBudget":{"budget":{"category":{"name":"Food"},"planned":75.5}}}`, func(s *Service) error {
			got, err := s.SetBudget(context.Background(), "cat-1", 75.5, "2026-05-01")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "Food", got.CategoryName)
			assert.Equal(t, 75.5, got.Planned)
			return nil
		})
	})

	t.Run("update flexible budget", func(t *testing.T) {
		runGraphQLCase(t, "UpdateFlexibleBudget", map[string]interface{}{"input": map[string]interface{}{"month": 5, "year": 2026, "plannedCashFlowAmount": 25.0}}, `{"updateOrCreateFlexBudgetItem":{"flexBudgetItem":{"month":5}}}`, func(s *Service) error {
			return s.UpdateFlexibleBudget(context.Background(), 5, 2026, 25)
		})
	})

	t.Run("rollover settings", func(t *testing.T) {
		runGraphQLCase(t, "UpdateFlexRolloverSettings", map[string]interface{}{"input": map[string]interface{}{"rolloverStartMonth": "may", "rolloverStartingBalance": 100.0, "rolloverEnabled": true}}, `{"updateBudgetSettings":{"budgetRolloverPeriod":{"id":"roll"}}}`, func(s *Service) error {
			return s.UpdateFlexRolloverSettings(context.Background(), "may", 100, true)
		})
	})

	t.Run("reset budget", func(t *testing.T) {
		runGraphQLCase(t, "ResetBudget", map[string]interface{}{"month": 5, "year": 2026}, `{"resetBudget":{"ok":true}}`, func(s *Service) error {
			return s.ResetBudget(context.Background(), 5, 2026)
		})
	})

	t.Run("cashflow", func(t *testing.T) {
		var calls []map[string]interface{}
		var client *mockClient
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result interface{}) error {
				assertReq(t, req, "GetTransactionsList")
				calls = append(calls, req.Variables)
				payload := `{"allTransactions":{"results":[{"id":"tx-1","date":"2026-01-01","amount":10,"plaidName":"plaid","merchant":{"name":"Paycheck"},"category":{"name":"Income"},"account":{"id":"acc-1","displayName":"Checking"}}],"totalCount":3}}`
				if len(calls) == 2 {
					payload = `{"allTransactions":{"results":[{"id":"tx-2","date":"2026-01-02","amount":-2,"plaidName":"plaid","merchant":{"name":"Transit"},"category":{"name":"Travel"},"account":{"id":"acc-1","displayName":"Checking"}},{"id":"tx-3","date":"2026-01-03","amount":-4,"plaidName":"plaid","merchant":{"name":"Coffee"},"category":{"name":"Food"},"account":{"id":"acc-1","displayName":"Checking"}}],"totalCount":3}}`
				}
				return client.respond(result, payload)
			},
		}
		got, err := NewService(client).ListCashflow(context.Background(), "2026-01-01", "2026-01-03")
		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "2026-01-01", got[0].Period)
		assert.Equal(t, 10.0, got[0].Income)
		assert.Equal(t, -2.0, got[1].Expense)
		assert.Equal(t, 0.0, got[1].Income)
		assert.Equal(t, -4.0, got[2].Expense)
		assert.Len(t, calls, 2)
	})

	t.Run("cashflow summary", func(t *testing.T) {
		runGraphQLCase(t, "GetCashflowSummary", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, `{"aggregates":[{"summary":{"sumIncome":200,"sumExpense":100,"savings":100,"savingsRate":50}}]}`, func(s *Service) error {
			got, err := s.GetCashflowSummary(context.Background(), "2026-01-01", "2026-01-31")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, 50.0, got.SavingsRate)
			return nil
		})
	})

	t.Run("cashflow categories", func(t *testing.T) {
		runGraphQLCase(t, "GetCashflowCategories", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, `{"aggregates":[{"groupBy":{"category":{"name":"Food"}},"summary":{"sum":100}}]}`, func(s *Service) error {
			got, err := s.GetCashflowCategories(context.Background(), "2026-01-01", "2026-01-31")
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "Food", got[0].Name)
			return nil
		})
	})

	t.Run("cashflow merchants", func(t *testing.T) {
		runGraphQLCase(t, "GetCashflowMerchants", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, `{"aggregates":[{"groupBy":{"merchant":{"name":"Store"}},"summary":{"sumIncome":0,"sumExpense":100}}]}`, func(s *Service) error {
			got, err := s.GetCashflowMerchants(context.Background(), "2026-01-01", "2026-01-31")
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "Store", got[0].Name)
			return nil
		})
	})

	t.Run("cashflow trends by category group", func(t *testing.T) {
		runGraphQLCase(t, "GetAggregatesGraphCategoryGroup", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-03-31", "categories": []string{"cat-1"}, "accounts": []string{"acc-1"}}}, `{"aggregates":[{"groupBy":{"categoryGroup":{"id":"grp-1"},"month":"2026-01"},"summary":{"sum":-120}}]}`, func(s *Service) error {
			got, err := s.GetCashflowTrends(context.Background(), CashflowTrendOptions{
				StartDate:   "2026-01-01",
				EndDate:     "2026-03-31",
				GroupBy:     "category-group",
				Period:      "month",
				CategoryIDs: []string{"cat-1"},
				AccountIDs:  []string{"acc-1"},
			})
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "grp-1", got[0].GroupID)
			assert.Equal(t, "2026-01", got[0].Period)
			assert.Equal(t, -120.0, got[0].Sum)
			return nil
		})
	})

	t.Run("rules list", func(t *testing.T) {
		runGraphQLCase(t, "GetTransactionRules", nil, `{"transactionRules":[{"id":"r1","order":1,"merchantCriteriaUseOriginalStatement":false,"merchantCriteria":[{"operator":"contains","value":"coffee"}],"merchantNameCriteria":[{"operator":"eq","value":"shop"}],"amountCriteria":{"operator":"gt","isExpense":true,"value":5,"valueRange":null},"categoryIds":["cat-1"],"accountIds":["acc-1"],"setCategoryAction":{"id":"cat-1","name":"Food"},"setMerchantAction":null,"addTagsAction":[{"id":"tag-1","name":"Trip","color":"blue"}],"linkGoalAction":null,"setHideFromReportsAction":false,"reviewStatusAction":"needs_review","recentApplicationCount":2,"lastAppliedAt":"2026-05-01T00:00:00Z"}]}`, func(s *Service) error {
			got, err := s.ListRules(context.Background())
			require.NoError(t, err)
			require.Len(t, got, 1)
			require.Len(t, got[0].MerchantNameCriteria, 2)
			assert.Equal(t, "coffee", got[0].MerchantNameCriteria[0].Value)
			assert.Equal(t, "shop", got[0].MerchantNameCriteria[1].Value)
			return nil
		})
	})
}

func TestServiceTagsCategoriesAndLookupMethods(t *testing.T) {
	t.Run("list tags", func(t *testing.T) {
		runGraphQLCase(t, "GetTags", nil, `{"householdTransactionTags":[{"id":"tag-1","name":"Trip","color":"blue"}]}`, func(s *Service) error {
			got, err := s.ListTags(context.Background())
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "Trip", got[0].Name)
			return nil
		})
	})

	t.Run("create tag", func(t *testing.T) {
		runGraphQLCase(t, "CreateTag", map[string]interface{}{"name": "Trip", "color": "blue"}, `{"createHouseholdTransactionTag":{"tag":{"id":"tag-1","name":"Trip","color":"blue"}}}`, func(s *Service) error {
			got, err := s.CreateTag(context.Background(), "Trip", "blue")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "tag-1", got.ID)
			assert.Equal(t, "Trip", got.Name)
			return nil
		})
	})

	t.Run("list category groups", func(t *testing.T) {
		runGraphQLCase(t, "GetCategoryGroups", nil, `{"categoryGroups":[{"id":"g1","name":"Income","type":"income","categories":[{"id":"c1","name":"Salary"}]}]}`, func(s *Service) error {
			got, err := s.ListCategoryGroups(context.Background())
			require.NoError(t, err)
			require.Len(t, got, 1)
			require.Len(t, got[0].Categories, 1)
			assert.Equal(t, "Salary", got[0].Categories[0].Name)
			return nil
		})
	})

	t.Run("list categories", func(t *testing.T) {
		runGraphQLCase(t, "GetCategories", nil, `{"categories":[{"id":"c1","name":"Food","order":2,"icon":"restaurant","group":{"id":"g1","name":"Expenses","type":"expense"}}]}`, func(s *Service) error {
			got, err := s.ListCategories(context.Background())
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "Expenses", got[0].GroupName)
			assert.Equal(t, "g1", got[0].GroupID)
			assert.Equal(t, "expense", got[0].GroupType)
			assert.Equal(t, "restaurant", got[0].Icon)
			assert.Equal(t, 2, got[0].Order)
			return nil
		})
	})

	t.Run("create category", func(t *testing.T) {
		runGraphQLCase(t, "CreateCategory", map[string]interface{}{"name": "Food", "groupId": "g1"}, `{"createCategory":{"category":{"id":"c1","name":"Food"}}}`, func(s *Service) error {
			got, err := s.CreateCategory(context.Background(), "Food", "g1")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "c1", got.ID)
			assert.Equal(t, "Food", got.Name)
			return nil
		})
	})

	t.Run("delete category", func(t *testing.T) {
		runGraphQLCase(t, "DeleteCategory", map[string]interface{}{"id": "c1"}, `{"deleteCategory":{"ok":true}}`, func(s *Service) error {
			return s.DeleteCategory(context.Background(), "c1")
		})
	})

	t.Run("delete categories", func(t *testing.T) {
		runGraphQLCase(t, "DeleteCategories", map[string]interface{}{"ids": []string{"c1", "c2"}}, `{"deleteTransactionCategories":{"ok":true}}`, func(s *Service) error {
			return s.DeleteCategories(context.Background(), []string{"c1", "c2"})
		})
	})

	t.Run("credit history", func(t *testing.T) {
		runGraphQLCase(t, "GetCreditScoreSnapshots", nil, `{"creditScoreSnapshots":[{"reportedDate":"2026-05-01","score":790,"user":{"id":"u-1"}}]}`, func(s *Service) error {
			got, err := s.GetCreditHistory(context.Background())
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, 790, got[0].Score)
			assert.Equal(t, "u-1", got[0].UserID)
			return nil
		})
	})

	t.Run("institutions", func(t *testing.T) {
		runGraphQLCase(t, "GetInstitutionSettings", nil, `{"credentials":[{"id":"c1","updateRequired":false,"disconnectedFromDataProviderAt":"","dataProvider":"plaid","institution":{"id":"i1","plaidInstitutionId":"https://bank.example","name":"Bank","status":"connected"}}]}`, func(s *Service) error {
			got, err := s.ListInstitutions(context.Background())
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "https://bank.example", got[0].URL)
			return nil
		})
	})

	t.Run("subscription", func(t *testing.T) {
		runGraphQLCase(t, "GetSubscriptionDetails", nil, `{"subscription":{"id":"sub-1","paymentSource":"card","referralCode":"REF","isOnFreeTrial":true,"hasPremiumEntitlement":true}}`, func(s *Service) error {
			got, err := s.GetSubscriptionDetails(context.Background())
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "card", got.PaymentSource)
			assert.Equal(t, "REF", got.ReferralCode)
			assert.True(t, got.IsOnFreeTrial)
			assert.True(t, got.HasPremiumEntitlement)
			return nil
		})
	})

	t.Run("recurring list", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetUpcomingRecurringTransactionItems", map[string]interface{}{"startDate": "2026-05-01", "endDate": "2026-06-01", "filters": map[string]interface{}{}}, `{"recurringTransactionItems":[{"stream":{"id":"r1","frequency":"monthly","amount":20,"isApproximate":false,"merchant":{"name":"Gym"}},"date":"2026-06-01","isPast":false,"transactionId":"tx-1","amount":20,"amountDiff":0,"category":{"id":"c1","name":"Food"},"account":{"id":"acc-1","displayName":"Checking"}}]}`, func(s *Service) error {
			got, err := s.ListRecurring(context.Background(), "2026-05-01", "2026-06-01")
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "Gym", got[0].Merchant)
			return nil
		})
	})

	t.Run("recurring item details preserve stream and item fields", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetUpcomingRecurringTransactionItems", map[string]interface{}{"startDate": "2025-05-01", "endDate": "2026-05-31", "filters": map[string]interface{}{}}, `{"recurringTransactionItems":[{"stream":{"id":"r1","frequency":"yearly","amount":120,"isApproximate":true,"merchant":{"name":"Cloud Box"}},"date":"2026-02-01","isPast":true,"transactionId":"tx-1","amount":120,"amountDiff":5,"category":{"id":"c1","name":"Software"},"account":{"id":"acc-1","displayName":"Checking"}}]}`, func(s *Service) error {
			got, err := s.ListRecurringItems(context.Background(), "2025-05-01", "2026-05-31")
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "r1", got[0].Stream.ID)
			assert.Equal(t, "yearly", got[0].Stream.Frequency)
			assert.Equal(t, true, got[0].Stream.IsApproximate)
			assert.Equal(t, "Cloud Box", got[0].Stream.MerchantName)
			assert.Equal(t, "Software", got[0].CategoryName)
			assert.Equal(t, "acc-1", got[0].AccountID)
			assert.Equal(t, "Checking", got[0].AccountName)
			return nil
		})
	})

	t.Run("recurring update", func(t *testing.T) {
		runGraphQLCase(t, "UpdateRecurringTransaction", map[string]interface{}{"id": "r1", "amount": 21.5}, `{"updateRecurringTransaction":{"recurringTransaction":{"id":"r1","amount":21.5}}}`, func(s *Service) error {
			got, err := s.UpdateRecurring(context.Background(), "r1", 21.5)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "r1", got.ID)
			assert.Equal(t, 21.5, got.Amount)
			return nil
		})
	})

	t.Run("goals list", func(t *testing.T) {
		runGraphQLCase(t, "Web_GoalsV2", nil, `{"goalsV2":[{"id":"goal-1","name":"Vacation"}]}`, func(s *Service) error {
			got, err := s.ListGoals(context.Background())
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "goal-1", got[0].ID)
			assert.Equal(t, "Vacation", got[0].Name)
			return nil
		})
	})
}

func TestServiceTransactionMethods(t *testing.T) {
	t.Run("get transaction", func(t *testing.T) {
		runGraphQLCase(t, "GetTransaction", map[string]interface{}{"id": "tx-1"}, `{"getTransaction":{"id":"tx-1","date":"2026-05-08","amount":-20,"merchant":{"name":"Store"},"category":{"name":"Food"},"notes":"lunch","account":{"id":"acc-1","displayName":"Checking"},"tags":[{"id":"tag-1","name":"Trip","color":"blue"}]}}`, func(s *Service) error {
			got, err := s.GetTransaction(context.Background(), "tx-1")
			require.NoError(t, err)
			require.NotNil(t, got)
			require.Len(t, got.Tags, 1)
			assert.Equal(t, "Store", got.Merchant)
			assert.Equal(t, "Trip", got.Tags[0].Name)
			assert.Equal(t, "acc-1", got.AccountID)
			return nil
		})
	})

	t.Run("transactions summary", func(t *testing.T) {
		runGraphQLCase(t, "GetTransactionsPage", map[string]interface{}{"filters": map[string]interface{}{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}, "startDate": "2026-05-01", "endDate": "2026-05-31"}}, `{"aggregates":[{"summary":{"avg":20,"count":1,"max":20,"maxExpense":20,"sum":20,"sumIncome":0,"sumExpense":20,"first":"2026-05-01","last":"2026-05-31"}}]}`, func(s *Service) error {
			got, err := s.GetTransactionsSummary(context.Background(), "2026-05-01", "2026-05-31")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, 1, got.Count)
			assert.Equal(t, 20.0, got.Sum)
			assert.Equal(t, "2026-05-01", got.First)
			return nil
		})
	})

	t.Run("duplicate transactions", func(t *testing.T) {
		var client *mockClient
		callCount := 0
		client = &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result interface{}) error {
				assertReq(t, req, "GetTransactionsList")
				expectVars(t, req.Variables, map[string]interface{}{"offset": 0, "limit": 1000, "filters": map[string]interface{}{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}, "startDate": "2026-05-01", "endDate": "2026-05-31"}})
				callCount++
				return client.respond(result, `{"allTransactions":{"results":[{"id":"tx-1","date":"2026-05-08","amount":-20,"plaidName":"plaid","merchant":{"name":"Store"},"account":{"id":"acc-1","displayName":"Checking"}},{"id":"tx-2","date":"2026-05-08","amount":-20,"plaidName":"plaid","merchant":{"name":"Store"},"account":{"id":"acc-1","displayName":"Checking"}},{"id":"tx-3","date":"2026-05-08","amount":-20,"plaidName":"plaid","merchant":{"name":"Store"},"account":{"id":"acc-1","displayName":"Checking"}}],"totalCount":3}}`)
			},
		}
		got, err := NewService(client).GetDuplicateTransactions(context.Background(), "2026-05-01", "2026-05-31")
		require.NoError(t, err)
		assert.Equal(t, 1, callCount)
		require.Len(t, got, 3)
	})

	t.Run("get transaction splits", func(t *testing.T) {
		runGraphQLCase(t, "TransactionSplitQuery", map[string]interface{}{"id": "tx-1"}, `{"getTransaction":{"id":"tx-1","amount":-100,"splitTransactions":[{"id":"s1","amount":-60,"notes":"groceries","merchant":{"name":"Store"},"category":{"name":"Food"}},{"id":"s2","amount":-40,"notes":"household","merchant":{"name":"Store"},"category":{"name":"Home"}}]}}`, func(s *Service) error {
			got, err := s.GetTransactionSplits(context.Background(), "tx-1")
			require.NoError(t, err)
			require.Len(t, got, 2)
			assert.Equal(t, -60.0, got[0].Amount)
			assert.Equal(t, "Home", got[1].Category)
			return nil
		})
	})

	t.Run("update transaction", func(t *testing.T) {
		notes := "updated"
		categoryID := "cat-1"
		runGraphQLCase(t, "Web_TransactionDrawerUpdateTransaction", map[string]interface{}{"input": map[string]interface{}{"id": "tx-1", "notes": notes, "category": categoryID}}, `{"updateTransaction":{"transaction":{"id":"tx-1","amount":0,"date":"","notes":"updated","hideFromReports":false,"needsReview":false,"category":{"name":"Food"},"merchant":{"name":""}}}}`, func(s *Service) error {
			got, err := s.UpdateTransaction(context.Background(), "tx-1", &notes, &categoryID, nil, nil, nil, nil, nil)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "updated", got.Notes)
			assert.Equal(t, "Food", got.Category)
			return nil
		})
	})

	t.Run("delete transaction", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteTransactionMutation", map[string]interface{}{"input": map[string]interface{}{"transactionId": "tx-1"}}, `{"deleteTransaction":{"ok":true}}`, func(s *Service) error {
			return s.DeleteTransaction(context.Background(), "tx-1")
		})
	})

	t.Run("update splits", func(t *testing.T) {
		runGraphQLCase(t, "Common_SplitTransactionMutation", map[string]interface{}{"input": map[string]interface{}{"transactionId": "tx-1", "splitData": []map[string]interface{}{{"amount": 10.0, "categoryId": "cat-1", "notes": "split"}}}}, `{"updateTransactionSplit":{"errors":[],"transaction":{"id":"tx-1","hasSplitTransactions":true,"splitTransactions":[]}}}`, func(s *Service) error {
			return s.UpdateTransactionSplits(context.Background(), "tx-1", []SplitInput{{Amount: 10, CategoryID: "cat-1", Notes: "split"}})
		})
	})

	t.Run("create transaction", func(t *testing.T) {
		runGraphQLCase(t, "Common_CreateTransactionMutation", map[string]interface{}{"input": map[string]interface{}{"date": "2026-05-08", "accountId": "acc-1", "amount": -20.0, "merchantName": "Store", "categoryId": "cat-1", "notes": "lunch", "shouldUpdateBalance": false}}, `{"createTransaction":{"transaction":{"id":"tx-1","amount":-20,"date":"2026-05-08","merchant":{"name":"Store"}}}}`, func(s *Service) error {
			got, err := s.CreateTransaction(context.Background(), -20, "Store", "2026-05-08", "cat-1", "acc-1", "lunch")
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, "tx-1", got.ID)
			assert.Equal(t, "Store", got.Merchant)
			return nil
		})
	})

	t.Run("set transaction tags", func(t *testing.T) {
		runGraphQLCase(t, "Web_SetTransactionTags", map[string]interface{}{"input": map[string]interface{}{"transactionId": "tx-1", "tagIds": []string{"tag-1", "tag-2"}}}, `{"setTransactionTags":{"ok":true}}`, func(s *Service) error {
			return s.SetTransactionTags(context.Background(), "tx-1", []string{"tag-1", "tag-2"})
		})
	})

	t.Run("list transactions", func(t *testing.T) {
		pending := false
		hideFromReports := true
		runGraphQLCase(t, "GetTransactionsList", map[string]interface{}{"limit": 50, "offset": 5, "filters": map[string]interface{}{"search": "lunch", "categories": []string{}, "accounts": []string{}, "tags": []string{}, "goals": []string{"goal-1"}, "startDate": "2026-05-01", "endDate": "2026-05-31", "isPending": false, "hideFromReports": true}}, `{"allTransactions":{"results":[{"id":"tx-1","date":"2026-05-08","amount":-20,"pending":false,"hideFromReports":true,"dataProviderDescription":"STORE 123","plaidName":"plaid","merchant":{"name":"Store"},"category":{"name":"Food","group":{"id":"grp-1","name":"Dining","type":"expense"}},"account":{"id":"acc-1","displayName":"Checking","order":4,"type":{"group":"asset"}},"ownedByUser":{"displayName":"Alex"},"goal":{"id":"goal-1","name":"Vacation"},"notes":"lunch"}],"totalCount":1}}`, func(s *Service) error {
			got, total, err := s.ListTransactions(context.Background(), ListTransactionsOptions{Limit: 50, Offset: 5, Search: "lunch", StartDate: "2026-05-01", EndDate: "2026-05-31", Pending: &pending, HideFromReports: &hideFromReports, GoalIDs: []string{"goal-1"}})
			require.NoError(t, err)
			assert.Equal(t, 1, total)
			require.Len(t, got, 1)
			assert.Equal(t, "Food", got[0].Category)
			assert.Equal(t, "acc-1", got[0].AccountID)
			assert.Equal(t, "goal-1", got[0].Goal.ID)
			assert.Equal(t, "Alex", got[0].OwnerDisplayName)
			assert.Equal(t, 4, got[0].AccountOrder)
			assert.Equal(t, "asset", got[0].AccountTypeGroup)
			assert.Equal(t, "grp-1", got[0].CategoryGroup.ID)
			assert.Equal(t, "STORE 123", got[0].DataProviderDescription)
			return nil
		})
	})

	t.Run("list all transactions pages until total reached", func(t *testing.T) {
		var calls []int
		client := &mockClient{
			token: "token-123",
			handler: func(req *graphql.Request, result interface{}) error {
				assertReq(t, req, "GetTransactionsList")
				offset := req.Variables["offset"].(int)
				calls = append(calls, offset)
				filters := req.Variables["filters"].(map[string]interface{})
				if filters["startDate"] != "2026-05-01" || filters["endDate"] != "2026-05-31" {
					t.Fatalf("filters = %#v", filters)
				}
				switch offset {
				case 0:
					return clientRespond(result, `{"allTransactions":{"results":[{"id":"tx-1","date":"2026-05-01","amount":-10,"merchant":{"name":"A"},"category":{"name":"Food"},"account":{"id":"acc-1"}}],"totalCount":2}}`)
				case 1:
					return clientRespond(result, `{"allTransactions":{"results":[{"id":"tx-2","date":"2026-05-02","amount":-20,"merchant":{"name":"B"},"category":{"name":"Food"},"account":{"id":"acc-1"}}],"totalCount":2}}`)
				default:
					t.Fatalf("unexpected offset %d", offset)
				}
				return nil
			},
		}
		got, err := NewService(client).ListAllTransactions(context.Background(), ListTransactionsOptions{Limit: 1, StartDate: "2026-05-01", EndDate: "2026-05-31"})
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, []int{0, 1}, calls)
	})
}

func TestServiceInvestments(t *testing.T) {
	t.Run("portfolio", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetPortfolio", map[string]interface{}{"portfolioInput": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-05-10", "accounts": []string{"acc-1"}}}, `{"portfolio":{"performance":{"totalValue":1000,"totalChangePercent":0.12,"totalChangeDollars":120},"aggregateHoldings":{"edges":[{"node":{"id":"node-1","quantity":2,"basis":400,"totalValue":1000,"security":{"id":"sec-1","ticker":"ABC","name":"ABC Fund","currentPrice":500},"holdings":[{"id":"hold-1","type":"equity","typeDisplay":"Equity","name":"ABC Fund","ticker":"ABC","quantity":2,"value":1000,"account":{"id":"acc-1","displayName":"Brokerage","type":{"name":"investment","display":"Investment"},"subtype":{"name":"brokerage","display":"Brokerage"}}}]}}]}}}`, func(s *Service) error {
			got, err := s.GetInvestmentPortfolio(context.Background(), InvestmentPortfolioOptions{StartDate: "2026-01-01", EndDate: "2026-05-10", AccountIDs: []string{"acc-1"}})
			require.NoError(t, err)
			assert.Equal(t, 1000.0, got.Performance.TotalValue)
			require.Len(t, got.Holdings, 1)
			assert.Equal(t, "node-1", got.Holdings[0].ID)
			assert.Equal(t, "sec-1", got.Holdings[0].Security.ID)
			require.Len(t, got.Holdings[0].Holdings, 1)
			assert.Equal(t, "Brokerage", got.Holdings[0].Holdings[0].Account.DisplayName)
			return nil
		})
	})

	t.Run("security performance", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetInvestmentsHoldingDrawerHistoricalPerformance", map[string]interface{}{"input": map[string]interface{}{"securityIds": []string{"sec-1"}, "startDate": "2026-01-01", "endDate": "2026-05-10"}}, `{"securityHistoricalPerformance":[{"security":{"id":"sec-1","ticker":"ABC","name":"ABC Fund"},"historicalChart":[{"date":"2026-01-01","returnPercent":0.1,"value":100}]}]}`, func(s *Service) error {
			got, err := s.GetSecurityPerformance(context.Background(), SecurityPerformanceOptions{SecurityIDs: []string{"sec-1"}, StartDate: "2026-01-01", EndDate: "2026-05-10", IncludeValues: true})
			require.NoError(t, err)
			require.Len(t, got, 1)
			assert.Equal(t, "ABC", got[0].Security.Ticker)
			require.Len(t, got[0].HistoricalChart, 1)
			require.NotNil(t, got[0].HistoricalChart[0].Value)
			assert.Equal(t, 100.0, *got[0].HistoricalChart[0].Value)
			return nil
		})
	})
}

func TestServiceCacheAndExportHelpers(t *testing.T) {
	t.Run("export csv", func(t *testing.T) {
		var buf bytes.Buffer
		err := ExportTransactionsCSV([]Transaction{{Date: "2026-05-08", Merchant: "Store", Category: "Food", Amount: -20, Notes: "lunch"}}, &buf)
		if err != nil {
			t.Fatalf("ExportTransactionsCSV() error = %v", err)
		}
		if !strings.Contains(buf.String(), "Date,Merchant,Category,Amount,Notes") || !strings.Contains(buf.String(), "2026-05-08,Store,Food,-20.00,lunch") {
			t.Fatalf("ExportTransactionsCSV() output = %q", buf.String())
		}
	})

	t.Run("export csv header error", func(t *testing.T) {
		original := newCSVWriter
		newCSVWriter = func(io.Writer) csvWriter { return &fakeCSVWriter{failOnCall: 1} }
		defer func() { newCSVWriter = original }()

		if err := ExportTransactionsCSV([]Transaction{{Date: "2026-05-08"}}, io.Discard); err == nil {
			t.Fatal("ExportTransactionsCSV() error = nil, want failure")
		}
	})

	t.Run("export csv row error", func(t *testing.T) {
		original := newCSVWriter
		newCSVWriter = func(io.Writer) csvWriter { return &fakeCSVWriter{failOnCall: 2} }
		defer func() { newCSVWriter = original }()

		if err := ExportTransactionsCSV([]Transaction{{Date: "2026-05-08"}}, io.Discard); err == nil {
			t.Fatal("ExportTransactionsCSV() error = nil, want failure")
		}
	})

	t.Run("export csv flush error", func(t *testing.T) {
		original := newCSVWriter
		newCSVWriter = func(io.Writer) csvWriter { return &fakeCSVWriter{err: errors.New("flush failed")} }
		defer func() { newCSVWriter = original }()

		if err := ExportTransactionsCSV([]Transaction{{Date: "2026-05-08"}}, io.Discard); err == nil {
			t.Fatal("ExportTransactionsCSV() error = nil, want failure")
		}
	})
}

func TestServiceErrorBranches(t *testing.T) {
	t.Run("accounts and reference methods", func(t *testing.T) {
		runGraphQLErrorCase(t, "GetAccounts", nil, func(s *Service) error { _, err := s.ListAccounts(context.Background()); return err })
		runGraphQLErrorCase(t, "GetAccount", map[string]interface{}{"id": "acc-1"}, func(s *Service) error { _, err := s.GetAccount(context.Background(), "acc-1"); return err })
		runGraphQLErrorCase(t, "GetAccountTypeOptions", nil, func(s *Service) error { _, err := s.GetAccountTypes(context.Background()); return err })
		runGraphQLErrorCase(t, "GetAccountsRefreshStatus", nil, func(s *Service) error { _, err := s.GetAccountsRefreshStatus(context.Background()); return err })
		runGraphQLErrorCase(t, "CreateManualAccount", map[string]interface{}{"name": "Savings", "type": "bank", "balance": 10.0}, func(s *Service) error {
			_, err := s.CreateManualAccount(context.Background(), "Savings", "bank", 10)
			return err
		})
		runGraphQLErrorCase(t, "UpdateAccount", map[string]interface{}{"id": "acc-1"}, func(s *Service) error { _, err := s.UpdateAccount(context.Background(), "acc-1", nil, nil); return err })
		runGraphQLErrorCase(t, "RefreshAccounts", map[string]interface{}{}, func(s *Service) error { return s.RefreshAccounts(context.Background(), nil) })
		runGraphQLErrorCase(t, "DeleteAccount", map[string]interface{}{"id": "acc-1"}, func(s *Service) error { return s.DeleteAccount(context.Background(), "acc-1") })
		runGraphQLErrorCase(t, "Web_GetHoldings", nil, func(s *Service) error { _, err := s.GetAccountHoldings(context.Background(), "acc-1"); return err })
		runGraphQLErrorCase(t, "GetAccountHistory", map[string]interface{}{"filters": map[string]interface{}{}}, func(s *Service) error {
			_, err := s.GetAccountHistory(context.Background(), "acc-1", "", "")
			return err
		})
		runGraphQLErrorCase(t, "GetAccountRecentBalances", map[string]interface{}{"startDate": "2026-05-01"}, func(s *Service) error {
			_, err := s.GetAccountRecentBalances(context.Background(), "2026-05-01")
			return err
		})
		runGraphQLErrorCase(t, "Common_GetDisplayBalanceAtDate", map[string]interface{}{"date": "2026-05-10"}, func(s *Service) error {
			_, err := s.GetAccountBalancesAt(context.Background(), "2026-05-10", nil)
			return err
		})
		runGraphQLErrorCase(t, "GetSnapshotsByAccountType", map[string]interface{}{"startDate": "2026-05-01", "timeframe": "month"}, func(s *Service) error {
			_, err := s.GetSnapshotsByAccountType(context.Background(), "2026-05-01", "month")
			return err
		})
		runGraphQLErrorCase(t, "GetAggregateSnapshots", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-05-01"}}, func(s *Service) error {
			_, err := s.GetAggregateSnapshots(context.Background(), "2026-05-01", "", "")
			return err
		})
	})

	t.Run("budgets cashflow and lookup", func(t *testing.T) {
		runGraphQLErrorCase(t, "GetJointPlanningData", map[string]interface{}{"startDate": "2026-05-01", "endDate": "2026-05-31"}, func(s *Service) error {
			_, err := s.GetBudget(context.Background(), "cat-1", "2026-05-01", "2026-05-31")
			return err
		})
		runGraphQLErrorCase(t, "GetJointPlanningData", map[string]interface{}{"startDate": "", "endDate": ""}, func(s *Service) error {
			_, err := s.ListBudgets(context.Background(), ListBudgetsOptions{})
			return err
		})
		runGraphQLErrorCase(t, "SetBudget", map[string]interface{}{"input": map[string]interface{}{"categoryId": "cat-1", "amount": 10.0, "month": "2026-05-01"}}, func(s *Service) error {
			_, err := s.SetBudget(context.Background(), "cat-1", 10, "2026-05-01")
			return err
		})
		runGraphQLErrorCase(t, "ResetBudget", map[string]interface{}{"month": 1, "year": 2026}, func(s *Service) error { return s.ResetBudget(context.Background(), 1, 2026) })
		runGraphQLErrorCase(t, "GetTransactionsList", map[string]interface{}{"offset": 0, "limit": 1000, "filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, func(s *Service) error {
			_, err := s.ListCashflow(context.Background(), "2026-01-01", "2026-01-31")
			return err
		})
		runGraphQLErrorCase(t, "GetCashflowSummary", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, func(s *Service) error {
			_, err := s.GetCashflowSummary(context.Background(), "2026-01-01", "2026-01-31")
			return err
		})
		runGraphQLErrorCase(t, "GetCashflowCategories", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, func(s *Service) error {
			_, err := s.GetCashflowCategories(context.Background(), "2026-01-01", "2026-01-31")
			return err
		})
		runGraphQLErrorCase(t, "GetCashflowMerchants", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, func(s *Service) error {
			_, err := s.GetCashflowMerchants(context.Background(), "2026-01-01", "2026-01-31")
			return err
		})
		runGraphQLErrorCase(t, "GetAggregatesGraph", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-03-31"}}, func(s *Service) error {
			_, err := s.GetCashflowTrends(context.Background(), CashflowTrendOptions{StartDate: "2026-01-01", EndDate: "2026-03-31", GroupBy: "category", Period: "month"})
			return err
		})
		runGraphQLErrorCase(t, "GetTransactionRules", nil, func(s *Service) error { _, err := s.ListRules(context.Background()); return err })
		runGraphQLErrorCase(t, "GetCategoryGroups", nil, func(s *Service) error { _, err := s.ListCategoryGroups(context.Background()); return err })
		runGraphQLErrorCase(t, "GetCategories", nil, func(s *Service) error { _, err := s.ListCategories(context.Background()); return err })
		runGraphQLErrorCase(t, "CreateCategory", map[string]interface{}{"name": "Food", "groupId": "g1"}, func(s *Service) error { _, err := s.CreateCategory(context.Background(), "Food", "g1"); return err })
		runGraphQLErrorCase(t, "GetCreditScoreSnapshots", nil, func(s *Service) error { _, err := s.GetCreditHistory(context.Background()); return err })
		runGraphQLErrorCase(t, "GetInstitutionSettings", nil, func(s *Service) error { _, err := s.ListInstitutions(context.Background()); return err })
		runGraphQLErrorCase(t, "Web_GetUpcomingRecurringTransactionItems", map[string]interface{}{"startDate": "2026-05-01", "endDate": "2026-06-01", "filters": map[string]interface{}{}}, func(s *Service) error {
			_, err := s.ListRecurring(context.Background(), "2026-05-01", "2026-06-01")
			return err
		})
		runGraphQLErrorCase(t, "UpdateRecurringTransaction", map[string]interface{}{"id": "r1", "amount": 21.5}, func(s *Service) error { _, err := s.UpdateRecurring(context.Background(), "r1", 21.5); return err })
		runGraphQLErrorCase(t, "GetSubscriptionDetails", nil, func(s *Service) error { _, err := s.GetSubscriptionDetails(context.Background()); return err })
		runGraphQLErrorCase(t, "Web_GoalsV2", nil, func(s *Service) error { _, err := s.ListGoals(context.Background()); return err })
		runGraphQLErrorCase(t, "GetTags", nil, func(s *Service) error { _, err := s.ListTags(context.Background()); return err })
		runGraphQLErrorCase(t, "CreateTag", map[string]interface{}{"name": "Trip", "color": "blue"}, func(s *Service) error { _, err := s.CreateTag(context.Background(), "Trip", "blue"); return err })
	})

	t.Run("transactions", func(t *testing.T) {
		runGraphQLErrorCase(t, "GetTransaction", map[string]interface{}{"id": "tx-1"}, func(s *Service) error { _, err := s.GetTransaction(context.Background(), "tx-1"); return err })
		runGraphQLErrorCase(t, "GetTransactionsPage", map[string]interface{}{"filters": map[string]interface{}{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}, "startDate": "2026-05-01", "endDate": "2026-05-31"}}, func(s *Service) error {
			_, err := s.GetTransactionsSummary(context.Background(), "2026-05-01", "2026-05-31")
			return err
		})
		runGraphQLErrorCase(t, "GetTransactionsList", map[string]interface{}{"limit": 1000, "offset": 0, "filters": map[string]interface{}{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}, "startDate": "2026-05-01", "endDate": "2026-05-31"}}, func(s *Service) error {
			_, err := s.GetDuplicateTransactions(context.Background(), "2026-05-01", "2026-05-31")
			return err
		})
		runGraphQLErrorCase(t, "Web_TransactionDrawerUpdateTransaction", map[string]interface{}{"input": map[string]interface{}{"id": "tx-1"}}, func(s *Service) error {
			_, err := s.UpdateTransaction(context.Background(), "tx-1", nil, nil, nil, nil, nil, nil, nil)
			return err
		})
		runGraphQLErrorCase(t, "Common_DeleteTransactionMutation", map[string]interface{}{"input": map[string]interface{}{"transactionId": "tx-1"}}, func(s *Service) error { return s.DeleteTransaction(context.Background(), "tx-1") })
		runGraphQLErrorCase(t, "Common_CreateTransactionMutation", map[string]interface{}{"input": map[string]interface{}{"date": "2026-05-08", "accountId": "", "amount": -20.0, "merchantName": "Store", "categoryId": "cat-1", "notes": "", "shouldUpdateBalance": false}}, func(s *Service) error {
			_, err := s.CreateTransaction(context.Background(), -20, "Store", "2026-05-08", "cat-1", "", "")
			return err
		})
		runGraphQLErrorCase(t, "Web_SetTransactionTags", map[string]interface{}{"input": map[string]interface{}{"transactionId": "tx-1", "tagIds": []string{"tag-1"}}}, func(s *Service) error { return s.SetTransactionTags(context.Background(), "tx-1", []string{"tag-1"}) })
		runGraphQLErrorCase(t, "GetTransactionsList", map[string]interface{}{"limit": 10, "offset": 0, "filters": map[string]interface{}{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, func(s *Service) error {
			_, _, err := s.ListTransactions(context.Background(), ListTransactionsOptions{Limit: 10})
			return err
		})
	})

	t.Run("unavailable transaction paths", func(t *testing.T) {
		if err := NewService(&mockClient{}).UploadAttachment(context.Background(), "tx-1", "file.pdf"); err == nil || !strings.Contains(err.Error(), "FEATURE_UNAVAILABLE") {
			t.Fatalf("UploadAttachment() error = %v, want feature unavailable", err)
		}
	})

	t.Run("investments", func(t *testing.T) {
		runGraphQLErrorCase(t, "Web_GetPortfolio", map[string]interface{}{"portfolioInput": map[string]interface{}{}}, func(s *Service) error {
			_, err := s.GetInvestmentPortfolio(context.Background(), InvestmentPortfolioOptions{})
			return err
		})
		runGraphQLErrorCase(t, "Web_GetSecuritiesHistoricalPerformance", map[string]interface{}{"input": map[string]interface{}{"securityIds": []string{"sec-1"}, "startDate": "2026-01-01", "endDate": "2026-05-10"}}, func(s *Service) error {
			_, err := s.GetSecurityPerformance(context.Background(), SecurityPerformanceOptions{SecurityIDs: []string{"sec-1"}, StartDate: "2026-01-01", EndDate: "2026-05-10"})
			return err
		})
	})
}

func TestServiceHTTPHelpers(t *testing.T) {
	t.Run("download attachment success", func(t *testing.T) {
		orig := http.DefaultTransport
		defer func() { http.DefaultTransport = orig }()
		http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, "GET", req.Method)
			require.Equal(t, "https://files.example/attachment.csv", req.URL.String())
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("hello"))}, nil
		})

		var buf bytes.Buffer
		svc, _ := newMockService("token-123", nil)
		require.NoError(t, svc.DownloadAttachment(context.Background(), "https://files.example/attachment.csv", &buf))
		assert.Equal(t, "hello", buf.String())
	})

	t.Run("download attachment error", func(t *testing.T) {
		svc, _ := newMockService("token-123", nil)
		assert.Error(t, svc.DownloadAttachment(context.Background(), "://", io.Discard))
	})

	t.Run("download attachment transport error", func(t *testing.T) {
		orig := http.DefaultTransport
		defer func() { http.DefaultTransport = orig }()
		http.DefaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})

		var buf bytes.Buffer
		svc, _ := newMockService("token-123", nil)
		assert.Error(t, svc.DownloadAttachment(context.Background(), "https://files.example/attachment.csv", &buf))
	})

	t.Run("download attachment non-200", func(t *testing.T) {
		orig := http.DefaultTransport
		defer func() { http.DefaultTransport = orig }()
		http.DefaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(""))}, nil
		})

		var buf bytes.Buffer
		svc, _ := newMockService("token-123", nil)
		assert.Error(t, svc.DownloadAttachment(context.Background(), "https://files.example/attachment.csv", &buf))
	})

	t.Run("upload account balance history", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, "POST", req.Method)
			require.Equal(t, "web", req.Header.Get("Client-Platform"))
			require.Equal(t, "Token tok", req.Header.Get("Authorization"))
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
		})

		tmp := filepath.Join(t.TempDir(), "sample.csv")
		if err := os.WriteFile(tmp, []byte("a,b\n1,2\n"), 0600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		file, err := os.Open(tmp)
		require.NoError(t, err)
		defer file.Close()

		svc := NewService(&mockClient{token: "tok"})
		require.NoError(t, svc.UploadAccountBalanceHistory(context.Background(), "acc-1", file))
	})

	t.Run("upload account balance history non-200", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(""))}, nil
		})

		tmp := filepath.Join(t.TempDir(), "sample.csv")
		require.NoError(t, os.WriteFile(tmp, []byte("date,amount\n"), 0600))
		file, err := os.Open(tmp)
		require.NoError(t, err)
		defer file.Close()

		svc := NewService(&mockClient{token: "tok"})
		assert.Error(t, svc.UploadAccountBalanceHistory(context.Background(), "acc-1", file))
	})

	t.Run("upload account balance history network error", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})

		tmp := filepath.Join(t.TempDir(), "sample.csv")
		require.NoError(t, os.WriteFile(tmp, []byte("date,amount\n"), 0600))
		file, err := os.Open(tmp)
		require.NoError(t, err)
		defer file.Close()

		svc := NewService(&mockClient{token: "tok"})
		assert.Error(t, svc.UploadAccountBalanceHistory(context.Background(), "acc-1", file))
	})

	t.Run("upload account balance history request error", func(t *testing.T) {
		original := newBalanceHistoryRequest
		newBalanceHistoryRequest = func(context.Context, string, string, io.Reader) (*http.Request, error) {
			return nil, errors.New("request failed")
		}
		defer func() { newBalanceHistoryRequest = original }()

		svc := NewService(&mockClient{token: "tok"})
		assert.Error(t, svc.UploadAccountBalanceHistory(context.Background(), "acc-1", strings.NewReader("date,amount\n")))
	})

	t.Run("upload account balance history read error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "tok"})
		assert.Error(t, svc.UploadAccountBalanceHistory(context.Background(), "acc-1", failingReader{}))
	})

	t.Run("upload account balance history form file error", func(t *testing.T) {
		original := createBalanceHistoryFormFile
		createBalanceHistoryFormFile = func(*multipart.Writer, string, string) (io.Writer, error) {
			return nil, errors.New("form file failed")
		}
		defer func() { createBalanceHistoryFormFile = original }()

		svc := NewService(&mockClient{token: "tok"})
		assert.Error(t, svc.UploadAccountBalanceHistory(context.Background(), "acc-1", strings.NewReader("date,amount\n")))
	})

	t.Run("list transaction attachments", func(t *testing.T) {
		got, err := NewService(&mockClient{token: "tok"}).ListTransactionAttachments(context.Background(), "tx-1")
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("upload attachment unavailable", func(t *testing.T) {
		tmp := filepath.Join(t.TempDir(), "receipt.pdf")
		require.NoError(t, os.WriteFile(tmp, []byte("pdf"), 0600))
		err := NewService(&mockClient{token: "tok"}).UploadAttachment(context.Background(), "tx-1", tmp)
		assert.ErrorContains(t, err, "FEATURE_UNAVAILABLE")
	})
}
