package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/monarchmoney-cli/internal/cache"
	"github.com/thedavidweng/monarchmoney-cli/internal/errors"
	"github.com/thedavidweng/monarchmoney-cli/internal/hledger"
	"github.com/thedavidweng/monarchmoney-cli/internal/output"
)

var hledgerCmd = &cobra.Command{
	Use:     "hledger",
	Short:   "Plain-text ledger backups for hledger",
	GroupID: "utility",
	Example: "  monarch hledger backup",
}

var hledgerBackupCmd = &cobra.Command{
	Use:   "backup [FILE]",
	Short: "Regenerate a complete hledger journal from the local cache",
	Long: `Regenerate a complete hledger journal from the local cache.

The journal is rewritten from scratch on every run as a disposable derived
artifact with zero sync state; Monarch stays the source of truth. Keep
handwritten annotations in your own journal that includes this file.

The journal covers all accounts (including hidden and closed ones), the full
transaction history, closing balance assertions for every account, and
investment holdings as opening positions. Pending transactions are excluded.
Accounts whose cached history does not reconcile to their Monarch balance
(typical when an institution's feed predates your history) get a deterministic
opening-balance entry through equity:monarch:opening; audit those with
'hledger reg equity:monarch:opening'.

Reads only from the local cache, never the network. Run 'monarch cache sync
--all' first for archive-complete history. Configure 'backup_path' in your
config file to make every 'cache sync' regenerate this journal automatically.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		start := time.Now()
		renderer := output.NewRenderer(nil, nil, jsonMode, pretty)

		path := "./monarch.journal"
		if len(args) == 1 {
			path = args[0]
		}

		store, ok := openCache(renderer, "hledger.backup", start)
		if !ok {
			return
		}
		defer store.Close()

		result, warnings, err := writeJournal(store, path)
		if err != nil {
			handleError(renderer, "hledger.backup", errors.New(errors.InternalError, "failed to write journal", errors.CatInternal, false, err), start)
			return
		}

		if jsonMode {
			env := output.NewEnvelope("hledger.backup", profile, output.SchemaVersion, requestID, map[string]any{
				"status":       "backup complete",
				"file":         path,
				"accounts":     result.accounts,
				"transactions": result.transactions,
				"holdings":     result.holdings,
			}, time.Since(start))
			env.Meta.Warnings = warnings
			renderer.RenderSuccess(env)
		} else {
			fmt.Printf("Wrote %s (%d accounts, %d transactions, %d holdings).\n", path, result.accounts, result.transactions, result.holdings)
			for _, w := range warnings {
				fmt.Printf("Warning: %s\n", w)
			}
		}
	},
}

type backupResult struct {
	accounts     int
	transactions int
	holdings     int
}

func writeJournal(store *cache.Store, path string) (backupResult, []string, error) {
	accounts, err := store.Accounts()
	if err != nil {
		return backupResult{}, nil, err
	}
	txs, err := store.Transactions()
	if err != nil {
		return backupResult{}, nil, err
	}
	holdings, err := store.Holdings()
	if err != nil {
		return backupResult{}, nil, err
	}

	data := &hledger.Data{
		Accounts:     accounts,
		Transactions: txs,
		Holdings:     holdings,
		Anchor:       backupAnchor(store, txs),
	}
	journal := hledger.Generate(data)
	warnings := backupWarnings(store, data)

	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return backupResult{}, nil, err
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.WriteString(journal); err != nil {
		_ = tmp.Close()
		return backupResult{}, nil, err
	}
	if err := tmp.Close(); err != nil {
		return backupResult{}, nil, err
	}
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		return backupResult{}, nil, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return backupResult{}, nil, err
	}
	committed = true
	return backupResult{accounts: len(accounts), transactions: len(txs), holdings: len(holdings)}, warnings, nil
}

func backupWarnings(store *cache.Store, data *hledger.Data) []string {
	var warnings []string
	gaps := hledger.BalanceGaps(data)
	if len(gaps) > 0 {
		total := int64(0)
		for _, gap := range gaps {
			total += gap.Cents
		}
		warnings = append(warnings, fmt.Sprintf("%d account(s) have cached history that does not explain their balance (total %s); run 'monarch cache sync --all' to backfill before relying on this backup", len(gaps), hledger.FormatCents(total)))
	}
	if meta, err := store.LastSync(); err == nil && meta != nil {
		if age := time.Since(meta.SyncedAt); age > 7*24*time.Hour {
			warnings = append(warnings, fmt.Sprintf("cache was last synced %d days ago; run 'monarch cache sync' to capture recent activity", int(age.Hours()/24)))
		}
	}
	return warnings
}

func backupAnchor(store *cache.Store, txs []cache.Transaction) time.Time {
	var latest time.Time
	for i := range txs {
		if txs[i].Date.After(latest) {
			latest = txs[i].Date
		}
	}
	if !latest.IsZero() {
		return latest.AddDate(0, 0, 1)
	}
	meta, err := store.LastSync()
	if err != nil || meta == nil {
		return time.Time{}
	}
	y, m, d := meta.SyncedAt.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func init() {
	hledgerCmd.AddCommand(hledgerBackupCmd)
	RootCmd.AddCommand(hledgerCmd)
}
