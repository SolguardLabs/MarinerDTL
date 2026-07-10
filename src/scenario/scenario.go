package scenario

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"marinerdtl/src/domain"
	"marinerdtl/src/engine"
	"marinerdtl/src/money"
	"marinerdtl/src/policy"
)

type PolicyConfig struct {
	Name                       *string       `json:"name"`
	MinimumDeposit             *money.Amount `json:"minimumDeposit"`
	MinimumEscrowFunding       *money.Amount `json:"minimumEscrowFunding"`
	MinimumMilestoneAmount     *money.Amount `json:"minimumMilestoneAmount"`
	MaximumRouteEscrow         *money.Amount `json:"maximumRouteEscrow"`
	MaximumMilestonesPerRoute  *int          `json:"maximumMilestonesPerRoute"`
	MaximumRebateBudget        *money.Amount `json:"maximumRebateBudget"`
	MaximumRebateBps           *int64        `json:"maximumRebateBps"`
	DefaultMinProgressBps      *int64        `json:"defaultMinProgressBps"`
	DefaultPenaltyBps          *int64        `json:"defaultPenaltyBps"`
	MaximumPenaltyBps          *int64        `json:"maximumPenaltyBps"`
	DisputeFreezeBps           *int64        `json:"disputeFreezeBps"`
	MinimumCertificateHashSize *int          `json:"minimumCertificateHashSize"`
	RequireCustodianRole       *bool         `json:"requireCustodianRole"`
	AllowPartialDisputes       *bool         `json:"allowPartialDisputes"`
	AllowRouteCancellation     *bool         `json:"allowRouteCancellation"`
}

func (c PolicyConfig) Apply(base policy.Policy) policy.Policy {
	if c.Name != nil {
		base.Name = *c.Name
	}
	if c.MinimumDeposit != nil {
		base.MinimumDeposit = *c.MinimumDeposit
	}
	if c.MinimumEscrowFunding != nil {
		base.MinimumEscrowFunding = *c.MinimumEscrowFunding
	}
	if c.MinimumMilestoneAmount != nil {
		base.MinimumMilestoneAmount = *c.MinimumMilestoneAmount
	}
	if c.MaximumRouteEscrow != nil {
		base.MaximumRouteEscrow = *c.MaximumRouteEscrow
	}
	if c.MaximumMilestonesPerRoute != nil {
		base.MaximumMilestonesPerRoute = *c.MaximumMilestonesPerRoute
	}
	if c.MaximumRebateBudget != nil {
		base.MaximumRebateBudget = *c.MaximumRebateBudget
	}
	if c.MaximumRebateBps != nil {
		base.MaximumRebateBps = *c.MaximumRebateBps
	}
	if c.DefaultMinProgressBps != nil {
		base.DefaultMinProgressBps = *c.DefaultMinProgressBps
	}
	if c.DefaultPenaltyBps != nil {
		base.DefaultPenaltyBps = *c.DefaultPenaltyBps
	}
	if c.MaximumPenaltyBps != nil {
		base.MaximumPenaltyBps = *c.MaximumPenaltyBps
	}
	if c.DisputeFreezeBps != nil {
		base.DisputeFreezeBps = *c.DisputeFreezeBps
	}
	if c.MinimumCertificateHashSize != nil {
		base.MinimumCertificateHashSize = *c.MinimumCertificateHashSize
	}
	if c.RequireCustodianRole != nil {
		base.RequireCustodianRole = *c.RequireCustodianRole
	}
	if c.AllowPartialDisputes != nil {
		base.AllowPartialDisputes = *c.AllowPartialDisputes
	}
	if c.AllowRouteCancellation != nil {
		base.AllowRouteCancellation = *c.AllowRouteCancellation
	}
	return base
}

type Step struct {
	Action string `json:"action"`

	ID                  string            `json:"id"`
	AccountID           string            `json:"accountId"`
	Role                string            `json:"role"`
	DisplayName         string            `json:"displayName"`
	Asset               string            `json:"asset"`
	Symbol              string            `json:"symbol"`
	Decimals            int               `json:"decimals"`
	RouteID             string            `json:"routeId"`
	MilestoneID         string            `json:"milestoneId"`
	CertificateID       string            `json:"certificateId"`
	DisputeID           string            `json:"disputeId"`
	ShipperID           string            `json:"shipperId"`
	OperatorID          string            `json:"operatorId"`
	RebateAccountID     string            `json:"rebateAccountId"`
	TreasuryID          string            `json:"treasuryId"`
	CarrierID           string            `json:"carrierId"`
	CustodianID         string            `json:"custodianId"`
	ClaimantID          string            `json:"claimantId"`
	OriginPort          string            `json:"originPort"`
	DestinationPort     string            `json:"destinationPort"`
	Vessel              string            `json:"vessel"`
	BillOfLading        string            `json:"billOfLading"`
	Leg                 string            `json:"leg"`
	Location            string            `json:"location"`
	Description         string            `json:"description"`
	DocumentHash        string            `json:"documentHash"`
	Reason              string            `json:"reason"`
	Status              string            `json:"status"`
	Resolution          string            `json:"resolution"`
	Memo                string            `json:"memo"`
	Sequence            int               `json:"sequence"`
	DueEpoch            int               `json:"dueEpoch"`
	Epochs              int               `json:"epochs"`
	To                  int               `json:"to"`
	Amount              money.Amount      `json:"amount"`
	RebateBudget        money.Amount      `json:"rebateBudget"`
	RebateBasisBps      int64             `json:"rebateBasisBps"`
	MinProgressBps      int64             `json:"minProgressBps"`
	PenaltyBps          int64             `json:"penaltyBps"`
	CancellationPenalty int64             `json:"cancellationPenaltyBps"`
	Metadata            map[string]string `json:"metadata"`
}

type Scenario struct {
	Name     string                  `json:"name"`
	Policy   PolicyConfig            `json:"policy"`
	Assets   []engine.AssetRequest   `json:"assets"`
	Accounts []engine.AccountRequest `json:"accounts"`
	Routes   []engine.RouteRequest   `json:"routes"`
	Steps    []Step                  `json:"steps"`
}

type Result struct {
	Service *engine.Service
	Report  domain.StateReport
}

func LoadFile(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var scenario Scenario
	if err := json.Unmarshal(data, &scenario); err != nil {
		return nil, err
	}
	if strings.TrimSpace(scenario.Name) == "" {
		scenario.Name = "MarinerDTL"
	}
	return &scenario, nil
}

func RunFile(path string, includeEvents bool) (*Result, error) {
	loaded, err := LoadFile(path)
	if err != nil {
		return nil, err
	}
	return loaded.Run(includeEvents)
}

func (s Scenario) Run(includeEvents bool) (*Result, error) {
	rules := s.Policy.Apply(policy.Default())
	service, err := engine.New(s.Name, rules)
	if err != nil {
		return nil, err
	}
	for _, asset := range s.Assets {
		if err := service.RegisterAsset(asset); err != nil {
			return nil, fmt.Errorf("asset %s: %w", asset.ID, err)
		}
	}
	for _, account := range s.Accounts {
		if err := service.RegisterAccount(account); err != nil {
			return nil, fmt.Errorf("account %s: %w", account.ID, err)
		}
	}
	for _, route := range s.Routes {
		if err := service.CreateRoute(route); err != nil {
			return nil, fmt.Errorf("route %s: %w", route.ID, err)
		}
	}
	for index, step := range s.Steps {
		if err := executeStep(service, step); err != nil {
			return nil, fmt.Errorf("step %d %s: %w", index+1, step.Action, err)
		}
	}
	return &Result{Service: service, Report: service.Report(includeEvents)}, nil
}

func executeStep(service *engine.Service, step Step) error {
	switch strings.ToLower(strings.TrimSpace(step.Action)) {
	case "asset", "register_asset":
		return service.RegisterAsset(engine.AssetRequest{
			ID:       step.ID,
			Symbol:   step.Symbol,
			Decimals: step.Decimals,
			Metadata: step.Metadata,
		})
	case "account", "register_account":
		id := firstNonEmpty(step.ID, step.AccountID)
		return service.RegisterAccount(engine.AccountRequest{
			ID:          id,
			Role:        step.Role,
			DisplayName: step.DisplayName,
			Metadata:    step.Metadata,
		})
	case "deposit", "fund_account":
		return service.Deposit(engine.DepositRequest{
			AccountID: step.AccountID,
			Asset:     step.Asset,
			Amount:    step.Amount,
			Memo:      step.Memo,
		})
	case "route", "create_route":
		id := firstNonEmpty(step.ID, step.RouteID)
		return service.CreateRoute(engine.RouteRequest{
			ID:                  id,
			Asset:               step.Asset,
			ShipperID:           step.ShipperID,
			OperatorID:          step.OperatorID,
			RebateAccountID:     step.RebateAccountID,
			OriginPort:          step.OriginPort,
			DestinationPort:     step.DestinationPort,
			Vessel:              step.Vessel,
			BillOfLading:        step.BillOfLading,
			RebateBudget:        step.RebateBudget,
			RebateBasisBps:      step.RebateBasisBps,
			MinProgressBps:      step.MinProgressBps,
			CancellationPenalty: step.CancellationPenalty,
			Metadata:            step.Metadata,
		})
	case "fund_escrow", "escrow":
		return service.FundEscrow(engine.FundingRequest{
			RouteID: step.RouteID,
			Amount:  step.Amount,
		})
	case "milestone", "add_milestone":
		id := firstNonEmpty(step.ID, step.MilestoneID)
		return service.AddMilestone(engine.MilestoneRequest{
			ID:          id,
			RouteID:     step.RouteID,
			Sequence:    step.Sequence,
			Leg:         step.Leg,
			Location:    step.Location,
			CarrierID:   step.CarrierID,
			CustodianID: step.CustodianID,
			Description: step.Description,
			Amount:      step.Amount,
			DueEpoch:    step.DueEpoch,
			PenaltyBps:  step.PenaltyBps,
			Metadata:    step.Metadata,
		})
	case "certificate", "issue_certificate":
		id := firstNonEmpty(step.ID, step.CertificateID)
		hash := step.DocumentHash
		if strings.TrimSpace(hash) == "" {
			hash = fmt.Sprintf("manual-%s-%s-%d", step.RouteID, step.MilestoneID, service.State().Epoch)
		}
		return service.IssueCertificate(engine.CertificateRequest{
			ID:           id,
			RouteID:      step.RouteID,
			MilestoneID:  step.MilestoneID,
			CustodianID:  step.CustodianID,
			DocumentHash: hash,
			Metadata:     step.Metadata,
		})
	case "release", "release_milestone":
		return service.ReleaseMilestone(engine.ReleaseRequest{
			RouteID:     step.RouteID,
			MilestoneID: step.MilestoneID,
			Amount:      step.Amount,
		})
	case "dispute", "open_dispute":
		id := firstNonEmpty(step.ID, step.DisputeID)
		return service.OpenDispute(engine.DisputeRequest{
			ID:          id,
			RouteID:     step.RouteID,
			MilestoneID: step.MilestoneID,
			ClaimantID:  step.ClaimantID,
			Amount:      step.Amount,
			Reason:      step.Reason,
			PenaltyBps:  step.PenaltyBps,
			Metadata:    step.Metadata,
		})
	case "resolve_dispute", "resolve":
		id := firstNonEmpty(step.ID, step.DisputeID)
		return service.ResolveDispute(engine.ResolveDisputeRequest{
			DisputeID:  id,
			Status:     step.Status,
			Resolution: step.Resolution,
		})
	case "cancel_milestone", "cancel":
		return service.CancelMilestone(engine.CancelMilestoneRequest{
			RouteID:     step.RouteID,
			MilestoneID: step.MilestoneID,
			PenaltyBps:  step.PenaltyBps,
		})
	case "cancel_route":
		return service.CancelRoute(engine.CancelRouteRequest{
			RouteID:    step.RouteID,
			PenaltyBps: step.PenaltyBps,
		})
	case "claim_rebate", "rebate":
		return service.ClaimRouteRebate(engine.RebateRequest{
			RouteID:    step.RouteID,
			TreasuryID: step.TreasuryID,
			Amount:     step.Amount,
		})
	case "advance", "advance_epoch":
		return service.Advance(engine.AdvanceRequest{Epochs: step.Epochs, To: step.To})
	case "validate":
		service.Validate()
		return nil
	default:
		return fmt.Errorf("unknown action %q", step.Action)
	}
}

func ValidateFile(path string) ([]domain.AuditIssue, error) {
	result, err := RunFile(path, false)
	if err != nil {
		return nil, err
	}
	return result.Service.Validate(), nil
}

func EncodeReport(report domain.StateReport, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(report, "", "  ")
	}
	return json.Marshal(report)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
