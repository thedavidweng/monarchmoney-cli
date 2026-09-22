package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/monarch"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
	"github.com/thedavidweng/monarchmoney-cli/internal/safety"
)

var (
	receiptStatus         string
	receiptSource         string
	receiptMatchedOnly    bool
	receiptUnmatchedOnly  bool
	receiptTransactionID  string
	receiptMerchant       string
	receiptDate           string
	receiptSubtotal       float64
	receiptTax            float64
	receiptTip            float64
	receiptTotal          float64
	receiptAutoCategorize bool
	receiptUpdateNotes    bool
)

var receiptsCmd = &cobra.Command{
	Use:     "receipts",
	Short:   "Manage Monarch receipt inbox (scan, match, download)",
	GroupID: "core",
	Example: "  monarch receipts list --json\n  monarch receipts upload receipt.jpg --confirm\n  monarch receipts match <receipt-id> --transaction <transaction-id> --confirm",
}

var receiptsUploadCmd = &cobra.Command{
	Use:   "upload <file>",
	Short: "Upload a receipt to the inbox for AI categorization and matching (requires --confirm)",
	Long:  `Upload an image or PDF for AI extraction and auto-matching. Parsing is asynchronous: a fresh receipts show may report in_progress or pending before merchant and totals appear. Unmatched results land in receipts list --unmatched for manual matching.`,
	Example: `  monarch receipts upload receipt.jpg --confirm --json
  monarch receipts list --unmatched --json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		runMutation(cmd, "receipts.upload", "failed to upload receipt", safety.TierMutation, func() (mutation, *errors.Error) {
			var sync *monarch.ReceiptSync
			return mutation{
				planAfter: map[string]string{"file": path},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					result, err := svc.UploadReceiptToInbox(ctx, path)
					if err != nil {
						return nil, err
					}
					sync = result
					return result, nil
				},
				human: func() {
					fmt.Printf("Receipt uploaded. Sync %s status: %s\n", sync.ID, sync.Status)
				},
			}, nil
		})
	},
}

var receiptsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List receipt inbox entries with match status",
	Long:  `List receipt inbox entries. --status and --source filter server-side. --matched/--unmatched apply client-side: the CLI fetches all pages, filters by match state, then applies --limit/--offset, so totals reflect the filter. There is no server-side merchant or amount search: pull with --json and filter orders by merchantName, date, and grandTotal locally. Pair with transactions search to find match candidates.`,
	Example: `  monarch receipts list --unmatched --json
  monarch receipts list --status pending_matches --source upload --json`,
	Run: func(cmd *cobra.Command, args []string) {
		vendor, err := receiptVendorFilter(receiptSource)
		if err != nil {
			failReceiptsList(err)
			return
		}
		if receiptMatchedOnly && receiptUnmatchedOnly {
			failReceiptsList(errors.New(errors.InvalidArguments, "--matched and --unmatched are mutually exclusive", errors.CatValidation, false, nil))
			return
		}
		var matchedOnly *bool
		switch {
		case receiptMatchedOnly:
			matchedOnly = &receiptMatchedOnly
		case receiptUnmatchedOnly:
			unmatched := false
			matchedOnly = &unmatched
		}
		var receipts []*monarch.Receipt
		var total int
		run(cmd.Context(), "receipts.list", "failed to list receipts",
			func(ctx context.Context, svc *monarch.Service) (map[string]any, error) {
				list, tot, err := svc.ListReceipts(ctx, receiptStatus, vendor, limit, offset, matchedOnly)
				if err != nil {
					return nil, err
				}
				receipts, total = list, tot
				return map[string]any{"receipts": list, "total": tot}, nil
			},
			func(_ map[string]any) {
				fmt.Printf("%-36s %-12s %-14s %-8s %s\n", "ID", "STATUS", "MERCHANT", "MATCHED", "TOTAL")
				for _, r := range receipts {
					merchant, grandTotal := receiptSummary(r)
					fmt.Printf("%-36s %-12s %-14s %-8t %s\n", r.ID, r.Status, merchant, r.IsMatched(), grandTotal)
				}
				fmt.Printf("\nTotal receipts: %d\n", total)
			})
	},
}

var receiptsShowCmd = &cobra.Command{
	Use:   "show <receipt-id>",
	Short: "Show receipt details including matched transactions",
	Long:  `Show merchant, date, totals, line items, and matched transaction IDs for one receipt. Follow the IDs with transactions show. Unmatched receipts are match candidates: search transactions by the receipt merchant within a few days of the receipt date.`,
	Example: `  monarch receipts show <receipt-id> --json
  monarch transactions show <transaction-id> --json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "receipts.show", "failed to get receipt",
			func(ctx context.Context, svc *monarch.Service) (*monarch.Receipt, error) {
				return svc.GetReceipt(ctx, args[0])
			},
			func(r *monarch.Receipt) {
				merchant, grandTotal := receiptSummary(r)
				fmt.Printf("ID:       %s\n", r.ID)
				fmt.Printf("Status:   %s\n", r.Status)
				fmt.Printf("Source:   %s\n", r.Vendor)
				fmt.Printf("Merchant: %s\n", merchant)
				fmt.Printf("Total:    %s\n", grandTotal)
				fmt.Printf("Matched:  %t %v\n", r.IsMatched(), r.MatchedTransactionIDs())
				for _, o := range r.Orders {
					fmt.Printf("Order:    %s date=%s status=%s\n", o.ID, o.Date, o.DisplayStatus)
					for _, li := range o.LineItems {
						fmt.Printf("  - %s\n", li.Title)
					}
				}
			})
	},
}

var receiptsDownloadCmd = &cobra.Command{
	Use:     "download <receipt-id>",
	Short:   "Download the receipt image",
	Long:    `Download the first attachment of a receipt. Without --output the file is saved under the remote filename; use --output to choose a path. Fails when the receipt has no image.`,
	Example: `  monarch receipts download <receipt-id> --output /tmp/receipt.jpg`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var outPath string
		run(cmd.Context(), "receipts.download", "failed to download receipt",
			func(ctx context.Context, svc *monarch.Service) (map[string]string, error) {
				var buf bytes.Buffer
				filename, err := svc.DownloadReceipt(ctx, args[0], &buf)
				if err != nil {
					return nil, wrapError(err, "failed to download receipt")
				}
				outPath = outputFile
				if outPath == "" {
					outPath = filename
				}
				if err := os.WriteFile(outPath, buf.Bytes(), 0o600); err != nil {
					return nil, errors.New(errors.InternalError, "failed to write output file: "+err.Error(), errors.CatInternal, false, err)
				}
				return map[string]string{"status": "downloaded", "path": outPath, "filename": filename}, nil
			},
			func(_ map[string]string) {
				fmt.Printf("Downloaded receipt to %s\n", outPath)
			})
	},
}

var receiptsDeleteCmd = &cobra.Command{
	Use:     "delete <receipt-id>",
	Short:   "Delete an unmatched receipt (requires --confirm)",
	Long:    `Delete a receipt that was never matched. The server rejects deleting matched receipts: receipts unmatch first.`,
	Example: `  monarch receipts delete <receipt-id> --dry-run --json`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "receipts.delete", "failed to delete receipt", safety.TierDestructive, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					if err := svc.DeleteReceipt(ctx, id); err != nil {
						return nil, err
					}
					return map[string]string{"status": "deleted"}, nil
				},
				human: func() { fmt.Printf("Successfully deleted receipt %s.\n", id) },
			}, nil
		})
	},
}

var receiptsMatchCmd = &cobra.Command{
	Use:   "match <receipt-id>",
	Short: "Manually match a receipt to a transaction (requires --confirm)",
	Long:  `Match an unmatched receipt to a transaction. Manual matching ignores amount differences, so discounts, tips, and tax misreads that defeat auto-match are fine: compare merchant and date instead. Fails on already-matched receipts and on receipts with no transaction slot. Preview with --dry-run; undo with receipts unmatch.`,
	Example: `  monarch receipts show <receipt-id> --json
  monarch transactions search "<merchant>" --from <date-3d> --to <date+3d> --json
  monarch receipts match <receipt-id> --transaction <transaction-id> --dry-run --json`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "receipts.match", "failed to match receipt", safety.TierMutation, func() (mutation, *errors.Error) {
			if receiptTransactionID == "" {
				return mutation{}, errors.New(errors.InvalidArguments, "--transaction is required", errors.CatValidation, false, nil)
			}
			var receipt *monarch.Receipt
			return mutation{
				resourceID: id,
				planAfter:  map[string]string{"transaction_id": receiptTransactionID},
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					matched, err := svc.MatchReceipt(ctx, id, receiptTransactionID)
					if err != nil {
						return nil, err
					}
					receipt = matched
					return matched, nil
				},
				human: func() { fmt.Printf("Receipt %s matched to %v.\n", id, receipt.MatchedTransactionIDs()) },
			}, nil
		})
	},
}

var receiptsUnmatchCmd = &cobra.Command{
	Use:     "unmatch <receipt-id>",
	Short:   "Remove the transaction match from a receipt (requires --confirm)",
	Long:    `Undo a wrong receipts match. Fails on receipts that are not currently matched.`,
	Example: `  monarch receipts unmatch <receipt-id> --dry-run --json`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "receipts.unmatch", "failed to unmatch receipt", safety.TierMutation, func() (mutation, *errors.Error) {
			return mutation{
				resourceID: id,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					receipt, err := svc.UnmatchReceipt(ctx, id)
					if err != nil {
						return nil, err
					}
					return receipt, nil
				},
				human: func() { fmt.Printf("Receipt %s unmatched.\n", id) },
			}, nil
		})
	},
}

var receiptsUpdateCmd = &cobra.Command{
	Use:     "update <receipt-id>",
	Short:   "Correct extracted receipt details (merchant, date, totals) (requires --confirm)",
	Long:    `Correct AI-extracted details on the first order of a receipt: merchant, date, subtotal, tax, tip, total. At least one correction flag is required. Fix the merchant or date before matching when the scan misread them.`,
	Example: `  monarch receipts update <receipt-id> --merchant "Whole Foods" --total 46.00 --dry-run --json`,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		runMutation(cmd, "receipts.update", "failed to update receipt", safety.TierMutation, func() (mutation, *errors.Error) {
			update := &monarch.ReceiptUpdate{}
			after := map[string]any{}
			if cmd.Flags().Changed("merchant") {
				update.MerchantName = &receiptMerchant
				after["merchant"] = receiptMerchant
			}
			if cmd.Flags().Changed("date") {
				update.Date = &receiptDate
				after["date"] = receiptDate
			}
			if cmd.Flags().Changed("subtotal") {
				update.TotalBeforeTax = &receiptSubtotal
				after["subtotal"] = receiptSubtotal
			}
			if cmd.Flags().Changed("tax") {
				update.Tax = &receiptTax
				after["tax"] = receiptTax
			}
			if cmd.Flags().Changed("tip") {
				update.Tip = &receiptTip
				after["tip"] = receiptTip
			}
			if cmd.Flags().Changed("total") {
				update.GrandTotal = &receiptTotal
				after["total"] = receiptTotal
			}
			if len(after) == 0 {
				return mutation{}, errors.New(errors.InvalidArguments, "at least one of --merchant, --date, --subtotal, --tax, --tip, --total is required", errors.CatValidation, false, nil)
			}
			var receipt *monarch.Receipt
			return mutation{
				resourceID: id,
				planAfter:  after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					updated, err := svc.UpdateReceipt(ctx, id, update)
					if err != nil {
						return nil, err
					}
					receipt = updated
					return updated, nil
				},
				human: func() { fmt.Printf("Successfully updated receipt %s.\n", receipt.ID) },
			}, nil
		})
	},
}

var receiptsSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Show receipt auto-categorize and notes preferences",
	Run: func(cmd *cobra.Command, args []string) {
		run(cmd.Context(), "receipts.settings", "failed to get receipt settings",
			func(ctx context.Context, svc *monarch.Service) ([]monarch.ReceiptSettings, error) {
				return svc.GetReceiptSettings(ctx)
			},
			func(settings []monarch.ReceiptSettings) {
				fmt.Printf("%-14s %-12s %s\n", "VENDOR", "AUTO-SPLIT", "UPDATE-NOTES")
				for _, st := range settings {
					fmt.Printf("%-14s %-12v %v\n", st.Vendor, boolValue(st.AutoCategorizeAndSplit), boolValue(st.UpdateTransactionNotes))
				}
			})
	},
}

var receiptsSettingsUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update receipt auto-categorize and notes preferences (requires --confirm)",
	Run: func(cmd *cobra.Command, args []string) {
		runMutation(cmd, "receipts.settings.update", "failed to update receipt settings", safety.TierMutation, func() (mutation, *errors.Error) {
			var autoCategorize, updateNotes *bool
			after := map[string]any{}
			if cmd.Flags().Changed("auto-categorize") {
				autoCategorize = &receiptAutoCategorize
				after["auto_categorize"] = receiptAutoCategorize
			}
			if cmd.Flags().Changed("update-notes") {
				updateNotes = &receiptUpdateNotes
				after["update_notes"] = receiptUpdateNotes
			}
			if len(after) == 0 {
				return mutation{}, errors.New(errors.InvalidArguments, "at least one of --auto-categorize, --update-notes is required", errors.CatValidation, false, nil)
			}
			var settings *monarch.ReceiptSettings
			return mutation{
				planAfter: after,
				do: func(ctx context.Context, svc *monarch.Service) (any, error) {
					updated, err := svc.UpdateReceiptSettings(ctx, autoCategorize, updateNotes)
					if err != nil {
						return nil, err
					}
					settings = updated
					return updated, nil
				},
				human: func() { fmt.Printf("Receipt settings updated for %s.\n", settings.Vendor) },
			}, nil
		})
	},
}

func failReceiptsList(err *errors.Error) {
	start := time.Now()
	renderer := output.NewRenderer(nil, nil, jsonMode, pretty)
	handleError(renderer, "receipts.list", err, start)
}

func receiptVendorFilter(source string) (string, *errors.Error) {
	switch source {
	case "", "all":
		return "", nil
	case "upload":
		return "user_import", nil
	case "email":
		return "email_import", nil
	default:
		return "", errors.New(errors.InvalidArguments, "--source must be upload or email", errors.CatValidation, false, nil)
	}
}

func receiptSummary(r *monarch.Receipt) (merchant, total string) {
	if len(r.Orders) == 0 {
		return "", ""
	}
	merchant = r.Orders[0].MerchantName
	if r.Orders[0].GrandTotal != nil {
		total = fmt.Sprintf("%.2f", *r.Orders[0].GrandTotal)
	}
	return merchant, total
}

func boolValue(b *bool) bool {
	return b != nil && *b
}

func init() {
	receiptsListCmd.Flags().IntVar(&limit, "limit", 100, "maximum number of receipts to return")
	receiptsListCmd.Flags().IntVar(&offset, "offset", 0, "number of receipts to skip")
	receiptsListCmd.Flags().StringVar(&receiptStatus, "status", "", "filter by status (completed, failed, in_progress, pending, pending_matches)")
	receiptsListCmd.Flags().StringVar(&receiptSource, "source", "", "filter by source (upload, email)")
	receiptsListCmd.Flags().BoolVar(&receiptMatchedOnly, "matched", false, "only show receipts matched to a transaction")
	receiptsListCmd.Flags().BoolVar(&receiptUnmatchedOnly, "unmatched", false, "only show unmatched receipts")

	receiptsDownloadCmd.Flags().StringVar(&outputFile, "output", "", "output file path")

	receiptsMatchCmd.Flags().StringVar(&receiptTransactionID, "transaction", "", "transaction ID to match the receipt to")
	receiptsMatchCmd.MarkFlagRequired("transaction") //nolint:errcheck // flag registered above

	receiptsUpdateCmd.Flags().StringVar(&receiptMerchant, "merchant", "", "corrected merchant name")
	receiptsUpdateCmd.Flags().StringVar(&receiptDate, "date", "", "corrected receipt date (YYYY-MM-DD)")
	receiptsUpdateCmd.Flags().Float64Var(&receiptSubtotal, "subtotal", 0, "corrected subtotal before tax")
	receiptsUpdateCmd.Flags().Float64Var(&receiptTax, "tax", 0, "corrected tax amount")
	receiptsUpdateCmd.Flags().Float64Var(&receiptTip, "tip", 0, "corrected tip amount")
	receiptsUpdateCmd.Flags().Float64Var(&receiptTotal, "total", 0, "corrected grand total")

	receiptsSettingsUpdateCmd.Flags().BoolVar(&receiptAutoCategorize, "auto-categorize", false, "auto-categorize and split matched transactions")
	receiptsSettingsUpdateCmd.Flags().BoolVar(&receiptUpdateNotes, "update-notes", false, "write receipt details into matched transaction notes")

	receiptsSettingsCmd.AddCommand(receiptsSettingsUpdateCmd)
	receiptsCmd.AddCommand(receiptsListCmd)
	receiptsCmd.AddCommand(receiptsShowCmd)
	receiptsCmd.AddCommand(receiptsDownloadCmd)
	receiptsCmd.AddCommand(receiptsUploadCmd)
	receiptsCmd.AddCommand(receiptsDeleteCmd)
	receiptsCmd.AddCommand(receiptsMatchCmd)
	receiptsCmd.AddCommand(receiptsUnmatchCmd)
	receiptsCmd.AddCommand(receiptsUpdateCmd)
	receiptsCmd.AddCommand(receiptsSettingsCmd)
	RootCmd.AddCommand(receiptsCmd)
}
