package monarch

import (
	"context"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetReportsDataQuery = queries.Get("reports/data.graphql")
var ListSavedReportsQuery = queries.Get("reports/list.graphql")
var CreateSavedReportMutation = queries.Get("reports/create.graphql")
var UpdateSavedReportMutation = queries.Get("reports/update.graphql")
var DeleteSavedReportMutation = queries.Get("reports/delete.graphql")

type ReportSummary struct {
	Sum         *float64 `json:"sum,omitempty"`
	Avg         *float64 `json:"avg,omitempty"`
	Count       int      `json:"count"`
	Max         *float64 `json:"max,omitempty"`
	SumIncome   *float64 `json:"sum_income,omitempty"`
	SumExpense  *float64 `json:"sum_expense,omitempty"`
	Savings     *float64 `json:"savings,omitempty"`
	SavingsRate *float64 `json:"savings_rate,omitempty"`
	First       string   `json:"first,omitempty"`
	Last        string   `json:"last,omitempty"`
}

type ReportRow struct {
	Date          string        `json:"date,omitempty"`
	Category      string        `json:"category,omitempty"`
	CategoryGroup string        `json:"category_group,omitempty"`
	Merchant      string        `json:"merchant,omitempty"`
	Summary       ReportSummary `json:"summary"`
}

type ReportResult struct {
	Summary ReportSummary `json:"summary"`
	Rows    []ReportRow   `json:"rows"`
}

type SavedReport struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	Timeframe   string   `json:"timeframe,omitempty"`
	ChartType   string   `json:"chart_type,omitempty"`
	Dimensions  []string `json:"dimensions,omitempty"`
}

type ReportDataOptions struct {
	StartDate string
	EndDate   string
	Search    string
	GroupBy   string
	Timeframe string
	SortBy    string
}

type rawReportSummary struct {
	Sum         *float64 `json:"sum"`
	Avg         *float64 `json:"avg"`
	Count       int      `json:"count"`
	Max         *float64 `json:"max"`
	SumIncome   *float64 `json:"sumIncome"`
	SumExpense  *float64 `json:"sumExpense"`
	Savings     *float64 `json:"savings"`
	SavingsRate *float64 `json:"savingsRate"`
	First       string   `json:"first"`
	Last        string   `json:"last"`
}

func toReportSummary(r *rawReportSummary) ReportSummary {
	if r == nil {
		return ReportSummary{}
	}
	return ReportSummary{
		Sum: r.Sum, Avg: r.Avg, Count: r.Count, Max: r.Max,
		SumIncome: r.SumIncome, SumExpense: r.SumExpense,
		Savings: r.Savings, SavingsRate: r.SavingsRate,
		First: r.First, Last: r.Last,
	}
}

func reportGroupByValue(groupBy string) (string, error) {
	switch groupBy {
	case "", "none":
		return "", nil
	case "category", "category-group", "merchant":
		if groupBy == "category-group" {
			return "category_group", nil
		}
		return groupBy, nil
	default:
		return "", errors.New(errors.InvalidArguments, "group-by must be category, category-group, merchant, or none", errors.CatValidation, false, nil)
	}
}

func (s *Service) GetReportData(ctx context.Context, opts *ReportDataOptions) (*ReportResult, error) {
	group, err := reportGroupByValue(opts.GroupBy)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Reports []struct {
			GroupBy struct {
				Date     string `json:"date"`
				Category *struct {
					Name string `json:"name"`
				} `json:"category"`
				CategoryGroup *struct {
					Name string `json:"name"`
				} `json:"categoryGroup"`
				Merchant *struct {
					Name string `json:"name"`
				} `json:"merchant"`
			} `json:"groupBy"`
			Summary *rawReportSummary `json:"summary"`
		} `json:"reports"`
		Aggregates []struct {
			Summary *rawReportSummary `json:"summary"`
		} `json:"aggregates"`
	}

	filters := map[string]any{
		"search":     opts.Search,
		"categories": []string{},
		"accounts":   []string{},
		"tags":       []string{},
	}
	if opts.StartDate != "" {
		filters["startDate"] = opts.StartDate
	}
	if opts.EndDate != "" {
		filters["endDate"] = opts.EndDate
	}

	variables := map[string]any{
		"filters":         filters,
		"fillEmptyValues": true,
	}
	var groups []string
	if group != "" {
		groups = []string{group}
		variables["groupBy"] = groups
	}
	if opts.Timeframe != "" {
		variables["groupByTimeframe"] = opts.Timeframe
	}
	if opts.SortBy != "" {
		variables["sortBy"] = opts.SortBy
	}
	variables["includeCategory"] = group == "category"
	variables["includeCategoryGroup"] = group == "category_group"
	variables["includeMerchant"] = group == "merchant"

	err = s.Client.Do(ctx, &graphql.Request{
		OperationName: "Common_GetReportsData",
		Query:         GetReportsDataQuery,
		Variables:     variables,
	}, &resp)
	if err != nil {
		return nil, err
	}

	result := &ReportResult{}
	if len(resp.Aggregates) > 0 {
		result.Summary = toReportSummary(resp.Aggregates[0].Summary)
	}
	for _, row := range resp.Reports {
		out := ReportRow{Summary: toReportSummary(row.Summary)}
		out.Date = row.GroupBy.Date
		if row.GroupBy.Category != nil {
			out.Category = row.GroupBy.Category.Name
		}
		if row.GroupBy.CategoryGroup != nil {
			out.CategoryGroup = row.GroupBy.CategoryGroup.Name
		}
		if row.GroupBy.Merchant != nil {
			out.Merchant = row.GroupBy.Merchant.Name
		}
		result.Rows = append(result.Rows, out)
	}
	return result, nil
}

func (s *Service) ListSavedReports(ctx context.Context) ([]SavedReport, error) {
	var resp struct {
		ReportConfigurations []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			ReportView  *struct {
				Timeframe  string   `json:"timeframe"`
				ChartType  string   `json:"chartType"`
				Dimensions []string `json:"dimensions"`
			} `json:"reportView"`
		} `json:"reportConfigurations"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Web_GetReportConfigurations",
		Query:         ListSavedReportsQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}

	out := make([]SavedReport, 0, len(resp.ReportConfigurations))
	for _, r := range resp.ReportConfigurations {
		report := SavedReport{ID: r.ID, DisplayName: r.DisplayName}
		if r.ReportView != nil {
			report.Timeframe = r.ReportView.Timeframe
			report.ChartType = r.ReportView.ChartType
			report.Dimensions = r.ReportView.Dimensions
		}
		out = append(out, report)
	}
	return out, nil
}

func (s *Service) GetSavedReport(ctx context.Context, id string) (*SavedReport, error) {
	reports, err := s.ListSavedReports(ctx)
	if err != nil {
		return nil, err
	}
	for i := range reports {
		if reports[i].ID == id {
			return &reports[i], nil
		}
	}
	return nil, errors.New(errors.ResourceNotFound, "saved report not found", errors.CatAPI, false, nil)
}

func (s *Service) CreateSavedReport(ctx context.Context, name, groupBy, timeframe string) (*SavedReport, error) {
	group, err := reportGroupByValue(groupBy)
	if err != nil {
		return nil, err
	}
	view := map[string]any{}
	if group != "" {
		view["dimensions"] = []string{group}
	}
	if timeframe != "" {
		view["timeframe"] = timeframe
	}

	var resp struct {
		CreateReportConfiguration struct {
			ReportConfiguration *struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"reportConfiguration"`
			Errors []payloadError `json:"errors"`
		} `json:"createReportConfiguration"`
	}

	err = s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Web_CreateReportConfiguration",
		Query:         CreateSavedReportMutation,
		Variables: map[string]any{
			"input": map[string]any{
				"displayName":        name,
				"transactionFilters": map[string]any{},
				"reportView":         view,
			},
		},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.CreateReportConfiguration.Errors, "failed to create saved report"); apiErr != nil {
		return nil, apiErr
	}
	if resp.CreateReportConfiguration.ReportConfiguration == nil {
		return nil, errors.New(errors.APISchemaChanged, "saved report creation response missing reportConfiguration", errors.CatAPI, false, nil)
	}
	r := resp.CreateReportConfiguration.ReportConfiguration
	return &SavedReport{ID: r.ID, DisplayName: r.DisplayName}, nil
}

func (s *Service) UpdateSavedReport(ctx context.Context, id, name string) (*SavedReport, error) {
	var resp struct {
		UpdateReportConfiguration struct {
			ReportConfiguration *struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"reportConfiguration"`
			Errors []payloadError `json:"errors"`
		} `json:"updateReportConfiguration"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Web_UpdateReportConfiguration",
		Query:         UpdateSavedReportMutation,
		Variables: map[string]any{
			"input": map[string]any{
				"id":          id,
				"displayName": name,
			},
		},
	}, &resp)
	if err != nil {
		return nil, err
	}
	if apiErr := payloadErrorsToError(resp.UpdateReportConfiguration.Errors, "failed to update saved report"); apiErr != nil {
		return nil, apiErr
	}
	if resp.UpdateReportConfiguration.ReportConfiguration == nil {
		return nil, errors.New(errors.APISchemaChanged, "saved report update response missing reportConfiguration", errors.CatAPI, false, nil)
	}
	r := resp.UpdateReportConfiguration.ReportConfiguration
	return &SavedReport{ID: r.ID, DisplayName: r.DisplayName}, nil
}

func (s *Service) DeleteSavedReport(ctx context.Context, id string) error {
	var resp struct {
		DeleteReportConfiguration struct {
			Deleted bool           `json:"deleted"`
			Errors  []payloadError `json:"errors"`
		} `json:"deleteReportConfiguration"`
	}

	err := s.Client.DoMutation(ctx, &graphql.Request{
		OperationName: "Web_DeleteReportConfiguration",
		Query:         DeleteSavedReportMutation,
		Variables:     map[string]any{"id": id},
	}, &resp)
	if err != nil {
		return err
	}
	if apiErr := payloadErrorsToError(resp.DeleteReportConfiguration.Errors, "failed to delete saved report"); apiErr != nil {
		return apiErr
	}
	if !resp.DeleteReportConfiguration.Deleted {
		return errors.New(errors.APIError, "failed to delete saved report", errors.CatAPI, false, nil)
	}
	return nil
}
