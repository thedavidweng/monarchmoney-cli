package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/monarchmoney-cli/monarch/internal/audit"
	"github.com/monarchmoney-cli/monarch/internal/auth"
	"github.com/monarchmoney-cli/monarch/internal/config"
	"github.com/monarchmoney-cli/monarch/internal/errors"
	"github.com/monarchmoney-cli/monarch/internal/graphql"
	"github.com/monarchmoney-cli/monarch/internal/monarch"
	"github.com/monarchmoney-cli/monarch/internal/output"
	"github.com/monarchmoney-cli/monarch/internal/safety"
	"github.com/spf13/cobra"
)

var (
	limit        int
	offset       int
	format       string
	outputFile   string
	txNotes      string
	txCategoryID string
	attachmentID string
	startDate    string
	endDate      string
)

var transactionsCmd = &cobra.Command{
	Use:   "transactions",
	Short: "Manage Monarch Money transactions",
}

var transactionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List transactions",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "transactions.list", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		txs, total, err := svc.ListTransactions(cmd.Context(), monarch.ListTransactionsOptions{
			Limit:     limit,
			Offset:    offset,
			StartDate: startDate,
			EndDate:   endDate,
		})
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list transactions", errors.CatAPI, false, err)
			}
			handleError(renderer, "transactions.list", cliErr, start)
			return
		}

		if jsonMode {
			data := map[string]interface{}{
				"transactions": txs,
				"total":        total,
			}
			env := output.NewEnvelope("transactions.list", profile, "2026-05-08", "", data, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-12s %-20s %-15s %10s %s\n", "DATE", "MERCHANT", "CATEGORY", "AMOUNT", "NOTES")
			for _, t := range txs {
				fmt.Printf("%-12s %-20s %-15s %10.2f %s\n", t.Date, t.Merchant, t.Category, t.Amount, t.Notes)
			}
			fmt.Printf("\nTotal transactions: %d\n", total)
		}
	},
}

var transactionsSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search transactions",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "transactions.search", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		txs, total, err := svc.ListTransactions(cmd.Context(), monarch.ListTransactionsOptions{
			Limit:     limit,
			Offset:    offset,
			Search:    args[0],
			StartDate: startDate,
			EndDate:   endDate,
		})
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to search transactions", errors.CatAPI, false, err)
			}
			handleError(renderer, "transactions.search", cliErr, start)
			return
		}

		if jsonMode {
			data := map[string]interface{}{
				"transactions": txs,
				"total":        total,
			}
			env := output.NewEnvelope("transactions.search", profile, "2026-05-08", "", data, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-12s %-20s %-15s %10s %s\n", "DATE", "MERCHANT", "CATEGORY", "AMOUNT", "NOTES")
			for _, t := range txs {
				fmt.Printf("%-12s %-20s %-15s %10.2f %s\n", t.Date, t.Merchant, t.Category, t.Amount, t.Notes)
			}
			fmt.Printf("\nTotal matches: %d\n", total)
		}
	},
}

var transactionsDuplicatesCmd = &cobra.Command{
	Use:   "duplicates",
	Short: "Find duplicate transactions",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "transactions.duplicates", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		txs, err := svc.GetDuplicateTransactions(cmd.Context())
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to find duplicates", errors.CatAPI, false, err)
			}
			handleError(renderer, "transactions.duplicates", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.duplicates", profile, "2026-05-08", "", txs, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-12s %-20s %10s %s\n", "DATE", "MERCHANT", "AMOUNT", "ID")
			for _, t := range txs {
				fmt.Printf("%-12s %-20s %10.2f %s\n", t.Date, t.Merchant, t.Amount, t.ID)
			}
		}
	},
}

var transactionsSplitsCmd = &cobra.Command{
	Use:   "splits <transaction-id>",
	Short: "Get splits for a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "transactions.splits", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		splits, err := svc.GetTransactionSplits(cmd.Context(), args[0])
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to get splits", errors.CatAPI, false, err)
			}
			handleError(renderer, "transactions.splits", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.splits", profile, "2026-05-08", "", splits, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-20s %10s %s\n", "CATEGORY", "AMOUNT", "NOTES")
			for _, s := range splits {
				fmt.Printf("%-20s %10.2f %s\n", s.Category, s.Amount, s.Notes)
			}
		}
	},
}

var transactionsUpdateCmd = &cobra.Command{
	Use:   "update <transaction-id>",
	Short: "Update a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		logger := audit.NewLogger()
		id := args[0]

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "transactions.update", err.(*errors.Error), start)
			return
		}

		var notes *string
		if cmd.Flags().Changed("notes") {
			notes = &txNotes
		}
		var categoryID *string
		if cmd.Flags().Changed("category") {
			categoryID = &txCategoryID
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("transactions.update", id, nil, map[string]interface{}{"notes": notes, "categoryId": categoryID})
			env := output.NewEnvelope("transactions.update", profile, "2026-05-08", "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "transactions.update", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		tx, err := svc.UpdateTransaction(cmd.Context(), id, notes, categoryID)
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{
			Command:    "transactions.update",
			ResourceID: id,
			DryRun:     dryRun,
			Confirmed:  confirm,
			Profile:    profile,
			Result:     result,
			ErrorCode:  errCode,
		})

		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to update transaction", errors.CatAPI, false, err)
			}
			handleError(renderer, "transactions.update", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.update", profile, "2026-05-08", "", tx, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully updated transaction %s.\n", tx.ID)
		}
	},
}

var transactionsDeleteCmd = &cobra.Command{
	Use:   "delete <transaction-id>",
	Short: "Delete a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		logger := audit.NewLogger()
		id := args[0]

		if err := safety.Check(safety.TierDestructive, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "transactions.delete", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("transactions.delete", id, nil, nil)
			env := output.NewEnvelope("transactions.delete", profile, "2026-05-08", "", plan, time.Since(start))
			renderer.RenderSuccess(env)
			return
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "transactions.delete", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		err = svc.DeleteTransaction(cmd.Context(), id)
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{
			Command:    "transactions.delete",
			ResourceID: id,
			DryRun:     dryRun,
			Confirmed:  confirm,
			Profile:    profile,
			Result:     result,
			ErrorCode:  errCode,
		})

		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to delete transaction", errors.CatAPI, false, err)
			}
			handleError(renderer, "transactions.delete", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.delete", profile, "2026-05-08", "", map[string]string{"status": "deleted"}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Successfully deleted transaction %s.\n", id)
		}
	},
}

var transactionsExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export transactions",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "transactions.export", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		txs, _, err := svc.ListTransactions(cmd.Context(), monarch.ListTransactionsOptions{
			Limit:     limit,
			Offset:    offset,
			StartDate: startDate,
			EndDate:   endDate,
		})
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list transactions", errors.CatAPI, false, err)
			}
			handleError(renderer, "transactions.export", cliErr, start)
			return
		}

		var out io.Writer = os.Stdout
		if outputFile != "" {
			f, err := os.Create(outputFile)
			if err != nil {
				handleError(renderer, "transactions.export", errors.New(errors.InternalError, "failed to create output file", errors.CatInternal, false, err), start)
				return
			}
			defer f.Close()
			out = f
		}

		if format == "csv" {
			if err := monarch.ExportTransactionsCSV(txs, out); err != nil {
				handleError(renderer, "transactions.export", errors.New(errors.InternalError, "failed to export CSV", errors.CatInternal, false, err), start)
				return
			}
		} else {
			// Default to JSON
			env := output.NewEnvelope("transactions.export", profile, "2026-05-08", "", txs, time.Since(start))
			renderer.RenderSuccess(env)
		}
	},
}

var transactionsAttachmentsCmd = &cobra.Command{
	Use:   "attachments",
	Short: "Manage transaction attachments",
}

var transactionsAttachmentsListCmd = &cobra.Command{
	Use:   "list <transaction-id>",
	Short: "List attachments for a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "transactions.attachments.list", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		atts, err := svc.ListTransactionAttachments(cmd.Context(), args[0])
		if err != nil {
			var cliErr *errors.Error
			if e, ok := err.(*errors.Error); ok {
				cliErr = e
			} else {
				cliErr = errors.New(errors.APIError, "failed to list attachments", errors.CatAPI, false, err)
			}
			handleError(renderer, "transactions.attachments.list", cliErr, start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.attachments.list", profile, "2026-05-08", "", atts, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("%-20s %-30s %s\n", "ID", "FILE NAME", "CREATED AT")
			for _, a := range atts {
				fmt.Printf("%-20s %-30s %s\n", a.ID, a.FileName, a.CreatedAt)
			}
		}
	},
}

var transactionsAttachmentsDownloadCmd = &cobra.Command{
	Use:   "download <transaction-id>",
	Short: "Download an attachment for a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if attachmentID == "" {
			handleError(renderer, "transactions.attachments.download", errors.New(errors.InvalidArguments, "--id is required", errors.CatValidation, false, nil), start)
			return
		}

		store := auth.NewStore(config.DefaultSessionPath())
		sess, err := store.Load()
		if err != nil {
			handleError(renderer, "transactions.attachments.download", errors.New(errors.AuthRequired, "not logged in", errors.CatAuth, false, err), start)
			return
		}

		client := graphql.NewClient("https://api.monarch.com/graphql", sess.Token, timeout)
		svc := monarch.NewService(client)

		atts, err := svc.ListTransactionAttachments(cmd.Context(), args[0])
		if err != nil {
			handleError(renderer, "transactions.attachments.download", errors.New(errors.APIError, "failed to find attachment", errors.CatAPI, false, err), start)
			return
		}

		var target *monarch.Attachment
		for _, a := range atts {
			if a.ID == attachmentID {
				target = &a
				break
			}
		}

		if target == nil {
			handleError(renderer, "transactions.attachments.download", errors.New(errors.ResourceNotFound, "attachment not found", errors.CatAPI, false, nil), start)
			return
		}

		path := outputFile
		if path == "" {
			path = target.FileName
		}

		f, err := os.Create(path)
		if err != nil {
			handleError(renderer, "transactions.attachments.download", errors.New(errors.InternalError, "failed to create local file", errors.CatInternal, false, err), start)
			return
		}
		defer f.Close()

		if err := svc.DownloadAttachment(cmd.Context(), target.URL, f); err != nil {
			handleError(renderer, "transactions.attachments.download", err.(*errors.Error), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.attachments.download", profile, "2026-05-08", "", map[string]string{"path": path}, time.Since(start))
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Downloaded attachment to %s\n", path)
		}
	},
}

func init() {
	transactionsCmd.PersistentFlags().StringVar(&startDate, "from", "", "start date (YYYY-MM-DD)")
	transactionsCmd.PersistentFlags().StringVar(&endDate, "to", "", "end date (YYYY-MM-DD)")

	transactionsListCmd.Flags().IntVar(&limit, "limit", 100, "maximum number of transactions to return")
	transactionsListCmd.Flags().IntVar(&offset, "offset", 0, "number of transactions to skip")

	transactionsSearchCmd.Flags().IntVar(&limit, "limit", 100, "maximum number of transactions to return")
	transactionsSearchCmd.Flags().IntVar(&offset, "offset", 0, "number of transactions to skip")

	transactionsExportCmd.Flags().IntVar(&limit, "limit", 1000, "maximum number of transactions to export")
	transactionsExportCmd.Flags().IntVar(&offset, "offset", 0, "number of transactions to skip")
	transactionsExportCmd.Flags().StringVar(&format, "format", "json", "export format (json or csv)")
	transactionsExportCmd.Flags().StringVar(&outputFile, "output", "", "output file path")

	transactionsUpdateCmd.Flags().StringVar(&txNotes, "notes", "", "transaction notes")
	transactionsUpdateCmd.Flags().StringVar(&txCategoryID, "category", "", "transaction category ID")

	transactionsAttachmentsDownloadCmd.Flags().StringVar(&attachmentID, "id", "", "attachment ID")
	transactionsAttachmentsDownloadCmd.Flags().StringVar(&outputFile, "output", "", "output file path")

	transactionsAttachmentsCmd.AddCommand(transactionsAttachmentsListCmd)
	transactionsAttachmentsCmd.AddCommand(transactionsAttachmentsDownloadCmd)
	transactionsCmd.AddCommand(transactionsAttachmentsCmd)
	transactionsCmd.AddCommand(transactionsListCmd)
	transactionsCmd.AddCommand(transactionsSearchCmd)
	transactionsCmd.AddCommand(transactionsDuplicatesCmd)
	transactionsCmd.AddCommand(transactionsSplitsCmd)
	transactionsCmd.AddCommand(transactionsExportCmd)
	transactionsCmd.AddCommand(transactionsUpdateCmd)
	transactionsCmd.AddCommand(transactionsDeleteCmd)
	RootCmd.AddCommand(transactionsCmd)
}
