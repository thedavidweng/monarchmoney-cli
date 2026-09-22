package monarch

import (
	"context"
	"fmt"
	"time"

	"github.com/thedavidweng/monarchmoney-cli/internal/graphql"
	"github.com/thedavidweng/monarchmoney-cli/queries"
)

var GetInstitutionsQuery = queries.Get("institutions/list.graphql")
var GetCredentialSyncHealthQuery = queries.Get("institutions/health.graphql")

type Institution struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (s *Service) ListInstitutions(ctx context.Context) ([]Institution, error) {
	var resp struct {
		Credentials []struct {
			ID                             string `json:"id"`
			UpdateRequired                 bool   `json:"updateRequired"`
			DisconnectedFromDataProviderAt string `json:"disconnectedFromDataProviderAt"`
			DataProvider                   string `json:"dataProvider"`
			Institution                    struct {
				ID                 string `json:"id"`
				PlaidInstitutionID string `json:"plaidInstitutionId"`
				Name               string `json:"name"`
				Status             string `json:"status"`
			} `json:"institution"`
		} `json:"credentials"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "Web_GetInstitutionSettings",
		Query:         GetInstitutionsQuery,
	}, &resp)

	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	insts := make([]Institution, 0)
	for _, cred := range resp.Credentials {
		inst := cred.Institution
		if inst.Name != "" && !seen[inst.ID] {
			seen[inst.ID] = true
			insts = append(insts, Institution{
				ID:   inst.ID,
				Name: inst.Name,
				URL:  inst.PlaidInstitutionID,
			})
		}
	}

	return insts, nil
}

type ConnectionHealth struct {
	CredentialID       string   `json:"credential_id"`
	Institution        string   `json:"institution"`
	InstitutionURL     string   `json:"institution_url,omitempty"`
	DataProvider       string   `json:"data_provider,omitempty"`
	LastUpdated        string   `json:"last_updated,omitempty"`
	StaleDays          *int     `json:"stale_days,omitempty"`
	UpdateRequired     bool     `json:"update_required"`
	DisconnectedAt     string   `json:"disconnected_at,omitempty"`
	SyncDisabledAt     string   `json:"sync_disabled_at,omitempty"`
	SyncDisabledReason string   `json:"sync_disabled_reason,omitempty"`
	NeedsAttention     bool     `json:"needs_attention"`
	Reasons            []string `json:"reasons"`
	Accounts           []string `json:"accounts"`
}

type SyncHealthReport struct {
	ConnectionCount     int                `json:"connection_count"`
	NeedsAttentionCount int                `json:"needs_attention_count"`
	NeedsAttention      []ConnectionHealth `json:"needs_attention"`
	Connections         []ConnectionHealth `json:"connections"`
}

func (s *Service) GetConnectionHealth(ctx context.Context, staleAfterDays int) (*SyncHealthReport, error) {
	var resp struct {
		Credentials []struct {
			ID                             string `json:"id"`
			UpdateRequired                 bool   `json:"updateRequired"`
			DisconnectedFromDataProviderAt string `json:"disconnectedFromDataProviderAt"`
			SyncDisabledAt                 string `json:"syncDisabledAt"`
			SyncDisabledReason             string `json:"syncDisabledReason"`
			DataProvider                   string `json:"dataProvider"`
			DisplayLastUpdatedAt           string `json:"displayLastUpdatedAt"`
			Institution                    struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				URL  string `json:"url"`
			} `json:"institution"`
			Accounts []struct {
				ID          string `json:"id"`
				DisplayName string `json:"displayName"`
			} `json:"accounts"`
		} `json:"credentials"`
	}

	err := s.Client.Do(ctx, &graphql.Request{
		OperationName: "GetCredentialSyncHealth",
		Query:         GetCredentialSyncHealthQuery,
	}, &resp)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	report := &SyncHealthReport{}
	for i := range resp.Credentials {
		cred := &resp.Credentials[i]
		conn := ConnectionHealth{
			CredentialID: cred.ID, Institution: cred.Institution.Name,
			InstitutionURL: cred.Institution.URL, DataProvider: cred.DataProvider,
			LastUpdated: cred.DisplayLastUpdatedAt, UpdateRequired: cred.UpdateRequired,
			DisconnectedAt: cred.DisconnectedFromDataProviderAt,
			SyncDisabledAt: cred.SyncDisabledAt, SyncDisabledReason: cred.SyncDisabledReason,
		}
		if cred.DisplayLastUpdatedAt != "" {
			if ts, err := time.Parse(time.RFC3339, cred.DisplayLastUpdatedAt); err == nil {
				days := int(now.Sub(ts).Hours() / 24)
				conn.StaleDays = &days
			}
		}
		if cred.UpdateRequired {
			conn.Reasons = append(conn.Reasons, "credentials need re-authentication")
		}
		if cred.DisconnectedFromDataProviderAt != "" {
			conn.Reasons = append(conn.Reasons, "disconnected by the data provider")
		}
		if cred.SyncDisabledAt != "" {
			reason := "syncing disabled"
			if cred.SyncDisabledReason != "" {
				reason += ": " + cred.SyncDisabledReason
			}
			conn.Reasons = append(conn.Reasons, reason)
		}
		if conn.StaleDays != nil && *conn.StaleDays >= staleAfterDays {
			conn.Reasons = append(conn.Reasons, fmt.Sprintf("no successful update in %d days", *conn.StaleDays))
		}
		conn.NeedsAttention = len(conn.Reasons) > 0
		for _, a := range cred.Accounts {
			conn.Accounts = append(conn.Accounts, a.DisplayName)
		}
		if conn.Reasons == nil {
			conn.Reasons = []string{}
		}
		if conn.Accounts == nil {
			conn.Accounts = []string{}
		}
		report.Connections = append(report.Connections, conn)
		if conn.NeedsAttention {
			report.NeedsAttention = append(report.NeedsAttention, conn)
		}
	}
	if report.Connections == nil {
		report.Connections = []ConnectionHealth{}
	}
	if report.NeedsAttention == nil {
		report.NeedsAttention = []ConnectionHealth{}
	}
	report.ConnectionCount = len(report.Connections)
	report.NeedsAttentionCount = len(report.NeedsAttention)
	return report, nil
}
