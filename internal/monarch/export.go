package monarch

import (
	"encoding/csv"
	"fmt"
	"io"
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

	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}

	return nil
}
