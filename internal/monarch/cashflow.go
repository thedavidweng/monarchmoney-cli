package monarch

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetCashflowSummaryQuery = queries.Get("cashflow/summary.graphql")
var GetCashflowCategoriesQuery = queries.Get("cashflow/categories.graphql")
var GetCashflowMerchantsQuery = queries.Get("cashflow/merchants.graphql")
var GetCashflowTrendsQuery = queries.Get("cashflow/trends.graphql")

type CashflowSummary struct {
	Income      float64 `json:"income"`
	Expense     float64 `json:"expense"`
	Savings     float64 `json:"savings"`
	SavingsRate float64 `json:"savings_rate"`
}

type CashflowRecord struct {
	Name   string  `json:"name"`
	Amount float64 `json:"amount"`
}

type CashflowPeriod struct {
	Period  string  `json:"period"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Savings float64 `json:"savings"`
}

type CashflowTrendOptions struct {
	StartDate   string
	EndDate     string
	GroupBy     string
	Period      string
	AccountIDs  []string
	CategoryIDs []string
}

type CashflowTrendRow struct {
	GroupID    string  `json:"group_id"`
	GroupName  string  `json:"group_name"`
	GroupType  string  `json:"group_type"`
	Period     string  `json:"period"`
	Sum        float64 `json:"sum"`
	SumIncome  float64 `json:"sum_income"`
	SumExpense float64 `json:"sum_expense"`
}

func (s *Service) ListCashflow(ctx context.Context, startDate, endDate string) ([]CashflowPeriod, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, errors.New(errors.InvalidArguments, "invalid cashflow start date", errors.CatValidation, false, err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, errors.New(errors.InvalidArguments, "invalid cashflow end date", errors.CatValidation, false, err)
	}

	if end.Before(start) {
		start, end = end, start
	}
	startDate = start.Format("2006-01-02")
	endDate = end.Format("2006-01-02")

	periods := make(map[string]*CashflowPeriod)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		periodKey := day.Format("2006-01-02")
		periods[periodKey] = &CashflowPeriod{Period: periodKey}
	}

	const pageSize = 1000
	for offset := 0; ; offset += pageSize {
		page, total, err := s.ListTransactions(ctx, ListTransactionsOptions{
			Limit:     pageSize,
			Offset:    offset,
			StartDate: startDate,
			EndDate:   endDate,
		})
		if err != nil {
			return nil, err
		}

		for _, tx := range page {
			period, ok := periods[tx.Date]
			if !ok {
				continue
			}
			if tx.Amount >= 0 {
				period.Income += tx.Amount
			} else {
				period.Expense += tx.Amount
			}
			period.Savings = period.Income + period.Expense
		}

		if len(page) == 0 || offset+len(page) >= total {
			break
		}
	}

	keys := make([]string, 0, len(periods))
	for key := range periods {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]CashflowPeriod, 0, len(keys))
	for _, key := range keys {
		out = append(out, *periods[key])
	}

	return out, nil
}

func (s *Service) GetCashflowSummary(ctx context.Context, startDate, endDate string) (*CashflowSummary, error) {
	var resp struct {
		Aggregates []struct {
			Summary struct {
				SumIncome   float64 `json:"sumIncome"`
				SumExpense  float64 `json:"sumExpense"`
				Savings     float64 `json:"savings"`
				SavingsRate float64 `json:"savingsRate"`
			} `json:"summary"`
		} `json:"aggregates"`
	}

	variables := map[string]any{
		"filters": map[string]any{
			"startDate":  startDate,
			"endDate":    endDate,
			"search":     "",
			"categories": []string{},
			"accounts":   []string{},
			"tags":       []string{},
		},
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCashflowSummary",
		Query:         GetCashflowSummaryQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	if len(resp.Aggregates) == 0 {
		return &CashflowSummary{}, nil
	}

	return &CashflowSummary{
		Income:      resp.Aggregates[0].Summary.SumIncome,
		Expense:     resp.Aggregates[0].Summary.SumExpense,
		Savings:     resp.Aggregates[0].Summary.Savings,
		SavingsRate: resp.Aggregates[0].Summary.SavingsRate,
	}, nil
}

func (s *Service) GetCashflowCategories(ctx context.Context, startDate, endDate string) ([]CashflowRecord, error) {
	var resp struct {
		Aggregates []struct {
			GroupBy struct {
				Category struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"category"`
			} `json:"groupBy"`
			Summary struct {
				Sum float64 `json:"sum"`
			} `json:"summary"`
		} `json:"aggregates"`
	}

	variables := map[string]any{
		"filters": map[string]any{
			"startDate":  startDate,
			"endDate":    endDate,
			"search":     "",
			"categories": []string{},
			"accounts":   []string{},
			"tags":       []string{},
		},
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCashflowCategories",
		Query:         GetCashflowCategoriesQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	records := make([]CashflowRecord, 0, len(resp.Aggregates))
	for _, a := range resp.Aggregates {
		if a.GroupBy.Category.Name != "" {
			records = append(records, CashflowRecord{
				Name:   a.GroupBy.Category.Name,
				Amount: a.Summary.Sum,
			})
		}
	}

	return records, nil
}

func (s *Service) GetCashflowMerchants(ctx context.Context, startDate, endDate string) ([]CashflowRecord, error) {
	var resp struct {
		Aggregates []struct {
			GroupBy struct {
				Merchant struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"merchant"`
			} `json:"groupBy"`
			Summary struct {
				SumIncome  float64 `json:"sumIncome"`
				SumExpense float64 `json:"sumExpense"`
			} `json:"summary"`
		} `json:"aggregates"`
	}

	variables := map[string]any{
		"filters": map[string]any{
			"startDate":  startDate,
			"endDate":    endDate,
			"search":     "",
			"categories": []string{},
			"accounts":   []string{},
			"tags":       []string{},
		},
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCashflowMerchants",
		Query:         GetCashflowMerchantsQuery,
		Variables:     variables,
	}, &resp)

	if err != nil {
		return nil, err
	}

	records := make([]CashflowRecord, 0, len(resp.Aggregates))
	for _, a := range resp.Aggregates {
		if a.GroupBy.Merchant.Name != "" {
			records = append(records, CashflowRecord{
				Name:   a.GroupBy.Merchant.Name,
				Amount: a.Summary.SumExpense,
			})
		}
	}

	return records, nil
}

func (s *Service) GetCashflowTrends(ctx context.Context, opts CashflowTrendOptions) ([]CashflowTrendRow, error) {
	groupBy, err := monarchAggregateGroupBy(opts.GroupBy)
	if err != nil {
		return nil, err
	}
	period, err := monarchAggregatePeriod(opts.Period)
	if err != nil {
		return nil, err
	}
	operationName := "GetAggregatesGraph"
	if groupBy == "categoryGroup" {
		operationName = "GetAggregatesGraphCategoryGroup"
	}

	var resp struct {
		Aggregates []struct {
			GroupBy map[string]json.RawMessage `json:"groupBy"`
			Summary struct {
				Sum        float64 `json:"sum"`
				SumIncome  float64 `json:"sumIncome"`
				SumExpense float64 `json:"sumExpense"`
			} `json:"summary"`
		} `json:"aggregates"`
	}

	filters := map[string]any{
		"startDate": opts.StartDate,
		"endDate":   opts.EndDate,
	}
	if len(opts.CategoryIDs) > 0 {
		filters["categories"] = opts.CategoryIDs
	}
	if len(opts.AccountIDs) > 0 {
		filters["accounts"] = opts.AccountIDs
	}

	err = s.Client.Do(ctx, &graphql.Request{
		OperationName: operationName,
		Query:         cashflowTrendsQuery(groupBy, period),
		Variables:     map[string]any{"filters": filters},
	}, &resp)
	if err != nil {
		return nil, err
	}

	rows := make([]CashflowTrendRow, 0, len(resp.Aggregates))
	for _, aggregate := range resp.Aggregates {
		row := CashflowTrendRow{
			Sum:        aggregate.Summary.Sum,
			SumIncome:  aggregate.Summary.SumIncome,
			SumExpense: aggregate.Summary.SumExpense,
		}
		if periodRaw := aggregate.GroupBy[period]; len(periodRaw) > 0 {
			_ = json.Unmarshal(periodRaw, &row.Period)
		}
		if opts.GroupBy == "category-group" {
			row.GroupID = cashflowTrendGroupID(aggregate.GroupBy["categoryGroup"])
		} else {
			row.GroupID = cashflowTrendGroupID(aggregate.GroupBy["category"])
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func monarchAggregateGroupBy(value string) (string, error) {
	switch value {
	case "category":
		return "category", nil
	case "category-group":
		return "categoryGroup", nil
	default:
		return "", errors.New(errors.InvalidArguments, "group-by must be category or category-group", errors.CatValidation, false, nil)
	}
}

func monarchAggregatePeriod(value string) (string, error) {
	switch value {
	case "month", "quarter", "year":
		return value, nil
	default:
		return "", errors.New(errors.InvalidArguments, "period must be month, quarter, or year", errors.CatValidation, false, nil)
	}
}

func cashflowTrendsQuery(groupBy, period string) string {
	if groupBy == "category" && period == "month" {
		return GetCashflowTrendsQuery
	}

	operationName := "GetAggregatesGraph"
	groupSelection := `category {
        id
      }`
	if groupBy == "categoryGroup" {
		operationName = "GetAggregatesGraphCategoryGroup"
		groupSelection = `categoryGroup {
        id
      }`
	}
	return fmt.Sprintf(`query %s($filters: TransactionFilterInput) {
  aggregates(filters: $filters, groupBy: [%q, %q], fillEmptyValues: false) {
    groupBy {
      %s
      %s
    }
    summary {
      sum
    }
  }
}`, operationName, groupBy, period, groupSelection, period)
}

func cashflowTrendGroupID(raw json.RawMessage) string {
	var group struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(raw, &group)
	return group.ID
}
