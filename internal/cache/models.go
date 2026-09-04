package cache

import (
	"database/sql"
	"errors"
	"time"
)

type Account struct {
	ID             string
	DisplayName    string
	AccountType    string
	TypeGroup      string
	DisplayBalance float64
	CurrentBalance float64
	IsManual       bool
	IsHidden       bool
	IsClosed       bool
	UpdatedAt      time.Time
}

type Tag struct {
	ID   string
	Name string
}

type Split struct {
	ID       string
	Amount   float64
	Category string
	Merchant string
	Notes    string
}

type Transaction struct {
	ID                  string
	Date                time.Time
	Amount              float64
	Merchant            string
	PlaidName           string
	ProviderDescription string
	Category            string
	CategoryGroup       string
	CategoryGroupType   string
	Notes               string
	Pending             bool
	HideFromReports     bool
	IsRecurring         bool
	ReviewStatus        string
	NeedsReview         bool
	GoalID              string
	GoalName            string
	AccountID           string
	Tags                []Tag
	Splits              []Split
}

type Holding struct {
	ID        string
	Ticker    string
	Name      string
	Quantity  float64
	Basis     float64
	Value     float64
	AccountID string
}

type SyncMeta struct {
	ID       uint      `json:"-"`
	SyncedAt time.Time `json:"synced_at"`
	Accounts int       `json:"accounts"`
	TxCount  int       `json:"transactions"`
}

var ErrSchemaOutdated = errors.New("cache predates the archive-replica schema; run 'monarch cache sync' to rebuild it")

const schema = `
CREATE TABLE IF NOT EXISTS accounts (
	id              TEXT PRIMARY KEY,
	display_name    TEXT,
	account_type    TEXT,
	type_group      TEXT,
	display_balance REAL,
	current_balance REAL,
	is_manual       INTEGER,
	is_hidden       INTEGER,
	is_closed       INTEGER,
	updated_at      TEXT
);
CREATE TABLE IF NOT EXISTS transactions (
	id                   TEXT PRIMARY KEY,
	date                 TEXT,
	amount               REAL,
	merchant             TEXT,
	plaid_name           TEXT,
	provider_description TEXT,
	category             TEXT,
	category_group       TEXT,
	category_group_type  TEXT,
	notes                TEXT,
	hide_from_reports    INTEGER,
	is_recurring         INTEGER,
	pending              INTEGER,
	review_status        TEXT,
	needs_review         INTEGER,
	goal_id              TEXT,
	goal_name            TEXT,
	account_id           TEXT
);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);
CREATE INDEX IF NOT EXISTS idx_transactions_merchant ON transactions(merchant);
CREATE INDEX IF NOT EXISTS idx_transactions_category ON transactions(category);
CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON transactions(account_id);
CREATE TABLE IF NOT EXISTS transaction_tags (
	transaction_id TEXT,
	tag_id         TEXT,
	name           TEXT,
	PRIMARY KEY (transaction_id, tag_id)
);
CREATE TABLE IF NOT EXISTS transaction_splits (
	id             TEXT PRIMARY KEY,
	transaction_id TEXT,
	amount         REAL,
	category       TEXT,
	merchant       TEXT,
	notes          TEXT
);
CREATE INDEX IF NOT EXISTS idx_transaction_splits_transaction_id ON transaction_splits(transaction_id);
CREATE TABLE IF NOT EXISTS holdings (
	id         TEXT PRIMARY KEY,
	ticker     TEXT,
	name       TEXT,
	quantity   REAL,
	basis      REAL,
	value      REAL,
	account_id TEXT
);
CREATE INDEX IF NOT EXISTS idx_holdings_account_id ON holdings(account_id);
CREATE TABLE IF NOT EXISTS sync_meta (
	id        INTEGER PRIMARY KEY AUTOINCREMENT,
	synced_at TEXT,
	accounts  INTEGER,
	tx_count  INTEGER
);
`

func Migrate(db *sql.DB) error {
	legacy, err := isLegacySchema(db)
	if err != nil {
		return err
	}
	if legacy {
		return ErrSchemaOutdated
	}
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	return addMissingColumns(db)
}

func addMissingColumns(db *sql.DB) error {
	for _, col := range []string{"hide_from_reports", "is_recurring"} {
		var present int
		if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('transactions') WHERE name = ?`, col).Scan(&present); err != nil {
			return err
		}
		if present == 0 {
			if _, err := db.Exec(`ALTER TABLE transactions ADD COLUMN ` + col + ` INTEGER`); err != nil {
				return err
			}
		}
	}
	return nil
}

func Rebuild(db *sql.DB) error {
	legacy, err := isLegacySchema(db)
	if err != nil {
		return err
	}
	if legacy {
		if _, err := db.Exec(`DROP TABLE IF EXISTS accounts; DROP TABLE IF EXISTS transactions; DROP TABLE IF EXISTS transaction_tags; DROP TABLE IF EXISTS transaction_splits; DROP TABLE IF EXISTS sync_meta;`); err != nil {
			return err
		}
	}
	_, err = db.Exec(schema)
	return err
}

func isLegacySchema(db *sql.DB) (bool, error) {
	var dataTables int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name IN ('accounts', 'transactions')`).Scan(&dataTables); err != nil {
		return false, err
	}
	if dataTables == 0 {
		return false, nil
	}
	var archiveTables int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'holdings'`).Scan(&archiveTables); err != nil {
		return false, err
	}
	return archiveTables == 0, nil
}
