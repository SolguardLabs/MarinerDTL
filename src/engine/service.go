package engine

import (
	"fmt"
	"strings"

	"marinerdtl/src/custody"
	"marinerdtl/src/domain"
	"marinerdtl/src/ids"
	"marinerdtl/src/ledger"
	"marinerdtl/src/policy"
	"marinerdtl/src/settlement"
)

type Service struct {
	state   *domain.State
	policy  policy.Policy
	journal *ledger.Journal
	issues  []domain.AuditIssue
}

func New(name string, rules policy.Policy) (*Service, error) {
	if strings.TrimSpace(name) == "" {
		name = "MarinerDTL"
	}
	if err := rules.Validate(); err != nil {
		return nil, err
	}
	return &Service{
		state:   domain.NewState(name),
		policy:  rules,
		journal: ledger.NewJournal(),
		issues:  []domain.AuditIssue{},
	}, nil
}

func MustNew(name string, rules policy.Policy) *Service {
	service, err := New(name, rules)
	if err != nil {
		panic(err)
	}
	return service
}

func (s *Service) State() *domain.State {
	return s.state
}

func (s *Service) Policy() policy.Policy {
	return s.policy
}

func (s *Service) Journal() *ledger.Journal {
	return s.journal
}

func (s *Service) RegisterAsset(req AssetRequest) error {
	id, err := ids.NewAssetID(req.ID)
	if err != nil {
		return invalid(err.Error())
	}
	if req.Decimals < 0 || req.Decimals > 18 {
		return invalid("asset decimals out of range")
	}
	asset := domain.NewAsset(id.String(), req.Symbol, req.Decimals)
	for key, value := range req.Metadata {
		asset.Metadata[key] = value
	}
	if err := s.state.AddAsset(asset); err != nil {
		return stateError(err.Error())
	}
	event := s.state.NextEvent(domain.EventAssetRegistered)
	event.Asset = asset.ID
	event.Message = "asset registered"
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) RegisterAccount(req AccountRequest) error {
	id, err := ids.NewAccountID(req.ID)
	if err != nil {
		return invalid(err.Error())
	}
	account := domain.NewAccount(id.String(), domain.AccountRoleFromString(req.Role))
	account.DisplayName = strings.TrimSpace(req.DisplayName)
	for key, value := range req.Metadata {
		account.Metadata[key] = value
	}
	if err := s.state.AddAccount(account); err != nil {
		return stateError(err.Error())
	}
	event := s.state.NextEvent(domain.EventAccountRegistered)
	event.AccountID = account.ID
	event.Message = "account registered"
	event.Fields["role"] = string(account.Role)
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) Deposit(req DepositRequest) error {
	accountID, err := ids.NewAccountID(req.AccountID)
	if err != nil {
		return invalid(err.Error())
	}
	assetID, err := ids.NewAssetID(req.Asset)
	if err != nil {
		return invalid(err.Error())
	}
	if err := s.policy.CheckDeposit(req.Amount); err != nil {
		return policyError(err.Error())
	}
	if _, err := s.state.RequireAsset(assetID.String()); err != nil {
		return notFound(err.Error())
	}
	account, err := s.state.RequireAccount(accountID.String())
	if err != nil {
		return notFound(err.Error())
	}
	memo := strings.TrimSpace(req.Memo)
	if memo == "" {
		memo = "account funded"
	}
	entries, err := ledger.Deposit(s.state.Epoch, account, assetID.String(), req.Amount, memo)
	if err != nil {
		return solvencyError(err.Error())
	}
	s.journal.AppendAll(entries)
	s.state.AddMetric("deposits", req.Amount)
	event := s.state.NextEvent(domain.EventDeposit)
	event.AccountID = account.ID
	event.Asset = assetID.String()
	event.Amount = req.Amount
	event.Message = memo
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) CreateRoute(req RouteRequest) error {
	routeID, err := ids.NewRouteID(req.ID)
	if err != nil {
		return invalid(err.Error())
	}
	assetID, err := ids.NewAssetID(req.Asset)
	if err != nil {
		return invalid(err.Error())
	}
	shipperID, err := ids.NewAccountID(req.ShipperID)
	if err != nil {
		return invalid(err.Error())
	}
	operatorID, err := ids.NewAccountID(req.OperatorID)
	if err != nil {
		return invalid(err.Error())
	}
	if _, err := s.state.RequireAsset(assetID.String()); err != nil {
		return notFound(err.Error())
	}
	if _, err := s.state.RequireRole(shipperID.String(), domain.RoleShipper); err != nil {
		return invalid(err.Error())
	}
	if _, err := s.state.RequireRole(operatorID.String(), domain.RoleOperator); err != nil {
		return invalid(err.Error())
	}
	rebateAccountID := operatorID
	if strings.TrimSpace(req.RebateAccountID) != "" {
		rebateAccountID, err = ids.NewAccountID(req.RebateAccountID)
		if err != nil {
			return invalid(err.Error())
		}
		if _, err := s.state.RequireAccount(rebateAccountID.String()); err != nil {
			return notFound(err.Error())
		}
	}
	if err := s.policy.CheckRebateBudget(req.RebateBudget); err != nil {
		return policyError(err.Error())
	}
	if err := s.policy.CheckRebateBps(req.RebateBasisBps); err != nil {
		return policyError(err.Error())
	}
	minProgress, err := s.policy.NormalizeProgressBps(req.MinProgressBps)
	if err != nil {
		return policyError(err.Error())
	}
	penaltyBps, err := s.policy.NormalizePenaltyBps(req.CancellationPenalty)
	if err != nil {
		return policyError(err.Error())
	}
	route := domain.NewRoute(routeID.String(), assetID.String(), shipperID.String(), operatorID.String(), s.state.Epoch)
	route.RebateAccountID = rebateAccountID.String()
	route.OriginPort = strings.TrimSpace(req.OriginPort)
	route.DestinationPort = strings.TrimSpace(req.DestinationPort)
	route.Vessel = strings.TrimSpace(req.Vessel)
	route.BillOfLading = strings.TrimSpace(req.BillOfLading)
	route.RebateBudget = req.RebateBudget
	route.RebateBasisBps = req.RebateBasisBps
	route.MinProgressBps = minProgress
	route.CancellationPenalty = penaltyBps
	for key, value := range req.Metadata {
		route.Metadata[key] = value
	}
	if err := s.state.AddRoute(route); err != nil {
		return stateError(err.Error())
	}
	event := s.state.NextEvent(domain.EventRouteCreated)
	event.RouteID = route.ID
	event.Asset = route.Asset
	event.Message = "route created"
	event.Fields["origin"] = route.OriginPort
	event.Fields["destination"] = route.DestinationPort
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) FundEscrow(req FundingRequest) error {
	routeID, err := ids.NewRouteID(req.RouteID)
	if err != nil {
		return invalid(err.Error())
	}
	route, err := s.state.RequireRoute(routeID.String())
	if err != nil {
		return notFound(err.Error())
	}
	if !route.CanMutate() {
		return stateError("route %s cannot be funded in status %s", route.ID, route.Status)
	}
	if err := s.policy.CheckFunding(route, req.Amount); err != nil {
		return policyError(err.Error())
	}
	shipper, err := s.state.RequireAccount(route.ShipperID)
	if err != nil {
		return notFound(err.Error())
	}
	entries, err := ledger.FundEscrow(s.state.Epoch, shipper, route, req.Amount)
	if err != nil {
		return solvencyError(err.Error())
	}
	s.journal.AppendAll(entries)
	s.state.AddMetric("escrowFunded", req.Amount)
	event := s.state.NextEvent(domain.EventEscrowFunded)
	event.RouteID = route.ID
	event.AccountID = shipper.ID
	event.Asset = route.Asset
	event.Amount = req.Amount
	event.Message = "route escrow funded"
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) AddMilestone(req MilestoneRequest) error {
	milestoneID, err := ids.NewMilestoneID(req.ID)
	if err != nil {
		return invalid(err.Error())
	}
	routeID, err := ids.NewRouteID(req.RouteID)
	if err != nil {
		return invalid(err.Error())
	}
	carrierID, err := ids.NewAccountID(req.CarrierID)
	if err != nil {
		return invalid(err.Error())
	}
	custodianID, err := ids.NewAccountID(req.CustodianID)
	if err != nil {
		return invalid(err.Error())
	}
	route, err := s.state.RequireRoute(routeID.String())
	if err != nil {
		return notFound(err.Error())
	}
	if !route.CanMutate() {
		return stateError("route %s cannot accept milestones in status %s", route.ID, route.Status)
	}
	if err := s.policy.CheckMilestone(route, req.Amount); err != nil {
		return policyError(err.Error())
	}
	if _, err := s.state.RequireRole(carrierID.String(), domain.RoleCarrier, domain.RoleOperator); err != nil {
		return invalid(err.Error())
	}
	registry := custody.NewRegistry(s.state, s.policy)
	if _, err := registry.ValidateCustodian(custodianID.String()); err != nil {
		return invalid(err.Error())
	}
	penaltyBps, err := s.policy.NormalizePenaltyBps(req.PenaltyBps)
	if err != nil {
		return policyError(err.Error())
	}
	sequence := req.Sequence
	if sequence == 0 {
		sequence = len(route.Milestones) + 1
	}
	milestone := domain.NewMilestone(milestoneID.String(), route.ID, sequence)
	milestone.Leg = strings.TrimSpace(req.Leg)
	milestone.Location = strings.TrimSpace(req.Location)
	milestone.CarrierID = carrierID.String()
	milestone.CustodianID = custodianID.String()
	milestone.Description = strings.TrimSpace(req.Description)
	milestone.Amount = req.Amount
	milestone.DueEpoch = req.DueEpoch
	milestone.PenaltyBps = penaltyBps
	for key, value := range req.Metadata {
		milestone.Metadata[key] = value
	}
	if err := s.state.AddMilestone(milestone); err != nil {
		return stateError(err.Error())
	}
	entries, err := ledger.LockMilestone(s.state.Epoch, route, milestone)
	if err != nil {
		return solvencyError(err.Error())
	}
	s.journal.AppendAll(entries)
	event := s.state.NextEvent(domain.EventMilestoneAdded)
	event.RouteID = route.ID
	event.MilestoneID = milestone.ID
	event.AccountID = carrierID.String()
	event.Asset = route.Asset
	event.Amount = milestone.Amount
	event.Message = "milestone added"
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) IssueCertificate(req CertificateRequest) error {
	certificateID, err := ids.NewCertificateID(req.ID)
	if err != nil {
		return invalid(err.Error())
	}
	routeID, err := ids.NewRouteID(req.RouteID)
	if err != nil {
		return invalid(err.Error())
	}
	milestoneID, err := ids.NewMilestoneID(req.MilestoneID)
	if err != nil {
		return invalid(err.Error())
	}
	custodianID, err := ids.NewAccountID(req.CustodianID)
	if err != nil {
		return invalid(err.Error())
	}
	route, err := s.state.RequireRoute(routeID.String())
	if err != nil {
		return notFound(err.Error())
	}
	milestone, err := s.state.RequireMilestone(milestoneID.String())
	if err != nil {
		return notFound(err.Error())
	}
	documentHash := custody.NormalizeHash(req.DocumentHash)
	registry := custody.NewRegistry(s.state, s.policy)
	if err := registry.ValidateCertificateDraft(route, milestone, custodianID.String(), documentHash); err != nil {
		return invalid(err.Error())
	}
	certificate := domain.NewCertificate(certificateID.String(), route.ID, milestone.ID, custodianID.String(), documentHash, s.state.Epoch)
	for key, value := range req.Metadata {
		certificate.Metadata[key] = value
	}
	certificate.Verify(s.state.Epoch)
	if err := s.state.AddCertificate(certificate); err != nil {
		return stateError(err.Error())
	}
	milestone.MarkCertified(certificate.ID)
	event := s.state.NextEvent(domain.EventCertificateIssued)
	event.RouteID = route.ID
	event.MilestoneID = milestone.ID
	event.AccountID = custodianID.String()
	event.Message = "certificate issued"
	event.Fields["certificate"] = certificate.ID
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) ReleaseMilestone(req ReleaseRequest) error {
	routeID, err := ids.NewRouteID(req.RouteID)
	if err != nil {
		return invalid(err.Error())
	}
	milestoneID, err := ids.NewMilestoneID(req.MilestoneID)
	if err != nil {
		return invalid(err.Error())
	}
	route, err := s.state.RequireRoute(routeID.String())
	if err != nil {
		return notFound(err.Error())
	}
	milestone, err := s.state.RequireMilestone(milestoneID.String())
	if err != nil {
		return notFound(err.Error())
	}
	plan, err := settlement.BuildReleasePlan(route, milestone, req.Amount)
	if err != nil {
		return stateError(err.Error())
	}
	shipper, err := s.state.RequireAccount(route.ShipperID)
	if err != nil {
		return notFound(err.Error())
	}
	carrier, err := s.state.RequireAccount(plan.CarrierID)
	if err != nil {
		return notFound(err.Error())
	}
	registry := custody.NewRegistry(s.state, s.policy)
	if certificate, err := s.state.RequireCertificate(milestone.CertificateID); err != nil {
		return stateError(err.Error())
	} else if err := registry.VerifyCertificate(certificate, route, milestone); err != nil {
		return invalid(err.Error())
	}
	entries, err := ledger.ReleaseMilestone(s.state.Epoch, shipper, carrier, route, milestone, plan.Amount)
	if err != nil {
		return solvencyError(err.Error())
	}
	s.journal.AppendAll(entries)
	s.state.AddMetric("released", plan.Amount)
	if s.policy.ShouldCompleteRoute(s.state, route) {
		route.Status = domain.RouteCompleted
		route.ClosedEpoch = s.state.Epoch
	}
	event := s.state.NextEvent(domain.EventMilestoneReleased)
	event.RouteID = route.ID
	event.MilestoneID = milestone.ID
	event.AccountID = carrier.ID
	event.Asset = route.Asset
	event.Amount = plan.Amount
	event.Message = "milestone released"
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) OpenDispute(req DisputeRequest) error {
	disputeID, err := ids.NewDisputeID(req.ID)
	if err != nil {
		return invalid(err.Error())
	}
	routeID, err := ids.NewRouteID(req.RouteID)
	if err != nil {
		return invalid(err.Error())
	}
	milestoneID, err := ids.NewMilestoneID(req.MilestoneID)
	if err != nil {
		return invalid(err.Error())
	}
	claimantID, err := ids.NewAccountID(req.ClaimantID)
	if err != nil {
		return invalid(err.Error())
	}
	route, err := s.state.RequireRoute(routeID.String())
	if err != nil {
		return notFound(err.Error())
	}
	milestone, err := s.state.RequireMilestone(milestoneID.String())
	if err != nil {
		return notFound(err.Error())
	}
	claimant, err := s.state.RequireAccount(claimantID.String())
	if err != nil {
		return notFound(err.Error())
	}
	plan, err := settlement.BuildDisputePlan(route, milestone, claimant, req.Amount, req.Reason, s.policy)
	if err != nil {
		return stateError(err.Error())
	}
	dispute := domain.NewDispute(disputeID.String(), route.ID, milestone.ID, claimant.ID, s.state.Epoch)
	dispute.AmountFrozen = plan.FreezeAmount
	dispute.PenaltyBps = plan.PenaltyBps
	dispute.Reason = strings.TrimSpace(req.Reason)
	if req.PenaltyBps > 0 {
		dispute.PenaltyBps = req.PenaltyBps
	}
	for key, value := range req.Metadata {
		dispute.Metadata[key] = value
	}
	if err := s.state.AddDispute(dispute); err != nil {
		return stateError(err.Error())
	}
	entries, err := ledger.MarkDispute(s.state.Epoch, route, milestone, dispute)
	if err != nil {
		return stateError(err.Error())
	}
	s.journal.AppendAll(entries)
	s.state.AddMetric("disputed", dispute.AmountFrozen)
	event := s.state.NextEvent(domain.EventDisputeOpened)
	event.RouteID = route.ID
	event.MilestoneID = milestone.ID
	event.AccountID = claimant.ID
	event.Asset = route.Asset
	event.Amount = dispute.AmountFrozen
	event.Message = "dispute opened"
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) ResolveDispute(req ResolveDisputeRequest) error {
	disputeID, err := ids.NewDisputeID(req.DisputeID)
	if err != nil {
		return invalid(err.Error())
	}
	dispute, err := s.state.RequireDispute(disputeID.String())
	if err != nil {
		return notFound(err.Error())
	}
	if !dispute.IsOpen() {
		return stateError("dispute %s is not open", dispute.ID)
	}
	route, err := s.state.RequireRoute(dispute.RouteID)
	if err != nil {
		return notFound(err.Error())
	}
	milestone, err := s.state.RequireMilestone(dispute.MilestoneID)
	if err != nil {
		return notFound(err.Error())
	}
	status, err := settlement.ResolutionStatus(strings.ToLower(strings.TrimSpace(req.Status)))
	if err != nil {
		return invalid(err.Error())
	}
	entries, err := ledger.ResolveDispute(s.state.Epoch, route, milestone, dispute, status, req.Resolution)
	if err != nil {
		return stateError(err.Error())
	}
	s.journal.AppendAll(entries)
	event := s.state.NextEvent(domain.EventDisputeResolved)
	event.RouteID = route.ID
	event.MilestoneID = milestone.ID
	event.AccountID = dispute.ClaimantID
	event.Asset = route.Asset
	event.Amount = dispute.AmountFrozen
	event.Message = "dispute resolved"
	event.Fields["status"] = string(status)
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) CancelMilestone(req CancelMilestoneRequest) error {
	routeID, err := ids.NewRouteID(req.RouteID)
	if err != nil {
		return invalid(err.Error())
	}
	milestoneID, err := ids.NewMilestoneID(req.MilestoneID)
	if err != nil {
		return invalid(err.Error())
	}
	route, err := s.state.RequireRoute(routeID.String())
	if err != nil {
		return notFound(err.Error())
	}
	milestone, err := s.state.RequireMilestone(milestoneID.String())
	if err != nil {
		return notFound(err.Error())
	}
	plan, err := settlement.BuildCancellationPlan(route, milestone, s.policy, req.PenaltyBps)
	if err != nil {
		return stateError(err.Error())
	}
	shipper, err := s.state.RequireAccount(route.ShipperID)
	if err != nil {
		return notFound(err.Error())
	}
	beneficiary, err := s.state.RequireAccount(plan.Beneficiary)
	if err != nil {
		return notFound(err.Error())
	}
	entries, err := ledger.CancelRemainder(s.state.Epoch, shipper, beneficiary, route, milestone, plan.Refund, plan.Penalty)
	if err != nil {
		return solvencyError(err.Error())
	}
	s.journal.AppendAll(entries)
	s.state.AddMetric("cancelled", plan.Refund)
	s.state.AddMetric("penalties", plan.Penalty)
	event := s.state.NextEvent(domain.EventMilestoneCancelled)
	event.RouteID = route.ID
	event.MilestoneID = milestone.ID
	event.AccountID = beneficiary.ID
	event.Asset = route.Asset
	event.Amount = plan.Refund.MustAdd(plan.Penalty)
	event.Message = "milestone cancelled"
	event.Fields["refund"] = plan.Refund.String()
	event.Fields["penalty"] = plan.Penalty.String()
	s.state.AppendEvent(event)
	progress := settlement.CalculateProgress(route, s.state.RouteMilestones(route.ID))
	if progress.IsFullyTerminal() {
		route.Status = domain.RouteCancelled
		route.ClosedEpoch = s.state.Epoch
	}
	return nil
}

func (s *Service) CancelRoute(req CancelRouteRequest) error {
	routeID, err := ids.NewRouteID(req.RouteID)
	if err != nil {
		return invalid(err.Error())
	}
	route, err := s.state.RequireRoute(routeID.String())
	if err != nil {
		return notFound(err.Error())
	}
	if route.Status == domain.RouteCompleted || route.Status == domain.RouteCancelled {
		return stateError("route %s is terminal", route.ID)
	}
	for _, milestone := range s.state.RouteMilestones(route.ID) {
		if milestone.Status == domain.MilestoneReleased || milestone.Status == domain.MilestoneCancelled {
			continue
		}
		if err := s.CancelMilestone(CancelMilestoneRequest{
			RouteID:     route.ID,
			MilestoneID: milestone.ID,
			PenaltyBps:  req.PenaltyBps,
		}); err != nil {
			return err
		}
	}
	route.Status = domain.RouteCancelled
	route.ClosedEpoch = s.state.Epoch
	event := s.state.NextEvent(domain.EventRouteCancelled)
	event.RouteID = route.ID
	event.Asset = route.Asset
	event.Message = "route cancelled"
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) ClaimRouteRebate(req RebateRequest) error {
	routeID, err := ids.NewRouteID(req.RouteID)
	if err != nil {
		return invalid(err.Error())
	}
	treasuryID, err := ids.NewAccountID(req.TreasuryID)
	if err != nil {
		return invalid(err.Error())
	}
	route, err := s.state.RequireRoute(routeID.String())
	if err != nil {
		return notFound(err.Error())
	}
	plan, err := settlement.BuildRebatePlan(route, s.state.RouteMilestones(route.ID), treasuryID.String(), s.policy)
	if err != nil {
		return policyError(err.Error())
	}
	amount := plan.ClaimableAmount
	if req.Amount > 0 {
		if req.Amount > plan.ClaimableAmount {
			return policyError("rebate request exceeds claimable amount")
		}
		amount = req.Amount
	}
	treasury, err := s.state.RequireRole(plan.TreasuryID, domain.RoleTreasury)
	if err != nil {
		return invalid(err.Error())
	}
	recipient, err := s.state.RequireAccount(plan.RecipientID)
	if err != nil {
		return notFound(err.Error())
	}
	entries, err := ledger.PayRebate(s.state.Epoch, treasury, recipient, route, amount)
	if err != nil {
		return solvencyError(err.Error())
	}
	s.journal.AppendAll(entries)
	s.state.AddMetric("rebates", amount)
	event := s.state.NextEvent(domain.EventRebateClaimed)
	event.RouteID = route.ID
	event.AccountID = recipient.ID
	event.Asset = route.Asset
	event.Amount = amount
	event.Message = "route rebate claimed"
	event.Fields["progressBps"] = fmt.Sprintf("%d", plan.ProgressBps)
	s.state.AppendEvent(event)
	return nil
}

func (s *Service) Advance(req AdvanceRequest) error {
	if req.To > 0 {
		return s.state.Advance(req.To)
	}
	if req.Epochs <= 0 {
		req.Epochs = 1
	}
	for i := 0; i < req.Epochs; i++ {
		if err := s.state.Advance(0); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Validate() []domain.AuditIssue {
	issues := AuditState(s.state)
	s.issues = issues
	return append([]domain.AuditIssue{}, issues...)
}

func (s *Service) Report(includeEvents bool) domain.StateReport {
	issues := s.Validate()
	return s.state.Report(includeEvents, issues)
}
