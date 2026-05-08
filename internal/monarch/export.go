package monarch

import (
	"encoding/csv"
	"fmt"
	"io"
)

func ExportTransactionsCSV(txs []Transaction, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	header := []string{"Date", "Merchant", "Category", "Amount", "Notes"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, t := range txs {
		row := []string{
			t.Date,
			t.Merchant,
			t.Category,
			fmt.Sprintf("%.2f", t.Amount),
			t.Notes,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
