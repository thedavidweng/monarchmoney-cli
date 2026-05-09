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
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].ID != "a1" || got[0].AccountType != "bank" || got[0].DisplayBalance != 42.5 {
				t.Fatalf("ListAccounts() = %#v", got)
			}
			return nil
		})
	})

	t.Run("get account", func(t *testing.T) {
		runGraphQLCase(t, "GetAccount", map[string]interface{}{"id": "acc-1"}, `{"account":{"id":"acc-1","displayName":"Cash","type":{"name":"cash"},"subtype":{"name":"cash"},"displayBalance":9.5,"updatedAt":"2026-05-08"}}`, func(s *Service) error {
			got, err := s.GetAccount(context.Background(), "acc-1")
			if err != nil {
				return err
			}
			if got.ID != "acc-1" || got.DisplayName != "Cash" || got.AccountType != "cash" {
				t.Fatalf("GetAccount() = %#v", got)
			}
			return nil
		})
	})

	t.Run("account types", func(t *testing.T) {
		runGraphQLCase(t, "GetAccountTypeOptions", nil, `{"accountTypes":[{"name":"bank"},{"name":"credit"}]}`, func(s *Service) error {
			got, err := s.GetAccountTypes(context.Background())
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(got, []string{"bank", "credit"}) {
				t.Fatalf("GetAccountTypes() = %#v", got)
			}
			return nil
		})
	})

	t.Run("refresh status", func(t *testing.T) {
		runGraphQLCase(t, "GetAccountsRefreshStatus", nil, `{"accountRefreshProgress":{"isComplete":true,"status":"done","startTime":"s","endTime":"e"}}`, func(s *Service) error {
			got, err := s.GetAccountsRefreshStatus(context.Background())
			if err != nil {
				return err
			}
			if got["is_complete"] != true || got["status"] != "done" {
				t.Fatalf("GetAccountsRefreshStatus() = %#v", got)
			}
			return nil
		})
	})

	t.Run("create manual account", func(t *testing.T) {
		runGraphQLCase(t, "CreateManualAccount", map[string]interface{}{"name": "Savings", "type": "bank", "balance": 10.0}, `{"createManualAccount":{"account":{"id":"a2","displayName":"Savings","displayBalance":10}}}`, func(s *Service) error {
			got, err := s.CreateManualAccount(context.Background(), "Savings", "bank", 10)
			if err != nil {
				return err
			}
			if got.ID != "a2" || got.DisplayName != "Savings" || got.DisplayBalance != 10 {
				t.Fatalf("CreateManualAccount() = %#v", got)
			}
			return nil
		})
	})

	t.Run("update account", func(t *testing.T) {
		name := "New name"
		balance := 11.25
		runGraphQLCase(t, "UpdateAccount", map[string]interface{}{"id": "acc-1", "displayName": name, "balance": balance}, `{"updateAccount":{"account":{"id":"acc-1","displayName":"New name","displayBalance":11.25}}}`, func(s *Service) error {
			got, err := s.UpdateAccount(context.Background(), "acc-1", &name, &balance)
			if err != nil {
				return err
			}
			if got.DisplayName != "New name" || got.DisplayBalance != 11.25 {
				t.Fatalf("UpdateAccount() = %#v", got)
			}
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
		runGraphQLCase(t, "Web_GetHoldings", map[string]interface{}{"input": map[string]interface{}{"accountId": "acc-1"}}, `{"portfolio":{"aggregateHoldings":{"edges":[{"node":{"id":"h1","quantity":2,"basis":3,"totalValue":6}}]}}}`, func(s *Service) error {
			got, err := s.GetAccountHoldings(context.Background(), "acc-1")
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].ID != "h1" || got[0].Basis != 3 || got[0].TotalValue != 6 {
				t.Fatalf("GetAccountHoldings() = %#v", got)
			}
			return nil
		})
	})

	t.Run("account history", func(t *testing.T) {
		runGraphQLCase(t, "GetAccountHistory", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-05-01", "endDate": "2026-05-31"}}, `{"aggregateSnapshots":[{"date":"2026-05-01","balance":10}]}`, func(s *Service) error {
			got, err := s.GetAccountHistory(context.Background(), "acc-1", "2026-05-01", "2026-05-31")
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].Amount != 10 {
				t.Fatalf("GetAccountHistory() = %#v", got)
			}
			return nil
		})
	})

	t.Run("recent balances", func(t *testing.T) {
		runGraphQLCase(t, "GetAccountRecentBalances", map[string]interface{}{"startDate": "2026-05-01"}, `{"accounts":[{"id":"acc-1","recentBalances":[1,2,3]}]}`, func(s *Service) error {
			got, err := s.GetAccountRecentBalances(context.Background(), "2026-05-01")
			if err != nil {
				return err
			}
			b, _ := json.Marshal(got)
			if !strings.Contains(string(b), "recentBalances") {
				t.Fatalf("GetAccountRecentBalances() = %s", string(b))
			}
			return nil
		})
	})

	t.Run("snapshots by type", func(t *testing.T) {
		runGraphQLCase(t, "GetSnapshotsByAccountType", map[string]interface{}{"startDate": "2026-05-01", "timeframe": "month"}, `{"snapshotsByAccountType":[{"accountType":"bank","month":"2026-05","balance":1}],"accountTypes":[{"name":"bank","group":"asset"}]}`, func(s *Service) error {
			got, err := s.GetSnapshotsByAccountType(context.Background(), "2026-05-01", "month")
			if err != nil {
				return err
			}
			b, _ := json.Marshal(got)
			if !strings.Contains(string(b), "snapshotsByAccountType") {
				t.Fatalf("GetSnapshotsByAccountType() = %s", string(b))
			}
			return nil
		})
	})

	t.Run("aggregate snapshots", func(t *testing.T) {
		runGraphQLCase(t, "GetAggregateSnapshots", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-05-01", "endDate": "2026-05-31", "accountType": "bank"}}, `{"aggregateSnapshots":[{"date":"2026-05-01","balance":1}]}`, func(s *Service) error {
			got, err := s.GetAggregateSnapshots(context.Background(), "2026-05-01", "2026-05-31", "bank")
			if err != nil {
				return err
			}
			if reflect.ValueOf(got).Len() != 1 {
				b, _ := json.Marshal(got)
				t.Fatalf("GetAggregateSnapshots() = %s", string(b))
			}
			return nil
		})
	})
}

func TestServiceBudgetCashflowAndReferenceMethods(t *testing.T) {
	t.Run("get budget", func(t *testing.T) {
		runGraphQLCase(t, "GetJointPlanningData", map[string]interface{}{"startDate": "2026-05-01", "endDate": "2026-05-31"}, `{"budgetData":{"monthlyAmountsByCategory":[{"category":{"id":"cat-1","name":"Food"},"monthlyAmounts":[{"month":"2026-05","plannedCashFlowAmount":100,"actualAmount":80}]}]}}`, func(s *Service) error {
			got, err := s.GetBudget(context.Background(), "cat-1", "2026-05-01", "2026-05-31")
			if err != nil {
				return err
			}
			if got.CategoryID != "cat-1" || got.CategoryName != "Food" || got.Planned != 100 || got.Actual != 80 {
				t.Fatalf("GetBudget() = %#v", got)
			}
			return nil
		})
	})

	t.Run("list budgets", func(t *testing.T) {
		runGraphQLCase(t, "GetJointPlanningData", map[string]interface{}{"startDate": "2026-05-01", "endDate": "2026-05-31"}, `{"budgetData":{"monthlyAmountsByCategory":[{"category":{"id":"cat-1","name":"Food"},"monthlyAmounts":[{"month":"2026-05","plannedCashFlowAmount":100,"actualAmount":80}]}]}}`, func(s *Service) error {
			got, err := s.ListBudgets(context.Background(), ListBudgetsOptions{StartDate: "2026-05-01", EndDate: "2026-05-31"})
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].CategoryName != "Food" {
				t.Fatalf("ListBudgets() = %#v", got)
			}
			return nil
		})
	})

	t.Run("set budget", func(t *testing.T) {
		runGraphQLCase(t, "SetBudget", map[string]interface{}{"input": map[string]interface{}{"categoryId": "cat-1", "amount": 75.5, "month": "2026-05-01"}}, `{"setBudget":{"budget":{"category":{"name":"Food"},"planned":75.5}}}`, func(s *Service) error {
			got, err := s.SetBudget(context.Background(), "cat-1", 75.5, "2026-05-01")
			if err != nil {
				return err
			}
			if got.CategoryName != "Food" || got.Planned != 75.5 {
				t.Fatalf("SetBudget() = %#v", got)
			}
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
		runGraphQLCase(t, "GetCashflow", map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31"}, `{"cashflow":{"byPeriod":[{"period":"2026-01","income":200,"expense":100,"savings":100}]}}`, func(s *Service) error {
			got, err := s.ListCashflow(context.Background(), "2026-01-01", "2026-01-31")
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].Period != "2026-01" {
				t.Fatalf("ListCashflow() = %#v", got)
			}
			return nil
		})
	})

	t.Run("cashflow summary", func(t *testing.T) {
		runGraphQLCase(t, "GetCashflowSummary", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, `{"aggregates":[{"summary":{"sumIncome":200,"sumExpense":100,"savings":100,"savingsRate":50}}]}`, func(s *Service) error {
			got, err := s.GetCashflowSummary(context.Background(), "2026-01-01", "2026-01-31")
			if err != nil {
				return err
			}
			if got.SavingsRate != 50 {
				t.Fatalf("GetCashflowSummary() = %#v", got)
			}
			return nil
		})
	})

	t.Run("cashflow categories", func(t *testing.T) {
		runGraphQLCase(t, "GetCashflowCategories", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, `{"aggregates":[{"groupBy":{"category":{"name":"Food"}},"summary":{"sum":100}}]}`, func(s *Service) error {
			got, err := s.GetCashflowCategories(context.Background(), "2026-01-01", "2026-01-31")
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].Name != "Food" {
				t.Fatalf("GetCashflowCategories() = %#v", got)
			}
			return nil
		})
	})

	t.Run("cashflow merchants", func(t *testing.T) {
		runGraphQLCase(t, "GetCashflowMerchants", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, `{"aggregates":[{"groupBy":{"merchant":{"name":"Store"}},"summary":{"sumIncome":0,"sumExpense":100}}]}`, func(s *Service) error {
			got, err := s.GetCashflowMerchants(context.Background(), "2026-01-01", "2026-01-31")
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].Name != "Store" {
				t.Fatalf("GetCashflowMerchants() = %#v", got)
			}
			return nil
		})
	})
}

func TestServiceTagsCategoriesAndLookupMethods(t *testing.T) {
	t.Run("list tags", func(t *testing.T) {
		runGraphQLCase(t, "GetTags", nil, `{"householdTransactionTags":[{"id":"tag-1","name":"Trip","color":"blue"}]}`, func(s *Service) error {
			got, err := s.ListTags(context.Background())
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].Name != "Trip" {
				t.Fatalf("ListTags() = %#v", got)
			}
			return nil
		})
	})

	t.Run("create tag", func(t *testing.T) {
		runGraphQLCase(t, "CreateTag", map[string]interface{}{"name": "Trip", "color": "blue"}, `{"createHouseholdTransactionTag":{"tag":{"id":"tag-1","name":"Trip","color":"blue"}}}`, func(s *Service) error {
			got, err := s.CreateTag(context.Background(), "Trip", "blue")
			if err != nil {
				return err
			}
			if got.ID != "tag-1" || got.Name != "Trip" {
				t.Fatalf("CreateTag() = %#v", got)
			}
			return nil
		})
	})

	t.Run("list category groups", func(t *testing.T) {
		runGraphQLCase(t, "GetCategoryGroups", nil, `{"categoryGroups":[{"id":"g1","name":"Income","type":"income","categories":[{"id":"c1","name":"Salary"}]}]}`, func(s *Service) error {
			got, err := s.ListCategoryGroups(context.Background())
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].Categories[0].Name != "Salary" {
				t.Fatalf("ListCategoryGroups() = %#v", got)
			}
			return nil
		})
	})

	t.Run("list categories", func(t *testing.T) {
		runGraphQLCase(t, "GetCategories", nil, `{"categories":[{"id":"c1","name":"Food","group":{"id":"g1","name":"Expenses"}}]}`, func(s *Service) error {
			got, err := s.ListCategories(context.Background())
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].GroupName != "Expenses" {
				t.Fatalf("ListCategories() = %#v", got)
			}
			return nil
		})
	})

	t.Run("create category", func(t *testing.T) {
		runGraphQLCase(t, "CreateCategory", map[string]interface{}{"name": "Food", "groupId": "g1"}, `{"createCategory":{"category":{"id":"c1","name":"Food"}}}`, func(s *Service) error {
			got, err := s.CreateCategory(context.Background(), "Food", "g1")
			if err != nil {
				return err
			}
			if got.ID != "c1" || got.Name != "Food" {
				t.Fatalf("CreateCategory() = %#v", got)
			}
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
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].Score != 790 || got[0].UserID != "u-1" {
				t.Fatalf("GetCreditHistory() = %#v", got)
			}
			return nil
		})
	})

	t.Run("institutions", func(t *testing.T) {
		runGraphQLCase(t, "GetInstitutionSettings", nil, `{"credentials":[{"id":"c1","updateRequired":false,"disconnectedFromDataProviderAt":"","dataProvider":"plaid","institution":{"id":"i1","plaidInstitutionId":"https://bank.example","name":"Bank","status":"connected"}}]}`, func(s *Service) error {
			got, err := s.ListInstitutions(context.Background())
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].URL != "https://bank.example" {
				t.Fatalf("ListInstitutions() = %#v", got)
			}
			return nil
		})
	})

	t.Run("subscription", func(t *testing.T) {
		runGraphQLCase(t, "GetSubscriptionDetails", nil, `{"subscription":{"id":"sub-1","paymentSource":"card","referralCode":"REF","isOnFreeTrial":true,"hasPremiumEntitlement":true}}`, func(s *Service) error {
			got, err := s.GetSubscriptionDetails(context.Background())
			if err != nil {
				return err
			}
			if got.PaymentSource != "card" || got.ReferralCode != "REF" || !got.IsOnFreeTrial || !got.HasPremiumEntitlement {
				t.Fatalf("GetSubscriptionDetails() = %#v", got)
			}
			return nil
		})
	})

	t.Run("recurring list", func(t *testing.T) {
		runGraphQLCase(t, "Web_GetUpcomingRecurringTransactionItems", map[string]interface{}{"startDate": "2026-05-01", "endDate": "2026-06-01", "filters": map[string]interface{}{}}, `{"recurringTransactionItems":[{"stream":{"id":"r1","frequency":"monthly","amount":20,"isApproximate":false,"merchant":{"name":"Gym"}},"date":"2026-06-01","isPast":false,"transactionId":"tx-1","amount":20,"amountDiff":0,"category":{"id":"c1","name":"Food"},"account":{"id":"acc-1","displayName":"Checking"}}]}`, func(s *Service) error {
			got, err := s.ListRecurring(context.Background(), "2026-05-01", "2026-06-01")
			if err != nil {
				return err
			}
			if len(got) != 1 || got[0].Merchant != "Gym" {
				t.Fatalf("ListRecurring() = %#v", got)
			}
			return nil
		})
	})

	t.Run("recurring update", func(t *testing.T) {
		runGraphQLCase(t, "UpdateRecurringTransaction", map[string]interface{}{"id": "r1", "amount": 21.5}, `{"updateRecurringTransaction":{"recurringTransaction":{"id":"r1","amount":21.5}}}`, func(s *Service) error {
			got, err := s.UpdateRecurring(context.Background(), "r1", 21.5)
			if err != nil {
				return err
			}
			if got.ID != "r1" || got.Amount != 21.5 {
				t.Fatalf("UpdateRecurring() = %#v", got)
			}
			return nil
		})
	})
}

func TestServiceTransactionMethods(t *testing.T) {
	t.Run("get transaction", func(t *testing.T) {
		runGraphQLCase(t, "GetTransaction", map[string]interface{}{"id": "tx-1"}, `{"getTransaction":{"id":"tx-1","date":"2026-05-08","amount":-20,"merchant":{"name":"Store"},"category":{"name":"Food"},"notes":"lunch","account":{"id":"acc-1","displayName":"Checking"},"tags":[{"id":"tag-1","name":"Trip","color":"blue"}]}}`, func(s *Service) error {
			got, err := s.GetTransaction(context.Background(), "tx-1")
			if err != nil {
				return err
			}
			if got.Merchant != "Store" || len(got.Tags) != 1 || got.Tags[0].Name != "Trip" || got.AccountID != "acc-1" {
				t.Fatalf("GetTransaction() = %#v", got)
			}
			return nil
		})
	})

	t.Run("transactions summary", func(t *testing.T) {
		runGraphQLCase(t, "GetTransactionsPage", map[string]interface{}{"filters": map[string]interface{}{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}, "startDate": "2026-05-01", "endDate": "2026-05-31"}}, `{"aggregates":[{"summary":{"avg":20,"count":1,"max":20,"maxExpense":20,"sum":20,"sumIncome":0,"sumExpense":20,"first":"2026-05-01","last":"2026-05-31"}}]}`, func(s *Service) error {
			got, err := s.GetTransactionsSummary(context.Background(), "2026-05-01", "2026-05-31")
			if err != nil {
				return err
			}
			if got.Count != 1 || got.Sum != 20 || got.First != "2026-05-01" {
				t.Fatalf("GetTransactionsSummary() = %#v", got)
			}
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
		if err != nil {
			t.Fatalf("GetDuplicateTransactions() error = %v", err)
		}
		if callCount != 1 {
			t.Fatalf("GetDuplicateTransactions() calls = %d, want 1", callCount)
		}
		if len(got) != 3 {
			t.Fatalf("GetDuplicateTransactions() = %#v", got)
		}
	})

	t.Run("transaction splits unavailable", func(t *testing.T) {
		got, err := NewService(&mockClient{}).GetTransactionSplits(context.Background(), "tx-1")
		if err == nil || !strings.Contains(err.Error(), "FEATURE_UNAVAILABLE") {
			t.Fatalf("GetTransactionSplits() error = %v, want feature unavailable", err)
		}
		if got != nil {
			t.Fatalf("GetTransactionSplits() = %#v, want nil", got)
		}
	})

	t.Run("update transaction", func(t *testing.T) {
		notes := "updated"
		categoryID := "cat-1"
		runGraphQLCase(t, "Web_TransactionDrawerUpdateTransaction", map[string]interface{}{"input": map[string]interface{}{"id": "tx-1", "notes": notes, "category": categoryID}}, `{"updateTransaction":{"transaction":{"id":"tx-1","notes":"updated","category":{"name":"Food"}}}}`, func(s *Service) error {
			got, err := s.UpdateTransaction(context.Background(), "tx-1", &notes, &categoryID)
			if err != nil {
				return err
			}
			if got.Notes != "updated" || got.Category != "Food" {
				t.Fatalf("UpdateTransaction() = %#v", got)
			}
			return nil
		})
	})

	t.Run("delete transaction", func(t *testing.T) {
		runGraphQLCase(t, "Common_DeleteTransactionMutation", map[string]interface{}{"input": map[string]interface{}{"transactionId": "tx-1"}}, `{"deleteTransaction":{"ok":true}}`, func(s *Service) error {
			return s.DeleteTransaction(context.Background(), "tx-1")
		})
	})

	t.Run("update splits unavailable", func(t *testing.T) {
		err := NewService(&mockClient{}).UpdateTransactionSplits(context.Background(), "tx-1", []SplitInput{{Amount: 10, CategoryID: "cat-1", Notes: "split"}})
		if err == nil || !strings.Contains(err.Error(), "FEATURE_UNAVAILABLE") {
			t.Fatalf("UpdateTransactionSplits() error = %v, want feature unavailable", err)
		}
	})

	t.Run("create transaction", func(t *testing.T) {
		runGraphQLCase(t, "Common_CreateTransactionMutation", map[string]interface{}{"input": map[string]interface{}{"date": "2026-05-08", "accountId": "acc-1", "amount": -20.0, "merchantName": "Store", "categoryId": "cat-1", "notes": "lunch", "shouldUpdateBalance": false}}, `{"createTransaction":{"transaction":{"id":"tx-1","amount":-20,"date":"2026-05-08","merchant":{"name":"Store"}}}}`, func(s *Service) error {
			got, err := s.CreateTransaction(context.Background(), -20, "Store", "2026-05-08", "cat-1", "acc-1", "lunch")
			if err != nil {
				return err
			}
			if got.ID != "tx-1" || got.Merchant != "Store" {
				t.Fatalf("CreateTransaction() = %#v", got)
			}
			return nil
		})
	})

	t.Run("set transaction tags", func(t *testing.T) {
		runGraphQLCase(t, "Web_SetTransactionTags", map[string]interface{}{"input": map[string]interface{}{"transactionId": "tx-1", "tagIds": []string{"tag-1", "tag-2"}}}, `{"setTransactionTags":{"ok":true}}`, func(s *Service) error {
			return s.SetTransactionTags(context.Background(), "tx-1", []string{"tag-1", "tag-2"})
		})
	})

	t.Run("list transactions", func(t *testing.T) {
		runGraphQLCase(t, "GetTransactionsList", map[string]interface{}{"limit": 50, "offset": 5, "filters": map[string]interface{}{"search": "lunch", "categories": []string{}, "accounts": []string{}, "tags": []string{}, "startDate": "2026-05-01", "endDate": "2026-05-31"}}, `{"allTransactions":{"results":[{"id":"tx-1","date":"2026-05-08","amount":-20,"plaidName":"plaid","merchant":{"name":"Store"},"category":{"name":"Food"},"account":{"id":"acc-1","displayName":"Checking"},"notes":"lunch"}],"totalCount":1}}`, func(s *Service) error {
			got, total, err := s.ListTransactions(context.Background(), ListTransactionsOptions{Limit: 50, Offset: 5, Search: "lunch", StartDate: "2026-05-01", EndDate: "2026-05-31"})
			if err != nil {
				return err
			}
			if total != 1 || len(got) != 1 || got[0].Category != "Food" || got[0].AccountID != "acc-1" {
				t.Fatalf("ListTransactions() = %#v, %d", got, total)
			}
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
		runGraphQLErrorCase(t, "CreateManualAccount", map[string]interface{}{"name": "Savings", "type": "bank", "balance": 10.0}, func(s *Service) error { _, err := s.CreateManualAccount(context.Background(), "Savings", "bank", 10); return err })
		runGraphQLErrorCase(t, "UpdateAccount", map[string]interface{}{"id": "acc-1"}, func(s *Service) error { _, err := s.UpdateAccount(context.Background(), "acc-1", nil, nil); return err })
		runGraphQLErrorCase(t, "RefreshAccounts", map[string]interface{}{}, func(s *Service) error { return s.RefreshAccounts(context.Background(), nil) })
		runGraphQLErrorCase(t, "DeleteAccount", map[string]interface{}{"id": "acc-1"}, func(s *Service) error { return s.DeleteAccount(context.Background(), "acc-1") })
		runGraphQLErrorCase(t, "Web_GetHoldings", map[string]interface{}{"input": map[string]interface{}{"accountId": "acc-1"}}, func(s *Service) error { _, err := s.GetAccountHoldings(context.Background(), "acc-1"); return err })
		runGraphQLErrorCase(t, "GetAccountHistory", map[string]interface{}{"filters": map[string]interface{}{}}, func(s *Service) error { _, err := s.GetAccountHistory(context.Background(), "acc-1", "", ""); return err })
		runGraphQLErrorCase(t, "GetAccountRecentBalances", map[string]interface{}{"startDate": "2026-05-01"}, func(s *Service) error { _, err := s.GetAccountRecentBalances(context.Background(), "2026-05-01"); return err })
		runGraphQLErrorCase(t, "GetSnapshotsByAccountType", map[string]interface{}{"startDate": "2026-05-01", "timeframe": "month"}, func(s *Service) error { _, err := s.GetSnapshotsByAccountType(context.Background(), "2026-05-01", "month"); return err })
		runGraphQLErrorCase(t, "GetAggregateSnapshots", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-05-01"}}, func(s *Service) error { _, err := s.GetAggregateSnapshots(context.Background(), "2026-05-01", "", ""); return err })
	})

	t.Run("budgets cashflow and lookup", func(t *testing.T) {
		runGraphQLErrorCase(t, "GetJointPlanningData", map[string]interface{}{"startDate": "2026-05-01", "endDate": "2026-05-31"}, func(s *Service) error { _, err := s.GetBudget(context.Background(), "cat-1", "2026-05-01", "2026-05-31"); return err })
		runGraphQLErrorCase(t, "GetJointPlanningData", map[string]interface{}{"startDate": "", "endDate": ""}, func(s *Service) error { _, err := s.ListBudgets(context.Background(), ListBudgetsOptions{}); return err })
		runGraphQLErrorCase(t, "SetBudget", map[string]interface{}{"input": map[string]interface{}{"categoryId": "cat-1", "amount": 10.0, "month": "2026-05-01"}}, func(s *Service) error { _, err := s.SetBudget(context.Background(), "cat-1", 10, "2026-05-01"); return err })
		runGraphQLErrorCase(t, "ResetBudget", map[string]interface{}{"month": 1, "year": 2026}, func(s *Service) error { return s.ResetBudget(context.Background(), 1, 2026) })
		runGraphQLErrorCase(t, "GetCashflow", map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31"}, func(s *Service) error { _, err := s.ListCashflow(context.Background(), "2026-01-01", "2026-01-31"); return err })
		runGraphQLErrorCase(t, "GetCashflowSummary", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, func(s *Service) error { _, err := s.GetCashflowSummary(context.Background(), "2026-01-01", "2026-01-31"); return err })
		runGraphQLErrorCase(t, "GetCashflowCategories", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, func(s *Service) error { _, err := s.GetCashflowCategories(context.Background(), "2026-01-01", "2026-01-31"); return err })
		runGraphQLErrorCase(t, "GetCashflowMerchants", map[string]interface{}{"filters": map[string]interface{}{"startDate": "2026-01-01", "endDate": "2026-01-31", "search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, func(s *Service) error { _, err := s.GetCashflowMerchants(context.Background(), "2026-01-01", "2026-01-31"); return err })
		runGraphQLErrorCase(t, "GetCategoryGroups", nil, func(s *Service) error { _, err := s.ListCategoryGroups(context.Background()); return err })
		runGraphQLErrorCase(t, "GetCategories", nil, func(s *Service) error { _, err := s.ListCategories(context.Background()); return err })
		runGraphQLErrorCase(t, "CreateCategory", map[string]interface{}{"name": "Food", "groupId": "g1"}, func(s *Service) error { _, err := s.CreateCategory(context.Background(), "Food", "g1"); return err })
		runGraphQLErrorCase(t, "GetCreditScoreSnapshots", nil, func(s *Service) error { _, err := s.GetCreditHistory(context.Background()); return err })
		runGraphQLErrorCase(t, "GetInstitutionSettings", nil, func(s *Service) error { _, err := s.ListInstitutions(context.Background()); return err })
		runGraphQLErrorCase(t, "Web_GetUpcomingRecurringTransactionItems", map[string]interface{}{"startDate": "2026-05-01", "endDate": "2026-06-01", "filters": map[string]interface{}{}}, func(s *Service) error { _, err := s.ListRecurring(context.Background(), "2026-05-01", "2026-06-01"); return err })
		runGraphQLErrorCase(t, "UpdateRecurringTransaction", map[string]interface{}{"id": "r1", "amount": 21.5}, func(s *Service) error { _, err := s.UpdateRecurring(context.Background(), "r1", 21.5); return err })
		runGraphQLErrorCase(t, "GetSubscriptionDetails", nil, func(s *Service) error { _, err := s.GetSubscriptionDetails(context.Background()); return err })
		runGraphQLErrorCase(t, "GetTags", nil, func(s *Service) error { _, err := s.ListTags(context.Background()); return err })
		runGraphQLErrorCase(t, "CreateTag", map[string]interface{}{"name": "Trip", "color": "blue"}, func(s *Service) error { _, err := s.CreateTag(context.Background(), "Trip", "blue"); return err })
	})

	t.Run("transactions", func(t *testing.T) {
		runGraphQLErrorCase(t, "GetTransaction", map[string]interface{}{"id": "tx-1"}, func(s *Service) error { _, err := s.GetTransaction(context.Background(), "tx-1"); return err })
		runGraphQLErrorCase(t, "GetTransactionsPage", map[string]interface{}{"filters": map[string]interface{}{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}, "startDate": "2026-05-01", "endDate": "2026-05-31"}}, func(s *Service) error { _, err := s.GetTransactionsSummary(context.Background(), "2026-05-01", "2026-05-31"); return err })
		runGraphQLErrorCase(t, "GetTransactionsList", map[string]interface{}{"limit": 1000, "offset": 0, "filters": map[string]interface{}{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}, "startDate": "2026-05-01", "endDate": "2026-05-31"}}, func(s *Service) error { _, err := s.GetDuplicateTransactions(context.Background(), "2026-05-01", "2026-05-31"); return err })
		runGraphQLErrorCase(t, "Web_TransactionDrawerUpdateTransaction", map[string]interface{}{"input": map[string]interface{}{"id": "tx-1"}}, func(s *Service) error { _, err := s.UpdateTransaction(context.Background(), "tx-1", nil, nil); return err })
		runGraphQLErrorCase(t, "Common_DeleteTransactionMutation", map[string]interface{}{"input": map[string]interface{}{"transactionId": "tx-1"}}, func(s *Service) error { return s.DeleteTransaction(context.Background(), "tx-1") })
		runGraphQLErrorCase(t, "Common_CreateTransactionMutation", map[string]interface{}{"input": map[string]interface{}{"date": "2026-05-08", "accountId": "", "amount": -20.0, "merchantName": "Store", "categoryId": "cat-1", "notes": "", "shouldUpdateBalance": false}}, func(s *Service) error { _, err := s.CreateTransaction(context.Background(), -20, "Store", "2026-05-08", "cat-1", "", ""); return err })
		runGraphQLErrorCase(t, "Web_SetTransactionTags", map[string]interface{}{"input": map[string]interface{}{"transactionId": "tx-1", "tagIds": []string{"tag-1"}}}, func(s *Service) error { return s.SetTransactionTags(context.Background(), "tx-1", []string{"tag-1"}) })
		runGraphQLErrorCase(t, "GetTransactionsList", map[string]interface{}{"limit": 10, "offset": 0, "filters": map[string]interface{}{"search": "", "categories": []string{}, "accounts": []string{}, "tags": []string{}}}, func(s *Service) error { _, _, err := s.ListTransactions(context.Background(), ListTransactionsOptions{Limit: 10}); return err })
	})

	t.Run("unavailable transaction paths", func(t *testing.T) {
		if _, err := NewService(&mockClient{}).GetTransactionSplits(context.Background(), "tx-1"); err == nil || !strings.Contains(err.Error(), "FEATURE_UNAVAILABLE") {
			t.Fatalf("GetTransactionSplits() error = %v, want feature unavailable", err)
		}
		if err := NewService(&mockClient{}).UpdateTransactionSplits(context.Background(), "tx-1", nil); err == nil || !strings.Contains(err.Error(), "FEATURE_UNAVAILABLE") {
			t.Fatalf("UpdateTransactionSplits() error = %v, want feature unavailable", err)
		}
		if _, err := NewService(&mockClient{}).ListTransactionAttachments(context.Background(), "tx-1"); err == nil || !strings.Contains(err.Error(), "FEATURE_UNAVAILABLE") {
			t.Fatalf("ListTransactionAttachments() error = %v, want feature unavailable", err)
		}
		if err := NewService(&mockClient{}).UploadAttachment(context.Background(), "tx-1", "file.pdf"); err == nil || !strings.Contains(err.Error(), "FEATURE_UNAVAILABLE") {
			t.Fatalf("UploadAttachment() error = %v, want feature unavailable", err)
		}
	})
}

func TestServiceHTTPHelpers(t *testing.T) {
	t.Run("download attachment success", func(t *testing.T) {
		orig := http.DefaultTransport
		defer func() { http.DefaultTransport = orig }()
		http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != "GET" || req.URL.String() != "https://files.example/attachment.csv" {
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("hello"))}, nil
		})

		var buf bytes.Buffer
		svc, _ := newMockService("token-123", nil)
		if err := svc.DownloadAttachment(context.Background(), "https://files.example/attachment.csv", &buf); err != nil {
			t.Fatalf("DownloadAttachment() error = %v", err)
		}
		if buf.String() != "hello" {
			t.Fatalf("DownloadAttachment() = %q", buf.String())
		}
	})

	t.Run("download attachment error", func(t *testing.T) {
		svc, _ := newMockService("token-123", nil)
		if err := svc.DownloadAttachment(context.Background(), "://", io.Discard); err == nil {
			t.Fatal("DownloadAttachment() error = nil, want failure")
		}
	})

	t.Run("download attachment transport error", func(t *testing.T) {
		orig := http.DefaultTransport
		defer func() { http.DefaultTransport = orig }()
		http.DefaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})

		var buf bytes.Buffer
		svc, _ := newMockService("token-123", nil)
		if err := svc.DownloadAttachment(context.Background(), "https://files.example/attachment.csv", &buf); err == nil {
			t.Fatal("DownloadAttachment() error = nil, want failure")
		}
	})

	t.Run("download attachment non-200", func(t *testing.T) {
		orig := http.DefaultTransport
		defer func() { http.DefaultTransport = orig }()
		http.DefaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(""))}, nil
		})

		var buf bytes.Buffer
		svc, _ := newMockService("token-123", nil)
		if err := svc.DownloadAttachment(context.Background(), "https://files.example/attachment.csv", &buf); err == nil {
			t.Fatal("DownloadAttachment() error = nil, want failure")
		}
	})

	t.Run("upload account balance history", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != "POST" {
				t.Fatalf("unexpected method: %s", req.Method)
			}
			if req.Header.Get("Client-Platform") != "web" || req.Header.Get("Authorization") != "Token tok" {
				t.Fatalf("unexpected headers: %#v", req.Header)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}, nil
		})

		tmp := filepath.Join(t.TempDir(), "sample.csv")
		if err := os.WriteFile(tmp, []byte("a,b\n1,2\n"), 0600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		file, err := os.Open(tmp)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer file.Close()

		svc := NewService(&mockClient{token: "tok"})
		if err := svc.UploadAccountBalanceHistory(context.Background(), "acc-1", file); err != nil {
			t.Fatalf("UploadAccountBalanceHistory() error = %v", err)
		}
	})

	t.Run("upload account balance history non-200", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader(""))}, nil
		})

		tmp := filepath.Join(t.TempDir(), "sample.csv")
		if err := os.WriteFile(tmp, []byte("date,amount\n"), 0600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		file, err := os.Open(tmp)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer file.Close()

		svc := NewService(&mockClient{token: "tok"})
		if err := svc.UploadAccountBalanceHistory(context.Background(), "acc-1", file); err == nil {
			t.Fatal("UploadAccountBalanceHistory() error = nil, want failure")
		}
	})

	t.Run("upload account balance history network error", func(t *testing.T) {
		origTransport := http.DefaultTransport
		defer func() { http.DefaultTransport = origTransport }()
		http.DefaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("network down")
		})

		tmp := filepath.Join(t.TempDir(), "sample.csv")
		if err := os.WriteFile(tmp, []byte("date,amount\n"), 0600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		file, err := os.Open(tmp)
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		defer file.Close()

		svc := NewService(&mockClient{token: "tok"})
		if err := svc.UploadAccountBalanceHistory(context.Background(), "acc-1", file); err == nil {
			t.Fatal("UploadAccountBalanceHistory() error = nil, want failure")
		}
	})

	t.Run("upload account balance history request error", func(t *testing.T) {
		original := newBalanceHistoryRequest
		newBalanceHistoryRequest = func(context.Context, string, string, io.Reader) (*http.Request, error) {
			return nil, errors.New("request failed")
		}
		defer func() { newBalanceHistoryRequest = original }()

		svc := NewService(&mockClient{token: "tok"})
		if err := svc.UploadAccountBalanceHistory(context.Background(), "acc-1", strings.NewReader("date,amount\n")); err == nil {
			t.Fatal("UploadAccountBalanceHistory() error = nil, want failure")
		}
	})

	t.Run("upload account balance history read error", func(t *testing.T) {
		svc := NewService(&mockClient{token: "tok"})
		if err := svc.UploadAccountBalanceHistory(context.Background(), "acc-1", failingReader{}); err == nil {
			t.Fatal("UploadAccountBalanceHistory() error = nil, want failure")
		}
	})

	t.Run("upload account balance history form file error", func(t *testing.T) {
		original := createBalanceHistoryFormFile
		createBalanceHistoryFormFile = func(*multipart.Writer, string, string) (io.Writer, error) {
			return nil, errors.New("form file failed")
		}
		defer func() { createBalanceHistoryFormFile = original }()

		svc := NewService(&mockClient{token: "tok"})
		if err := svc.UploadAccountBalanceHistory(context.Background(), "acc-1", strings.NewReader("date,amount\n")); err == nil {
			t.Fatal("UploadAccountBalanceHistory() error = nil, want failure")
		}
	})

	t.Run("list transaction attachments unavailable", func(t *testing.T) {
		got, err := NewService(&mockClient{token: "tok"}).ListTransactionAttachments(context.Background(), "tx-1")
		if err == nil || !strings.Contains(err.Error(), "FEATURE_UNAVAILABLE") {
			t.Fatalf("ListTransactionAttachments() error = %v, want feature unavailable", err)
		}
		if got != nil {
			t.Fatalf("ListTransactionAttachments() = %#v, want nil", got)
		}
	})

	t.Run("upload attachment unavailable", func(t *testing.T) {
		tmp := filepath.Join(t.TempDir(), "receipt.pdf")
		if err := os.WriteFile(tmp, []byte("pdf"), 0600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		err := NewService(&mockClient{token: "tok"}).UploadAttachment(context.Background(), "tx-1", tmp)
		if err == nil || !strings.Contains(err.Error(), "FEATURE_UNAVAILABLE") {
			t.Fatalf("UploadAttachment() error = %v, want feature unavailable", err)
		}
	})
}
