package engine

import "marinerdtl/src/money"

type AssetRequest struct {
	ID       string            `json:"id"`
	Symbol   string            `json:"symbol"`
	Decimals int               `json:"decimals"`
	Metadata map[string]string `json:"metadata"`
}

type AccountRequest struct {
	ID          string            `json:"id"`
	Role        string            `json:"role"`
	DisplayName string            `json:"displayName"`
	Metadata    map[string]string `json:"metadata"`
}

type DepositRequest struct {
	AccountID string       `json:"accountId"`
	Asset     string       `json:"asset"`
	Amount    money.Amount `json:"amount"`
	Memo      string       `json:"memo"`
}

type RouteRequest struct {
	ID                  string            `json:"id"`
	Asset               string            `json:"asset"`
	ShipperID           string            `json:"shipperId"`
	OperatorID          string            `json:"operatorId"`
	RebateAccountID     string            `json:"rebateAccountId"`
	OriginPort          string            `json:"originPort"`
	DestinationPort     string            `json:"destinationPort"`
	Vessel              string            `json:"vessel"`
	BillOfLading        string            `json:"billOfLading"`
	RebateBudget        money.Amount      `json:"rebateBudget"`
	RebateBasisBps      int64             `json:"rebateBasisBps"`
	MinProgressBps      int64             `json:"minProgressBps"`
	CancellationPenalty int64             `json:"cancellationPenaltyBps"`
	Metadata            map[string]string `json:"metadata"`
}

type FundingRequest struct {
	RouteID string       `json:"routeId"`
	Amount  money.Amount `json:"amount"`
}

type MilestoneRequest struct {
	ID          string            `json:"id"`
	RouteID     string            `json:"routeId"`
	Sequence    int               `json:"sequence"`
	Leg         string            `json:"leg"`
	Location    string            `json:"location"`
	CarrierID   string            `json:"carrierId"`
	CustodianID string            `json:"custodianId"`
	Description string            `json:"description"`
	Amount      money.Amount      `json:"amount"`
	DueEpoch    int               `json:"dueEpoch"`
	PenaltyBps  int64             `json:"penaltyBps"`
	Metadata    map[string]string `json:"metadata"`
}

type CertificateRequest struct {
	ID           string            `json:"id"`
	RouteID      string            `json:"routeId"`
	MilestoneID  string            `json:"milestoneId"`
	CustodianID  string            `json:"custodianId"`
	DocumentHash string            `json:"documentHash"`
	Metadata     map[string]string `json:"metadata"`
}

type ReleaseRequest struct {
	RouteID     string       `json:"routeId"`
	MilestoneID string       `json:"milestoneId"`
	Amount      money.Amount `json:"amount"`
}

type DisputeRequest struct {
	ID          string            `json:"id"`
	RouteID     string            `json:"routeId"`
	MilestoneID string            `json:"milestoneId"`
	ClaimantID  string            `json:"claimantId"`
	Amount      money.Amount      `json:"amount"`
	Reason      string            `json:"reason"`
	PenaltyBps  int64             `json:"penaltyBps"`
	Metadata    map[string]string `json:"metadata"`
}

type ResolveDisputeRequest struct {
	DisputeID  string       `json:"disputeId"`
	Status     string       `json:"status"`
	Resolution string       `json:"resolution"`
	Penalty    money.Amount `json:"penalty"`
}

type CancelMilestoneRequest struct {
	RouteID     string `json:"routeId"`
	MilestoneID string `json:"milestoneId"`
	PenaltyBps  int64  `json:"penaltyBps"`
}

type CancelRouteRequest struct {
	RouteID    string `json:"routeId"`
	PenaltyBps int64  `json:"penaltyBps"`
}

type RebateRequest struct {
	RouteID    string       `json:"routeId"`
	TreasuryID string       `json:"treasuryId"`
	Amount     money.Amount `json:"amount"`
}

type AdvanceRequest struct {
	Epochs int `json:"epochs"`
	To     int `json:"to"`
}
