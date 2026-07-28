package cli

import (
	"context"
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

	txHideFromReports bool
	txNeedsReview     bool
	txMarkReviewed    bool

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
		var txs []monarch.Transaction
		var total int
		runWarn(cmd.Context(), "transactions.list", "failed to list transactions",
			[]string{"uses legacy Monarch GraphQL root field: allTransactions"},
			func(ctx context.Context, svc *monarch.Service) (map[string]any, error) {
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

				t, tot, err := svc.ListTransactions(ctx, &opts)
				if err != nil {
					return nil, err
				}
				txs, total = t, tot
				return map[string]any{"transactions": t, "total": tot}, nil
			},
			func(_ map[string]any) {
				fmt.Printf("%-12s %-20s %-15s %10s %s\n", "DATE", "MERCHANT", "CATEGORY", "AMOUNT", "NOTES")
				for _, t := range txs {
					fmt.Printf("%-12s %-20s %-15s %10.2f %s\n", t.Date, t.Merchant, t.Category, t.Amount, t.Notes)
				}
				fmt.Printf("\nTotal transactions: %d\n", total)
			})
	},
}

var transactionsSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search transactions",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var txs []monarch.Transaction
		var total int
		runWarn(cmd.Context(), "transactions.search", "failed to search transactions",
			[]string{"uses legacy Monarch GraphQL root field: allTransactions"},
			func(ctx context.Context, svc *monarch.Service) (map[string]any, error) {
				t, tot, err := svc.ListTransactions(ctx, &monarch.ListTransactionsOptions{
					Limit:     limit,
					Offset:    offset,
					Search:    args[0],
					StartDate: txStartDate,
					EndDate:   txEndDate,
				})
				if err != nil {
					return nil, err
				}
				txs, total = t, tot
				return map[string]any{"transactions": t, "total": tot}, nil
			},
			func(_ map[string]any) {
				fmt.Printf("%-12s %-20s %-15s %10s %s\n", "DATE", "MERCHANT", "CATEGORY", "AMOUNT", "NOTES")
				for _, t := range txs {
					fmt.Printf("%-12s %-20s %-15s %10.2f %s\n", t.Date, t.Merchant, t.Category, t.Amount, t.Notes)
				}
				fmt.Printf("\nTotal matches: %d\n", total)
			})
	},
}

var transactionsDuplicatesCmd = &cobra.Command{
	Use:   "duplicates",
	Short: "Find duplicate transactions",
	Run: func(cmd *cobra.Command, args []string) {
		runWarn(cmd.Context(), "transactions.duplicates", "failed to find duplicates",
			[]string{"uses legacy Monarch GraphQL root field: allTransactions"},
			func(ctx context.Context, svc *monarch.Service) ([]monarch.Transaction, error) {
				now := time.Now()
				startDate := now.Format("2006-01-02")
				endDate := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
				return svc.GetDuplicateTransactions(ctx, startDate, endDate)
			},
			func(txs []monarch.Transaction) {
				fmt.Printf("%-12s %-20s %10s %s\n", "DATE", "MERCHANT", "AMOUNT", "ID")
				for _, t := range txs {
					fmt.Printf("%-12s %-20s %10.2f %s\n", t.Date, t.Merchant, t.Amount, t.ID)
				}
			})
	},
}

var transactionsSplitsCmd = &cobra.Command{
	Use:   "splits <transaction-id>",
	Short: "Get splits for a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "transactions.splits", "failed to get splits",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.TransactionSplit, error) {
				return svc.GetTransactionSplits(ctx, args[0])
			},
			func(splits []monarch.TransactionSplit) {
				fmt.Printf("%-20s %10s %s\n", "CATEGORY", "AMOUNT", "NOTES")
				for _, s := range splits {
					fmt.Printf("%-20s %10.2f %s\n", s.Category, s.Amount, s.Notes)
				}
			})
	},
}

var transactionsUpdateCmd = &cobra.Command{
	Use:   "update <transaction-id>",
	Short: "Update a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "transactions.update", "failed to update transaction", safety.TierMutation, func() (mutation, *errors.Error) {
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
			var tx *monarch.Transaction
			return mutation{
				resourceID: id,
				planAfter:  map[string]any{"notes": notes, "categoryId": categoryID, "amount": amount, "date": date, "merchant": merchantName, "hideFromReports": hideFromReports, "needsReview": needsReview},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					updated, err := svc.UpdateTransaction(ctx, id, notes, categoryID, amount, date, merchantName, hideFromReports, needsReview)
					if err != nil {
						return nil, err
					}
					tx = updated
					return updated, nil
				},
				human: func() { fmt.Printf("Successfully updated transaction %s.\n", tx.ID) },
			}, nil
		})
	},
}

var transactionsDeleteCmd = &cobra.Command{
	Use:   "delete <transaction-id>",
	Short: "Delete a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "transactions.delete", "failed to delete transaction", safety.TierDestructive, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.DeleteTransaction(ctx, id); err != nil {
						return nil, err
					}
					return map[string]string{"status": "deleted"}, nil
				},
				human: func() { fmt.Printf("Successfully deleted transaction %s.\n", id) },
			}, nil
		})
	},
}

var transactionsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a transaction",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "transactions.create", "failed to create transaction", safety.TierMutation, func() (mutation, *errors.Error) {
			if txDate == "" {
				txDate = time.Now().Format("2006-01-02")
			}
			var tx *monarch.Transaction
			return mutation{
				planAfter: map[string]any{"amount": txAmount, "merchant": txMerchant, "date": txDate, "categoryId": txCategoryID},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					created, err := svc.CreateTransaction(ctx, txAmount, txMerchant, txDate, txCategoryID, txAccountID, txNotes)
					if err != nil {
						return nil, err
					}
					tx = created
					return created, nil
				},
				human: func() { fmt.Printf("Successfully created transaction %s.\n", tx.ID) },
			}, nil
		})
	},
}

var transactionsSplitCmd = &cobra.Command{
	Use:   "split <transaction-id>",
	Short: "Split a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "transactions.split", "failed to split transaction", safety.TierMutation, func() (mutation, *errors.Error) {
			data, err := os.ReadFile(splitFile)
			if err != nil {
				return mutation{}, errors.New(errors.ValidationFailed, "failed to read split file: "+err.Error(), errors.CatValidation, false, err)
			}
			var splits []monarch.SplitInput
			if err := jsonUnmarshal(data, &splits); err != nil {
				return mutation{}, errors.New(errors.ValidationFailed, "invalid split JSON: "+err.Error(), errors.CatValidation, false, err)
			}
			return mutation{
				resourceID: id,
				planAfter:  map[string]any{"splits": splits},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.UpdateTransactionSplits(ctx, id, splits); err != nil {
						return nil, err
					}
					return map[string]string{"status": "split updated"}, nil
				},
				human: func() { fmt.Printf("Successfully split transaction %s.\n", id) },
			}, nil
		})
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

		txs, _, err := svc.ListTransactions(cmd.Context(), &opts)
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
			env := output.NewEnvelope("transactions.export", profile, output.SchemaVersion, requestID, txs, time.Since(start))
			renderer.RenderSuccess(env)
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
		id := args[0]
		runMutation(cmd, "transactions.tags.set", "failed to set transaction tags", safety.TierMutation, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				planAfter:  map[string]any{"tag_ids": tagIDs},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.SetTransactionTags(ctx, id, tagIDs); err != nil {
						return nil, err
					}
					return map[string]string{"status": "tags set"}, nil
				},
				human: func() { fmt.Printf("Successfully set tags for transaction %s.\n", id) },
			}, nil
		})
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
		run(cmd.Context(), "transactions.attachments.list", "failed to list attachments",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.Attachment, error) {
				return svc.ListTransactionAttachments(ctx, args[0])
			},
			func(attachments []monarch.Attachment) {
				if len(attachments) == 0 {
					fmt.Println("No attachments found.")
					return
				}
				fmt.Printf("%-36s %-20s %s\n", "ID", "FILENAME", "SIZE")
				for _, a := range attachments {
					fmt.Printf("%-36s %-20s %d bytes\n", a.ID, a.Filename, a.SizeBytes)
				}
			})
	},
}

var transactionsAttachmentsDownloadCmd = &cobra.Command{
	Use:   "download <transaction-id>",
	Short: "Download an attachment for a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var outPath string
		run(cmd.Context(), "transactions.attachments.download", "failed to download attachment",
			func(ctx context.Context, svc *monarch.Service) (map[string]string, error) {
				if attachmentID == "" {
					return nil, errors.New(errors.InvalidArguments, "--id flag is required", errors.CatValidation, false, nil)
				}
				attachments, err := svc.ListTransactionAttachments(ctx, args[0])
				if err != nil {
					return nil, wrapError(err, "failed to list attachments")
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
					return nil, errors.New(errors.ResourceNotFound, "attachment not found", errors.CatAPI, false, nil)
				}
				outPath = outputFile
				if outPath == "" {
					outPath = targetFilename
				}
				f, err := os.Create(outPath)
				if err != nil {
					return nil, errors.New(errors.InternalError, "failed to create output file: "+err.Error(), errors.CatInternal, false, err)
				}
				defer func() {
					if cerr := f.Close(); cerr != nil {
						fmt.Fprintf(os.Stderr, "warning: failed to close file: %v\n", cerr)
					}
				}()
				if err := svc.DownloadAttachment(ctx, targetURL, f); err != nil {
					return nil, wrapError(err, "failed to download attachment")
				}
				return map[string]string{"status": "downloaded", "path": outPath}, nil
			},
			func(_ map[string]string) {
				fmt.Printf("Downloaded attachment to %s\n", outPath)
			})
	},
}

var transactionsShowCmd = &cobra.Command{
	Use:   "show <transaction-id>",
	Short: "Show detailed information for a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "transactions.show", "failed to get transaction",
			func(ctx context.Context, svc *monarch.Service) (*monarch.Transaction, error) {
				return svc.GetTransaction(ctx, args[0])
			},
			func(tx *monarch.Transaction) {
				fmt.Printf("ID:       %s\n", tx.ID)
				fmt.Printf("Date:     %s\n", tx.Date)
				fmt.Printf("Merchant: %s\n", tx.Merchant)
				fmt.Printf("Category: %s\n", tx.Category)
				fmt.Printf("Amount:   %.2f\n", tx.Amount)
				fmt.Printf("Notes:    %s\n", tx.Notes)
			})
	},
}

var transactionsSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Get transaction summary",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "transactions.summary", "failed to get transaction summary",
			func(ctx context.Context, svc *monarch.Service) (*monarch.TransactionSummaryResult, error) {
				return svc.GetTransactionsSummary(ctx, txStartDate, txEndDate)
			},
			func(_ *monarch.TransactionSummaryResult) {
				fmt.Println("Transaction Summary")
			})
	},
}

var transactionsTagsClearCmd = &cobra.Command{
	Use:   "clear <transaction-id>",
	Short: "Clear all tags for a transaction",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "transactions.tags.clear", "failed to clear transaction tags", safety.TierMutation, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.SetTransactionTags(ctx, id, []string{}); err != nil {
						return nil, err
					}
					return map[string]string{"status": "tags cleared"}, nil
				},
				human: func() { fmt.Printf("Successfully cleared tags for transaction %s.\n", id) },
			}, nil
		})
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
			handleError(renderer, "transactions.tags.add", err, start)
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
			env := output.NewEnvelope("transactions.tags.add", profile, output.SchemaVersion, requestID, plan, time.Since(start))
			renderer.RenderSuccess(env)
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
			env := output.NewEnvelope("transactions.tags.add", profile, output.SchemaVersion, requestID, map[string]string{"status": "tags added"}, time.Since(start))
			renderer.RenderSuccess(env)
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
			handleError(renderer, "transactions.bulk-categorize", err, start)
			return
		}

		if dryRun {
			plan := safety.NewPlan()
			for _, id := range bulkTxIDs {
				plan.Add("transactions.update", id, nil, map[string]any{"categoryId": bulkCategoryID, "markReviewed": bulkMarkReviewed})
			}
			env := output.NewEnvelope("transactions.bulk-categorize", profile, output.SchemaVersion, requestID, plan, time.Since(start))
			renderer.RenderSuccess(env)
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
			env := output.NewEnvelope("transactions.bulk-categorize", profile, output.SchemaVersion, requestID, data, time.Since(start))
			renderer.RenderSuccess(env)
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
