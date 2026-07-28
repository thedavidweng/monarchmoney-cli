package monarch

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"
)

type FinancialOverview struct {
	AsOf             string           `json:"as_of"`
	NetWorth         float64          `json:"net_worth"`
	AccountCount     int              `json:"account_count"`
	Cashflow         *CashflowSummary `json:"cashflow"`
	Transactions     []Transaction    `json:"transactions"`
	TransactionTotal int              `json:"transaction_total"`
}

func (s *Service) GetFinancialOverview(ctx context.Context, startDate, endDate string) (*FinancialOverview, error) {
	if startDate == "" || endDate == "" {
		now := time.Now()
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
		endDate = time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	}

	var accounts []Account
	var cashflow *CashflowSummary
	var transactions []Transaction
	var txTotal int

	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		var err error
		accounts, err = s.ListAccounts(groupCtx)
		return err
	})
	group.Go(func() error {
		var err error
		cashflow, err = s.GetCashflowSummary(groupCtx, startDate, endDate)
		return err
	})
	group.Go(func() error {
		var err error
		transactions, txTotal, err = s.ListTransactions(groupCtx, &ListTransactionsOptions{
			StartDate: startDate,
			EndDate:   endDate,
			Limit:     10,
		})
		return err
	})
	if err := group.Wait(); err != nil {
		return nil, err
	}

	netWorth := 0.0
	visibleCount := 0
	for i := range accounts {
		a := &accounts[i]
		if a.DeactivatedAt != "" || a.IsHidden {
			continue
		}
		visibleCount++
		if a.IncludeInNetWorth || a.IncludeBalanceInNetWorth {
			if a.IsAsset {
				netWorth += a.DisplayBalance
			} else {
				netWorth -= a.DisplayBalance
			}
		}
	}

	return &FinancialOverview{
		AsOf:             time.Now().UTC().Format(time.RFC3339),
		NetWorth:         netWorth,
		AccountCount:     visibleCount,
		Cashflow:         cashflow,
		Transactions:     transactions,
		TransactionTotal: txTotal,
	}, nil
}
