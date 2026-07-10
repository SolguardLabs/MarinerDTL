package domain

import (
	"sort"

	"marinerdtl/src/money"
)

type AmountEntry struct {
	ID     string       `json:"id"`
	Amount money.Amount `json:"amount"`
}

type AccountReport struct {
	ID          string        `json:"id"`
	Role        AccountRole   `json:"role"`
	DisplayName string        `json:"displayName,omitempty"`
	Free        []AmountEntry `json:"free"`
	Reserved    []AmountEntry `json:"reserved"`
	Settled     []AmountEntry `json:"settled"`
	Rebates     []AmountEntry `json:"rebates"`
	Penalties   []AmountEntry `json:"penalties"`
}

type AssetReport struct {
	ID       string `json:"id"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

type RouteReport struct {
	ID                  string       `json:"id"`
	Asset               string       `json:"asset"`
	ShipperID           string       `json:"shipperId"`
	OperatorID          string       `json:"operatorId"`
	RebateAccountID     string       `json:"rebateAccountId"`
	OriginPort          string       `json:"originPort"`
	DestinationPort     string       `json:"destinationPort"`
	Vessel              string       `json:"vessel"`
	BillOfLading        string       `json:"billOfLading,omitempty"`
	Status              RouteStatus  `json:"status"`
	EscrowTotal         money.Amount `json:"escrowTotal"`
	EscrowRemaining     money.Amount `json:"escrowRemaining"`
	RebateBudget        money.Amount `json:"rebateBudget"`
	RebateClaimed       money.Amount `json:"rebateClaimed"`
	RebateBasisBps      int64        `json:"rebateBasisBps"`
	MinProgressBps      int64        `json:"minProgressBps"`
	CancellationPenalty int64        `json:"cancellationPenaltyBps"`
	DeliveredBps        int64        `json:"deliveredBps"`
	ReleasedAmount      money.Amount `json:"releasedAmount"`
	CancelledAmount     money.Amount `json:"cancelledAmount"`
	PenaltyAmount       money.Amount `json:"penaltyAmount"`
	FrozenAmount        money.Amount `json:"frozenAmount"`
	Milestones          []string     `json:"milestones"`
}

type MilestoneReport struct {
	ID              string          `json:"id"`
	RouteID         string          `json:"routeId"`
	Sequence        int             `json:"sequence"`
	Leg             string          `json:"leg"`
	Location        string          `json:"location"`
	CarrierID       string          `json:"carrierId"`
	CustodianID     string          `json:"custodianId"`
	Description     string          `json:"description,omitempty"`
	Amount          money.Amount    `json:"amount"`
	DueEpoch        int             `json:"dueEpoch"`
	PenaltyBps      int64           `json:"penaltyBps"`
	Status          MilestoneStatus `json:"status"`
	CertificateID   string          `json:"certificateId,omitempty"`
	DisputeID       string          `json:"disputeId,omitempty"`
	FrozenAmount    money.Amount    `json:"frozenAmount"`
	ReleasedAmount  money.Amount    `json:"releasedAmount"`
	CancelledAmount money.Amount    `json:"cancelledAmount"`
	PenaltyAmount   money.Amount    `json:"penaltyAmount"`
	RemainingAmount money.Amount    `json:"remainingAmount"`
}

type CertificateReport struct {
	ID            string            `json:"id"`
	RouteID       string            `json:"routeId"`
	MilestoneID   string            `json:"milestoneId"`
	CustodianID   string            `json:"custodianId"`
	DocumentHash  string            `json:"documentHash"`
	IssuedEpoch   int               `json:"issuedEpoch"`
	VerifiedEpoch int               `json:"verifiedEpoch"`
	Status        CertificateStatus `json:"status"`
}

type DisputeReport struct {
	ID             string        `json:"id"`
	RouteID        string        `json:"routeId"`
	MilestoneID    string        `json:"milestoneId"`
	ClaimantID     string        `json:"claimantId"`
	Reason         string        `json:"reason"`
	OpenedEpoch    int           `json:"openedEpoch"`
	ResolvedEpoch  int           `json:"resolvedEpoch,omitempty"`
	Status         DisputeStatus `json:"status"`
	AmountFrozen   money.Amount  `json:"amountFrozen"`
	PenaltyBps     int64         `json:"penaltyBps"`
	PenaltyCharged money.Amount  `json:"penaltyCharged"`
	Resolution     string        `json:"resolution,omitempty"`
}

type MetricReport struct {
	ID     string       `json:"id"`
	Amount money.Amount `json:"amount"`
}

type StateReport struct {
	Name         string              `json:"name"`
	Epoch        int                 `json:"epoch"`
	Assets       []AssetReport       `json:"assets"`
	Accounts     []AccountReport     `json:"accounts"`
	Routes       []RouteReport       `json:"routes"`
	Milestones   []MilestoneReport   `json:"milestones"`
	Certificates []CertificateReport `json:"certificates"`
	Disputes     []DisputeReport     `json:"disputes"`
	Metrics      []MetricReport      `json:"metrics"`
	Events       []Event             `json:"events,omitempty"`
	AuditIssues  []AuditIssue        `json:"auditIssues"`
}

type AuditIssue struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func (s *State) Report(includeEvents bool, issues []AuditIssue) StateReport {
	report := StateReport{
		Name:         s.Name,
		Epoch:        s.Epoch,
		Assets:       s.AssetReports(),
		Accounts:     s.AccountReports(),
		Routes:       s.RouteReports(),
		Milestones:   s.MilestoneReports(),
		Certificates: s.CertificateReports(),
		Disputes:     s.DisputeReports(),
		Metrics:      s.MetricReports(),
		AuditIssues:  append([]AuditIssue{}, issues...),
	}
	if includeEvents {
		report.Events = append([]Event(nil), s.Events...)
	}
	return report
}

func (s *State) AssetReports() []AssetReport {
	out := make([]AssetReport, 0, len(s.Assets))
	for _, id := range s.SortedAssetIDs() {
		asset := s.Assets[id]
		out = append(out, AssetReport{
			ID:       asset.ID,
			Symbol:   asset.Symbol,
			Decimals: asset.Decimals,
		})
	}
	return out
}

func (s *State) AccountReports() []AccountReport {
	out := make([]AccountReport, 0, len(s.Accounts))
	for _, id := range s.SortedAccountIDs() {
		account := s.Accounts[id]
		out = append(out, AccountReport{
			ID:          account.ID,
			Role:        account.Role,
			DisplayName: account.DisplayName,
			Free:        amountEntries(account.Free),
			Reserved:    amountEntries(account.Reserved),
			Settled:     amountEntries(account.Settled),
			Rebates:     amountEntries(account.Rebates),
			Penalties:   amountEntries(account.Penalties),
		})
	}
	return out
}

func (s *State) RouteReports() []RouteReport {
	out := make([]RouteReport, 0, len(s.Routes))
	for _, id := range s.SortedRouteIDs() {
		route := s.Routes[id]
		out = append(out, RouteReport{
			ID:                  route.ID,
			Asset:               route.Asset,
			ShipperID:           route.ShipperID,
			OperatorID:          route.OperatorID,
			RebateAccountID:     route.RebateAccountID,
			OriginPort:          route.OriginPort,
			DestinationPort:     route.DestinationPort,
			Vessel:              route.Vessel,
			BillOfLading:        route.BillOfLading,
			Status:              route.Status,
			EscrowTotal:         route.EscrowTotal,
			EscrowRemaining:     route.EscrowRemaining,
			RebateBudget:        route.RebateBudget,
			RebateClaimed:       route.RebateClaimed,
			RebateBasisBps:      route.RebateBasisBps,
			MinProgressBps:      route.MinProgressBps,
			CancellationPenalty: route.CancellationPenalty,
			DeliveredBps:        s.DeliveredProgressBps(route.ID),
			ReleasedAmount:      s.ReleasedAmount(route.ID),
			CancelledAmount:     s.CancelledAmount(route.ID),
			PenaltyAmount:       s.PenaltyAmount(route.ID),
			FrozenAmount:        s.FrozenAmount(route.ID),
			Milestones:          append([]string(nil), route.Milestones...),
		})
	}
	return out
}

func (s *State) MilestoneReports() []MilestoneReport {
	out := make([]MilestoneReport, 0, len(s.Milestones))
	for _, id := range s.SortedMilestoneIDs() {
		milestone := s.Milestones[id]
		out = append(out, MilestoneReport{
			ID:              milestone.ID,
			RouteID:         milestone.RouteID,
			Sequence:        milestone.Sequence,
			Leg:             milestone.Leg,
			Location:        milestone.Location,
			CarrierID:       milestone.CarrierID,
			CustodianID:     milestone.CustodianID,
			Description:     milestone.Description,
			Amount:          milestone.Amount,
			DueEpoch:        milestone.DueEpoch,
			PenaltyBps:      milestone.PenaltyBps,
			Status:          milestone.Status,
			CertificateID:   milestone.CertificateID,
			DisputeID:       milestone.DisputeID,
			FrozenAmount:    milestone.FrozenAmount,
			ReleasedAmount:  milestone.ReleasedAmount,
			CancelledAmount: milestone.CancelledAmount,
			PenaltyAmount:   milestone.PenaltyAmount,
			RemainingAmount: milestone.RemainingAmount(),
		})
	}
	return out
}

func (s *State) CertificateReports() []CertificateReport {
	out := make([]CertificateReport, 0, len(s.Certificates))
	for _, id := range s.SortedCertificateIDs() {
		certificate := s.Certificates[id]
		out = append(out, CertificateReport{
			ID:            certificate.ID,
			RouteID:       certificate.RouteID,
			MilestoneID:   certificate.MilestoneID,
			CustodianID:   certificate.CustodianID,
			DocumentHash:  certificate.DocumentHash,
			IssuedEpoch:   certificate.IssuedEpoch,
			VerifiedEpoch: certificate.VerifiedEpoch,
			Status:        certificate.Status,
		})
	}
	return out
}

func (s *State) DisputeReports() []DisputeReport {
	out := make([]DisputeReport, 0, len(s.Disputes))
	for _, id := range s.SortedDisputeIDs() {
		dispute := s.Disputes[id]
		out = append(out, DisputeReport{
			ID:             dispute.ID,
			RouteID:        dispute.RouteID,
			MilestoneID:    dispute.MilestoneID,
			ClaimantID:     dispute.ClaimantID,
			Reason:         dispute.Reason,
			OpenedEpoch:    dispute.OpenedEpoch,
			ResolvedEpoch:  dispute.ResolvedEpoch,
			Status:         dispute.Status,
			AmountFrozen:   dispute.AmountFrozen,
			PenaltyBps:     dispute.PenaltyBps,
			PenaltyCharged: dispute.PenaltyCharged,
			Resolution:     dispute.Resolution,
		})
	}
	return out
}

func (s *State) MetricReports() []MetricReport {
	out := make([]MetricReport, 0, len(s.Metrics))
	for _, entry := range amountEntries(s.Metrics) {
		out = append(out, MetricReport{ID: entry.ID, Amount: entry.Amount})
	}
	return out
}

func amountEntries(values map[string]money.Amount) []AmountEntry {
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if value == 0 {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]AmountEntry, 0, len(keys))
	for _, key := range keys {
		out = append(out, AmountEntry{ID: key, Amount: values[key]})
	}
	return out
}
