package monarch

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

type csvWriter interface {
	Write(record []string) error
	Flush()
	Error() error
}

var newCSVWriter = func(w io.Writer) csvWriter {
	return csv.NewWriter(w)
}

func ExportTransactionsCSV(txs []Transaction, w io.Writer) error {
	writer := newCSVWriter(w)

	header := []string{
		"Date", "ID", "AccountID", "Merchant", "PlaidName", "ProviderDescription",
		"Category", "CategoryGroup", "CategoryGroupType", "Amount", "Notes", "Tags",
		"Goal", "Pending", "HideFromReports", "IsRecurring", "ReviewStatus", "NeedsReview", "Splits",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	for i := range txs {
		t := &txs[i]
		tags := make([]string, 0, len(t.Tags))
		for _, tag := range t.Tags {
			tags = append(tags, tag.Name)
		}
		splits := make([]string, 0, len(t.Splits))
		for _, sp := range t.Splits {
			splits = append(splits, fmt.Sprintf("%s|%s|%.2f|%s", sp.Merchant, sp.Category, sp.Amount, sp.Notes))
		}
		row := []string{
			t.Date,
			t.ID,
			t.AccountID,
			t.Merchant,
			t.PlaidName,
			t.DataProviderDescription,
			t.Category,
			t.CategoryGroup.Name,
			t.CategoryGroup.Type,
			fmt.Sprintf("%.2f", t.Amount),
			t.Notes,
			strings.Join(tags, "; "),
			t.Goal.Name,
			formatBool(t.Pending),
			formatBool(t.HideFromReports),
			formatBool(t.IsRecurring),
			t.ReviewStatus,
			formatBool(t.NeedsReview),
			strings.Join(splits, " ;; "),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}

	return nil
}

func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
