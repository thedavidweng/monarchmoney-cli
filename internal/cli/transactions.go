package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/monarchmoney-cli/internal/audit"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
	"github.com/thedavidweng/monarchmoney-cli/internal/safety"
)

var jsonUnmarshal = json.Unmarshal

var (
	limit        int
	offset       int
	format       string
	outputFile   string
	txNotes      string
	txCategoryID string
	attachmentID string
	txStartDate  string
	txEndDate    string
	txAmount     float64
	txMerchant   string
	txDate       string
	txAccountID  string
	splitFile    string
	tagIDs       []string

	// Advanced transaction filters
	filterCategoryIDs []string
	filterAccountIDs  []string
	filterTagIDs      []string
	filterNeedsReview bool
	filterHasNotes    bool
	filterIsSplit     bool
	filterIsRecurring bool
	filterPending     bool
	filterHideReports bool
	filterGoalIDs     []string

	// Update transaction fields
	txHideFromReports bool
	txNeedsReview     bool
	txMarkReviewed    bool

	// Bulk categorize
	bulkTxIDs        []string
	bulkCategoryID   string
	bulkMarkReviewed bool
)

var transactionsCmd = &cobra.Command{
	Use:     "transactions",
	Short:   "Manage Monarch Money transactions",
	GroupID: "core",
	Example: "  monarch transactions list --limit 10 --json\n  monarch transactions search \"Amazon\"\n  monarch transactions update <id> --category cat_food --dry-run",
}

var transactionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List transactions",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "transactions.list", start)
		if !ok {
			return
		}
		svc := deps.Service

		opts := monarch.ListTransactionsOptions{
			Limit:       limit,
			Offset:      offset,
			StartDate:   txStartDate,
			EndDate:     txEndDate,
			CategoryIDs: filterCategoryIDs,
			AccountIDs:  filterAccountIDs,
			TagIDs:      filterTagIDs,
		}
		if cmd.Flags().Changed("needs-review") {
			opts.NeedsReview = &filterNeedsReview
		}
		if cmd.Flags().Changed("has-notes") {
			opts.HasNotes = &filterHasNotes
		}
		if cmd.Flags().Changed("is-split") {
			opts.IsSplit = &filterIsSplit
		}
		if cmd.Flags().Changed("is-recurring") {
			opts.IsRecurring = &filterIsRecurring
		}
		if cmd.Flags().Changed("pending") {
			opts.Pending = &filterPending
		}
		if cmd.Flags().Changed("hide-from-reports") {
			opts.HideFromReports = &filterHideReports
		}
		opts.GoalIDs = filterGoalIDs

		txs, total, err := svc.ListTransactions(cmd.Context(), opts)
		if err != nil {
			handleError(renderer, "transactions.list", wrapError(err, "failed to list transactions"), start)
			return
		}

		if jsonMode {
			data := map[string]any{
				"transactions": txs,
				"total":        total,
			}
			env := envelopeWithWarnings("transactions.list", data, start, "uses legacy Monarch GraphQL root field: allTransactions")
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
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

		deps, ok := newDeps(renderer, "transactions.search", start)
		if !ok {
			return
		}
		svc := deps.Service

		txs, total, err := svc.ListTransactions(cmd.Context(), monarch.ListTransactionsOptions{
			Limit:     limit,
			Offset:    offset,
			Search:    args[0],
			StartDate: txStartDate,
			EndDate:   txEndDate,
		})
		if err != nil {
			handleError(renderer, "transactions.search", wrapError(err, "failed to search transactions"), start)
			return
		}

		if jsonMode {
			data := map[string]any{
				"transactions": txs,
				"total":        total,
			}
			env := envelopeWithWarnings("transactions.search", data, start, "uses legacy Monarch GraphQL root field: allTransactions")
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
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

		deps, ok := newDeps(renderer, "transactions.duplicates", start)
		if !ok {
			return
		}
		svc := deps.Service

		// Default to current month for duplicate search
		now := time.Now()
		startDate := now.Format("2006-01-02")
		endDate := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Format("2006-01-02")

		txs, err := svc.GetDuplicateTransactions(cmd.Context(), startDate, endDate)
		if err != nil {
			handleError(renderer, "transactions.duplicates", wrapError(err, "failed to find duplicates"), start)
			return
		}

		if jsonMode {
			env := envelopeWithWarnings("transactions.duplicates", txs, start, "uses legacy Monarch GraphQL root field: allTransactions")
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
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

		deps, ok := newDeps(renderer, "transactions.splits", start)
		if !ok {
			return
		}
		svc := deps.Service

		splits, err := svc.GetTransactionSplits(cmd.Context(), args[0])
		if err != nil {
			handleError(renderer, "transactions.splits", wrapError(err, "failed to get splits"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.splits", profile, output.SchemaVersion, "", splits, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
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
		var amount *float64
		if cmd.Flags().Changed("amount") {
			amount = &txAmount
		}
		var date *string
		if cmd.Flags().Changed("date") {
			date = &txDate
		}
		var merchantName *string
		if cmd.Flags().Changed("merchant") {
			merchantName = &txMerchant
		}
		var hideFromReports *bool
		if cmd.Flags().Changed("hide-from-reports") {
			hideFromReports = &txHideFromReports
		}
		var needsReview *bool
		if cmd.Flags().Changed("needs-review") {
			needsReview = &txNeedsReview
		}
		if txMarkReviewed {
			f := false
			needsReview = &f
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("transactions.update", id, nil, map[string]any{"notes": notes, "categoryId": categoryID, "amount": amount, "date": date, "merchant": merchantName, "hideFromReports": hideFromReports, "needsReview": needsReview})
			env := output.NewEnvelope("transactions.update", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
			return
		}

		deps, ok := newDeps(renderer, "transactions.update", start)
		if !ok {
			return
		}

		result, err := deps.Mutate("transactions.update", id, func() (any, error) {
			return deps.Service.UpdateTransaction(cmd.Context(), id, notes, categoryID, amount, date, merchantName, hideFromReports, needsReview)
		}, "failed to update transaction")
		if err != nil {
			return
		}
		tx := result.(*monarch.Transaction)

		if jsonMode {
			env := output.NewEnvelope("transactions.update", profile, output.SchemaVersion, "", tx, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
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
		id := args[0]

		if err := safety.Check(safety.TierDestructive, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "transactions.delete", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("transactions.delete", id, nil, nil)
			env := output.NewEnvelope("transactions.delete", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
			return
		}

		deps, ok := newDeps(renderer, "transactions.delete", start)
		if !ok {
			return
		}

		if _, err := deps.Mutate("transactions.delete", id, func() (any, error) {
			return nil, deps.Service.DeleteTransaction(cmd.Context(), id)
		}, "failed to delete transaction"); err != nil {
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.delete", profile, output.SchemaVersion, "", map[string]string{"status": "deleted"}, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("Successfully deleted transaction %s.\n", id)
		}
	},
}

var transactionsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a transaction",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "transactions.create", err.(*errors.Error), start)
			return
		}

		if txDate == "" {
			txDate = time.Now().Format("2006-01-02")
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("transactions.create", "", nil, map[string]any{"amount": txAmount, "merchant": txMerchant, "date": txDate, "categoryId": txCategoryID})
			env := output.NewEnvelope("transactions.create", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
			return
		}

		deps, ok := newDeps(renderer, "transactions.create", start)
		if !ok {
			return
		}

		result, err := deps.Mutate("transactions.create", "", func() (any, error) {
			return deps.Service.CreateTransaction(cmd.Context(), txAmount, txMerchant, txDate, txCategoryID, txAccountID, txNotes)
		}, "failed to create transaction")
		if err != nil {
			return
		}
		tx := result.(*monarch.Transaction)

		if jsonMode {
			env := output.NewEnvelope("transactions.create", profile, output.SchemaVersion, "", tx, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("Successfully created transaction %s.\n", tx.ID)
		}
	},
}

var transactionsSplitCmd = &cobra.Command{
	Use:   "split <transaction-id>",
	Short: "Split a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		id := args[0]

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "transactions.split", err.(*errors.Error), start)
			return
		}

		data, err := os.ReadFile(splitFile)
		if err != nil {
			handleError(renderer, "transactions.split", errors.New(errors.ValidationFailed, "failed to read split file: "+err.Error(), errors.CatValidation, false, err), start)
			return
		}

		var splits []monarch.SplitInput
		if err := jsonUnmarshal(data, &splits); err != nil {
			handleError(renderer, "transactions.split", errors.New(errors.ValidationFailed, "invalid split JSON: "+err.Error(), errors.CatValidation, false, err), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("transactions.split", id, nil, map[string]any{"splits": splits})
			env := output.NewEnvelope("transactions.split", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
			return
		}

		deps, ok := newDeps(renderer, "transactions.split", start)
		if !ok {
			return
		}

		if _, err := deps.Mutate("transactions.split", id, func() (any, error) {
			return nil, deps.Service.UpdateTransactionSplits(cmd.Context(), id, splits)
		}, "failed to split transaction"); err != nil {
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.split", profile, output.SchemaVersion, "", map[string]string{"status": "split updated"}, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("Successfully split transaction %s.\n", id)
		}
	},
}

var transactionsExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export transactions",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "transactions.export", start)
		if !ok {
			return
		}
		svc := deps.Service

		opts := monarch.ListTransactionsOptions{
			Limit:     limit,
			Offset:    offset,
			StartDate: txStartDate,
			EndDate:   txEndDate,
			GoalIDs:   filterGoalIDs,
		}
		if cmd.Flags().Changed("pending") {
			opts.Pending = &filterPending
		}
		if cmd.Flags().Changed("hide-from-reports") {
			opts.HideFromReports = &filterHideReports
		}

		txs, _, err := svc.ListTransactions(cmd.Context(), opts)
		if err != nil {
			handleError(renderer, "transactions.export", wrapError(err, "failed to list transactions"), start)
			return
		}

		var out io.Writer = os.Stdout
		if outputFile != "" {
			f, err := os.Create(outputFile)
			if err != nil {
				handleError(renderer, "transactions.export", errors.New(errors.InternalError, "failed to create output file", errors.CatInternal, false, err), start)
				return
			}
			defer func() {
				if cerr := f.Close(); cerr != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to close file: %v\n", cerr)
				}
			}()
			out = f
		}

		if format == "csv" {
			if err := monarch.ExportTransactionsCSV(txs, out); err != nil {
				handleError(renderer, "transactions.export", errors.New(errors.InternalError, "failed to export CSV", errors.CatInternal, false, err), start)
				return
			}
		} else {
			// Default to JSON
			env := output.NewEnvelope("transactions.export", profile, output.SchemaVersion, "", txs, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		}
	},
}

var transactionsTagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "Manage transaction tags",
}

var transactionsTagsSetCmd = &cobra.Command{
	Use:   "set <transaction-id>",
	Short: "Set tags for a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		id := args[0]

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "transactions.tags.set", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("transactions.tags.set", id, nil, map[string]any{"tag_ids": tagIDs})
			env := output.NewEnvelope("transactions.tags.set", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
			return
		}

		deps, ok := newDeps(renderer, "transactions.tags.set", start)
		if !ok {
			return
		}

		if _, err := deps.Mutate("transactions.tags.set", id, func() (any, error) {
			return nil, deps.Service.SetTransactionTags(cmd.Context(), id, tagIDs)
		}, "failed to set transaction tags"); err != nil {
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.tags.set", profile, output.SchemaVersion, "", map[string]string{"status": "tags set"}, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("Successfully set tags for transaction %s.\n", id)
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

		deps, ok := newDeps(renderer, "transactions.attachments.list", start)
		if !ok {
			return
		}
		svc := deps.Service

		attachments, err := svc.ListTransactionAttachments(cmd.Context(), args[0])
		if err != nil {
			handleError(renderer, "transactions.attachments.list", wrapError(err, "failed to list attachments"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.attachments.list", profile, output.SchemaVersion, "", attachments, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			if len(attachments) == 0 {
				fmt.Println("No attachments found.")
			} else {
				fmt.Printf("%-36s %-20s %s\n", "ID", "FILENAME", "SIZE")
				for _, a := range attachments {
					fmt.Printf("%-36s %-20s %d bytes\n", a.ID, a.Filename, a.SizeBytes)
				}
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
			handleError(renderer, "transactions.attachments.download", errors.New(errors.InvalidArguments, "--id flag is required", errors.CatValidation, false, nil), start)
			return
		}

		deps, ok := newDeps(renderer, "transactions.attachments.download", start)
		if !ok {
			return
		}
		svc := deps.Service

		// Get attachments to find the URL
		attachments, err := svc.ListTransactionAttachments(cmd.Context(), args[0])
		if err != nil {
			handleError(renderer, "transactions.attachments.download", wrapError(err, "failed to list attachments"), start)
			return
		}

		var targetURL, targetFilename string
		for _, a := range attachments {
			if a.ID == attachmentID {
				targetURL = a.URL
				targetFilename = a.Filename
				break
			}
		}
		if targetURL == "" {
			handleError(renderer, "transactions.attachments.download", errors.New(errors.ResourceNotFound, "attachment not found", errors.CatAPI, false, nil), start)
			return
		}

		outPath := outputFile
		if outPath == "" {
			outPath = targetFilename
		}

		f, err := os.Create(outPath)
		if err != nil {
			handleError(renderer, "transactions.attachments.download", errors.New(errors.InternalError, "failed to create output file: "+err.Error(), errors.CatInternal, false, err), start)
			return
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to close file: %v\n", cerr)
			}
		}()

		if err := svc.DownloadAttachment(cmd.Context(), targetURL, f); err != nil {
			handleError(renderer, "transactions.attachments.download", wrapError(err, "failed to download attachment"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.attachments.download", profile, output.SchemaVersion, "", map[string]string{"status": "downloaded", "path": outPath}, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("Downloaded attachment to %s\n", outPath)
		}
	},
}

var transactionsShowCmd = &cobra.Command{
	Use:   "show <transaction-id>",
	Short: "Show detailed information for a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "transactions.show", start)
		if !ok {
			return
		}
		svc := deps.Service

		tx, err := svc.GetTransaction(cmd.Context(), args[0])
		if err != nil {
			handleError(renderer, "transactions.show", wrapError(err, "failed to get transaction"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.show", profile, output.SchemaVersion, "", tx, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("ID:       %s\n", tx.ID)
			fmt.Printf("Date:     %s\n", tx.Date)
			fmt.Printf("Merchant: %s\n", tx.Merchant)
			fmt.Printf("Category: %s\n", tx.Category)
			fmt.Printf("Amount:   %.2f\n", tx.Amount)
			fmt.Printf("Notes:    %s\n", tx.Notes)
		}
	},
}

var transactionsSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Get transaction summary",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		deps, ok := newDeps(renderer, "transactions.summary", start)
		if !ok {
			return
		}
		svc := deps.Service

		summary, err := svc.GetTransactionsSummary(cmd.Context(), txStartDate, txEndDate)
		if err != nil {
			handleError(renderer, "transactions.summary", wrapError(err, "failed to get transaction summary"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.summary", profile, output.SchemaVersion, "", summary, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Println("Transaction Summary")
		}
	},
}

var transactionsTagsClearCmd = &cobra.Command{
	Use:   "clear <transaction-id>",
	Short: "Clear all tags for a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		id := args[0]

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "transactions.tags.clear", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("transactions.tags.clear", id, nil, nil)
			env := output.NewEnvelope("transactions.tags.clear", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
			return
		}

		deps, ok := newDeps(renderer, "transactions.tags.clear", start)
		if !ok {
			return
		}

		if _, err := deps.Mutate("transactions.tags.clear", id, func() (any, error) {
			return nil, deps.Service.SetTransactionTags(cmd.Context(), id, []string{})
		}, "failed to clear transaction tags"); err != nil {
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.tags.clear", profile, output.SchemaVersion, "", map[string]string{"status": "tags cleared"}, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("Successfully cleared tags for transaction %s.\n", id)
		}
	},
}

var transactionsTagsAddCmd = &cobra.Command{
	Use:   "add <transaction-id>",
	Short: "Add tags to a transaction (appending to existing tags)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		logger := audit.NewLogger()
		id := args[0]

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "transactions.tags.add", err.(*errors.Error), start)
			return
		}

		if len(tagIDs) == 0 {
			handleError(renderer, "transactions.tags.add", errors.New(errors.InvalidArguments, "--tag is required", errors.CatValidation, false, nil), start)
			return
		}

		deps, ok := newDeps(renderer, "transactions.tags.add", start)
		if !ok {
			return
		}
		svc := deps.Service

		// Fetch existing tags
		tx, err := svc.GetTransaction(cmd.Context(), id)
		if err != nil {
			handleError(renderer, "transactions.tags.add", errors.New(errors.APIError, "failed to fetch current transaction", errors.CatAPI, false, err), start)
			return
		}

		existingTagIDs := make(map[string]bool)
		newTagIDs := []string{}

		for _, t := range tx.Tags {
			existingTagIDs[t.ID] = true
			newTagIDs = append(newTagIDs, t.ID)
		}

		for _, tid := range tagIDs {
			if !existingTagIDs[tid] {
				newTagIDs = append(newTagIDs, tid)
			}
		}

		if dryRun {
			plan := safety.NewPlan()
			plan.Add("transactions.tags.add", id, nil, map[string]any{"tag_ids": newTagIDs})
			env := output.NewEnvelope("transactions.tags.add", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
			return
		}

		err = svc.SetTransactionTags(cmd.Context(), id, newTagIDs)
		result := "success"
		var errCode string
		if err != nil {
			result = "failure"
			if e, ok := err.(*errors.Error); ok {
				errCode = string(e.Code)
			}
		}

		logger.Log(&audit.Record{ //nolint:errcheck // best-effort audit
			Command:    "transactions.tags.add",
			ResourceID: id,
			DryRun:     dryRun,
			Confirmed:  confirm,
			Profile:    profile,
			Result:     result,
			ErrorCode:  errCode,
		})

		if err != nil {
			handleError(renderer, "transactions.tags.add", wrapError(err, "failed to add transaction tags"), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("transactions.tags.add", profile, output.SchemaVersion, "", map[string]string{"status": "tags added"}, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("Successfully added tags to transaction %s.\n", id)
		}
	},
}

var transactionsBulkCategorizeCmd = &cobra.Command{
	Use:   "bulk-categorize",
	Short: "Apply a category to multiple transactions",
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		logger := audit.NewLogger()

		if len(bulkTxIDs) == 0 {
			handleError(renderer, "transactions.bulk-categorize", errors.New(errors.InvalidArguments, "at least one --id is required", errors.CatValidation, false, nil), start)
			return
		}
		if bulkCategoryID == "" {
			handleError(renderer, "transactions.bulk-categorize", errors.New(errors.InvalidArguments, "--category-id is required", errors.CatValidation, false, nil), start)
			return
		}

		if err := safety.Check(safety.TierMutation, readOnly, dryRun, confirm); err != nil {
			handleError(renderer, "transactions.bulk-categorize", err.(*errors.Error), start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			for _, id := range bulkTxIDs {
				plan.Add("transactions.update", id, nil, map[string]any{"categoryId": bulkCategoryID, "markReviewed": bulkMarkReviewed})
			}
			env := output.NewEnvelope("transactions.bulk-categorize", profile, output.SchemaVersion, "", plan, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
			return
		}

		deps, ok := newDeps(renderer, "transactions.bulk-categorize", start)
		if !ok {
			return
		}
		svc := deps.Service

		var needsReview *bool
		if bulkMarkReviewed {
			f := false
			needsReview = &f
		}

		successes := 0
		var failures []string
		for _, txID := range bulkTxIDs {
			_, err := svc.UpdateTransaction(cmd.Context(), txID, nil, &bulkCategoryID, nil, nil, nil, nil, needsReview)
			if err != nil {
				failures = append(failures, txID+": "+err.Error())
			} else {
				successes++
			}
		}

		result := "success"
		if len(failures) > 0 && successes == 0 {
			result = "failure"
		} else if len(failures) > 0 {
			result = "partial"
		}
		logger.Log(&audit.Record{Command: "transactions.bulk-categorize", DryRun: dryRun, Confirmed: confirm, Profile: profile, Result: result}) //nolint:errcheck // best-effort audit

		if jsonMode {
			data := map[string]any{"total": len(bulkTxIDs), "successful": successes, "failed": len(failures), "errors": failures}
			env := output.NewEnvelope("transactions.bulk-categorize", profile, output.SchemaVersion, "", data, time.Since(start))
			renderer.RenderSuccess(env) //nolint:errcheck // best-effort render
		} else {
			fmt.Printf("Bulk categorize: %d/%d successful.\n", successes, len(bulkTxIDs))
			for _, f := range failures {
				fmt.Printf("  FAILED: %s\n", f)
			}
		}
	},
}

var transactionsAttachmentsUploadCmd = &cobra.Command{
	Use:   "upload <transaction-id> <file>",
	Short: "Upload an attachment for a transaction",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
		handleError(renderer, "transactions.attachments.upload", errors.New(errors.FEATURE_UNAVAILABLE, "transaction attachment upload is unavailable in the current Monarch API", errors.CatAPI, false, nil), start)
	},
}

func init() {
	transactionsCmd.PersistentFlags().StringVar(&txStartDate, "from", "", "start date (YYYY-MM-DD)")
	transactionsCmd.PersistentFlags().StringVar(&txEndDate, "to", "", "end date (YYYY-MM-DD)")

	transactionsListCmd.Flags().IntVar(&limit, "limit", 100, "maximum number of transactions to return")
	transactionsListCmd.Flags().IntVar(&offset, "offset", 0, "number of transactions to skip")
	transactionsListCmd.Flags().StringSliceVar(&filterCategoryIDs, "category-id", nil, "filter by category ID (repeatable)")
	transactionsListCmd.Flags().StringSliceVar(&filterAccountIDs, "account-id", nil, "filter by account ID (repeatable)")
	transactionsListCmd.Flags().StringSliceVar(&filterTagIDs, "tag-id", nil, "filter by tag ID (repeatable)")
	transactionsListCmd.Flags().BoolVar(&filterNeedsReview, "needs-review", false, "filter for transactions needing review")
	transactionsListCmd.Flags().BoolVar(&filterHasNotes, "has-notes", false, "filter for transactions with notes")
	transactionsListCmd.Flags().BoolVar(&filterIsSplit, "is-split", false, "filter for split transactions")
	transactionsListCmd.Flags().BoolVar(&filterIsRecurring, "is-recurring", false, "filter for recurring transactions")
	transactionsListCmd.Flags().BoolVar(&filterPending, "pending", false, "filter by pending status")
	transactionsListCmd.Flags().BoolVar(&filterHideReports, "hide-from-reports", false, "filter by hide-from-reports status")
	transactionsListCmd.Flags().StringSliceVar(&filterGoalIDs, "goal-id", nil, "filter by goal ID (repeatable)")

	transactionsSearchCmd.Flags().IntVar(&limit, "limit", 100, "maximum number of transactions to return")
	transactionsSearchCmd.Flags().IntVar(&offset, "offset", 0, "number of transactions to skip")

	transactionsExportCmd.Flags().IntVar(&limit, "limit", 1000, "maximum number of transactions to export")
	transactionsExportCmd.Flags().IntVar(&offset, "offset", 0, "number of transactions to skip")
	transactionsExportCmd.Flags().StringVar(&format, "format", "json", "export format (json or csv)")
	must(transactionsExportCmd.RegisterFlagCompletionFunc("format", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"json", "csv"}, cobra.ShellCompDirectiveNoFileComp
	}))
	transactionsExportCmd.Flags().StringVar(&outputFile, "output", "", "output file path")
	transactionsExportCmd.Flags().BoolVar(&filterPending, "pending", false, "filter by pending status")
	transactionsExportCmd.Flags().BoolVar(&filterHideReports, "hide-from-reports", false, "filter by hide-from-reports status")
	transactionsExportCmd.Flags().StringSliceVar(&filterGoalIDs, "goal-id", nil, "filter by goal ID (repeatable)")

	transactionsUpdateCmd.Flags().StringVar(&txNotes, "notes", "", "transaction notes")
	transactionsUpdateCmd.Flags().StringVar(&txCategoryID, "category", "", "transaction category ID")
	transactionsUpdateCmd.Flags().Float64Var(&txAmount, "amount", 0, "transaction amount")
	transactionsUpdateCmd.Flags().StringVar(&txDate, "date", "", "transaction date (YYYY-MM-DD)")
	transactionsUpdateCmd.Flags().StringVar(&txMerchant, "merchant", "", "merchant name")
	transactionsUpdateCmd.Flags().BoolVar(&txHideFromReports, "hide-from-reports", false, "hide transaction from reports")
	transactionsUpdateCmd.Flags().BoolVar(&txNeedsReview, "needs-review", false, "mark transaction as needing review")
	transactionsUpdateCmd.Flags().BoolVar(&txMarkReviewed, "mark-reviewed", false, "mark transaction as reviewed (shortcut for --needs-review=false)")

	transactionsCreateCmd.Flags().Float64Var(&txAmount, "amount", 0, "transaction amount")
	transactionsCreateCmd.Flags().StringVar(&txMerchant, "merchant", "", "merchant name")
	transactionsCreateCmd.Flags().StringVar(&txDate, "date", "", "transaction date (YYYY-MM-DD)")
	transactionsCreateCmd.Flags().StringVar(&txCategoryID, "category", "", "category ID")
	transactionsCreateCmd.Flags().StringVar(&txAccountID, "account", "", "account ID")
	transactionsCreateCmd.Flags().StringVar(&txNotes, "notes", "", "transaction notes")
	transactionsCreateCmd.MarkFlagRequired("amount")   //nolint:errcheck // flag registered above
	transactionsCreateCmd.MarkFlagRequired("merchant") //nolint:errcheck // flag registered above
	transactionsCreateCmd.MarkFlagRequired("category") //nolint:errcheck // flag registered above

	transactionsSplitCmd.Flags().StringVar(&splitFile, "file", "", "JSON file with split data")
	transactionsSplitCmd.MarkFlagRequired("file") //nolint:errcheck // flag registered above

	transactionsTagsSetCmd.Flags().StringSliceVar(&tagIDs, "tag", []string{}, "tag IDs to set")
	transactionsTagsSetCmd.MarkFlagRequired("tag") //nolint:errcheck // flag registered above

	transactionsTagsAddCmd.Flags().StringSliceVar(&tagIDs, "tag", []string{}, "tag IDs to add")
	transactionsTagsAddCmd.MarkFlagRequired("tag") //nolint:errcheck // flag registered above

	transactionsAttachmentsDownloadCmd.Flags().StringVar(&attachmentID, "id", "", "attachment ID")
	transactionsAttachmentsDownloadCmd.Flags().StringVar(&outputFile, "output", "", "output file path")

	transactionsBulkCategorizeCmd.Flags().StringSliceVar(&bulkTxIDs, "id", nil, "transaction IDs to categorize (repeatable)")
	transactionsBulkCategorizeCmd.Flags().StringVar(&bulkCategoryID, "category-id", "", "category ID to apply")
	transactionsBulkCategorizeCmd.Flags().BoolVar(&bulkMarkReviewed, "mark-reviewed", true, "also mark transactions as reviewed")

	transactionsTagsCmd.AddCommand(transactionsTagsSetCmd)
	transactionsTagsCmd.AddCommand(transactionsTagsAddCmd)
	transactionsTagsCmd.AddCommand(transactionsTagsClearCmd)
	transactionsCmd.AddCommand(transactionsTagsCmd)

	transactionsAttachmentsCmd.AddCommand(transactionsAttachmentsListCmd)
	transactionsAttachmentsCmd.AddCommand(transactionsAttachmentsUploadCmd)
	transactionsAttachmentsCmd.AddCommand(transactionsAttachmentsDownloadCmd)
	transactionsCmd.AddCommand(transactionsAttachmentsCmd)

	transactionsCmd.AddCommand(transactionsListCmd)
	transactionsCmd.AddCommand(transactionsSearchCmd)
	transactionsCmd.AddCommand(transactionsShowCmd)
	transactionsCmd.AddCommand(transactionsSummaryCmd)
	transactionsCmd.AddCommand(transactionsDuplicatesCmd)
	transactionsCmd.AddCommand(transactionsSplitsCmd)
	transactionsCmd.AddCommand(transactionsExportCmd)
	transactionsCmd.AddCommand(transactionsUpdateCmd)
	transactionsCmd.AddCommand(transactionsDeleteCmd)
	transactionsCmd.AddCommand(transactionsCreateCmd)
	transactionsCmd.AddCommand(transactionsSplitCmd)
	transactionsCmd.AddCommand(transactionsBulkCategorizeCmd)
	RootCmd.AddCommand(transactionsCmd)
}
