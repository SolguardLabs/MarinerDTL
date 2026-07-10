package domain

import (
	"fmt"
	"sort"
	"strings"

	"marinerdtl/src/money"
)

type Asset struct {
	ID       string
	Symbol   string
	Decimals int
	Metadata map[string]string
}

func NewAsset(id, symbol string, decimals int) *Asset {
	if strings.TrimSpace(symbol) == "" {
		symbol = strings.ToUpper(id)
	}
	return &Asset{
		ID:       strings.TrimSpace(id),
		Symbol:   strings.ToUpper(strings.TrimSpace(symbol)),
		Decimals: decimals,
		Metadata: map[string]string{},
	}
}

type Route struct {
	ID                  string
	Asset               string
	ShipperID           string
	OperatorID          string
	RebateAccountID     string
	OriginPort          string
	DestinationPort     string
	Vessel              string
	BillOfLading        string
	Status              RouteStatus
	CreatedEpoch        int
	ClosedEpoch         int
	EscrowTotal         money.Amount
	EscrowRemaining     money.Amount
	RebateBudget        money.Amount
	RebateClaimed       money.Amount
	RebateBasisBps      int64
	MinProgressBps      int64
	CancellationPenalty int64
	Milestones          []string
	Metadata            map[string]string
}

func NewRoute(id, asset, shipper, operator string, epoch int) *Route {
	return &Route{
		ID:              strings.TrimSpace(id),
		Asset:           strings.TrimSpace(asset),
		ShipperID:       strings.TrimSpace(shipper),
		OperatorID:      strings.TrimSpace(operator),
		RebateAccountID: strings.TrimSpace(operator),
		Status:          RoutePlanned,
		CreatedEpoch:    epoch,
		RebateBasisBps:  0,
		MinProgressBps:  10000,
		Milestones:      []string{},
		Metadata:        map[string]string{},
	}
}

func (r *Route) IsTerminal() bool {
	return r.Status == RouteCompleted || r.Status == RouteCancelled
}

func (r *Route) CanMutate() bool {
	return !r.IsTerminal() && r.Status != RouteSuspended
}

func (r *Route) AddMilestone(id string) {
	for _, existing := range r.Milestones {
		if existing == id {
			return
		}
	}
	r.Milestones = append(r.Milestones, id)
	sort.Strings(r.Milestones)
}

func (r *Route) Fund(amount money.Amount) error {
	if err := amount.Validate(); err != nil {
		return err
	}
	total, err := r.EscrowTotal.Add(amount)
	if err != nil {
		return err
	}
	remaining, err := r.EscrowRemaining.Add(amount)
	if err != nil {
		return err
	}
	r.EscrowTotal = total
	r.EscrowRemaining = remaining
	if r.Status == RoutePlanned {
		r.Status = RouteFunded
	}
	return nil
}

func (r *Route) DebitEscrow(amount money.Amount) error {
	if err := amount.Validate(); err != nil {
		return err
	}
	next, err := r.EscrowRemaining.Sub(amount)
	if err != nil {
		return fmt.Errorf("route %s escrow cannot cover %s", r.ID, amount)
	}
	r.EscrowRemaining = next
	return nil
}

func (r *Route) CreditRebateClaim(amount money.Amount) error {
	next, err := r.RebateClaimed.Add(amount)
	if err != nil {
		return err
	}
	if next > r.RebateBudget {
		return fmt.Errorf("route %s rebate budget exceeded", r.ID)
	}
	r.RebateClaimed = next
	return nil
}

func (r *Route) OutstandingRebate() money.Amount {
	if r.RebateBudget <= r.RebateClaimed {
		return 0
	}
	return r.RebateBudget - r.RebateClaimed
}

func (r *Route) Clone() *Route {
	copy := *r
	copy.Milestones = append([]string(nil), r.Milestones...)
	copy.Metadata = map[string]string{}
	for key, value := range r.Metadata {
		copy.Metadata[key] = value
	}
	return &copy
}

type Milestone struct {
	ID              string
	RouteID         string
	Sequence        int
	Leg             string
	Location        string
	CarrierID       string
	CustodianID     string
	Description     string
	Amount          money.Amount
	DueEpoch        int
	PenaltyBps      int64
	Status          MilestoneStatus
	CertificateID   string
	DisputeID       string
	FrozenAmount    money.Amount
	ReleasedAmount  money.Amount
	CancelledAmount money.Amount
	PenaltyAmount   money.Amount
	Metadata        map[string]string
}

func NewMilestone(id, routeID string, sequence int) *Milestone {
	return &Milestone{
		ID:          strings.TrimSpace(id),
		RouteID:     strings.TrimSpace(routeID),
		Sequence:    sequence,
		Status:      MilestonePlanned,
		PenaltyBps:  0,
		Metadata:    map[string]string{},
		Description: "",
	}
}

func (m *Milestone) IsTerminal() bool {
	return m.Status == MilestoneReleased || m.Status == MilestoneCancelled
}

func (m *Milestone) CanCertify() bool {
	return m.Status == MilestoneLocked || m.Status == MilestoneCertified
}

func (m *Milestone) CanRelease() bool {
	return m.Status == MilestoneCertified && m.DisputeID == ""
}

func (m *Milestone) RemainingAmount() money.Amount {
	used := m.ReleasedAmount.MustAdd(m.CancelledAmount).MustAdd(m.PenaltyAmount)
	if used >= m.Amount {
		return 0
	}
	return m.Amount - used
}

func (m *Milestone) DeliveredValue() money.Amount {
	return m.ReleasedAmount
}

func (m *Milestone) ProtectedValue() money.Amount {
	if m.Status == MilestoneDisputed {
		return m.FrozenAmount
	}
	return 0
}

func (m *Milestone) MarkLocked() {
	if m.Status == MilestonePlanned {
		m.Status = MilestoneLocked
	}
}

func (m *Milestone) MarkCertified(certificateID string) {
	m.CertificateID = strings.TrimSpace(certificateID)
	if m.Status == MilestoneLocked || m.Status == MilestonePlanned {
		m.Status = MilestoneCertified
	}
}

func (m *Milestone) MarkDisputed(disputeID string, amount money.Amount) {
	m.DisputeID = strings.TrimSpace(disputeID)
	m.FrozenAmount = amount
	m.Status = MilestoneDisputed
}

func (m *Milestone) ClearDispute() {
	m.DisputeID = ""
	m.FrozenAmount = 0
	if m.CertificateID != "" && m.RemainingAmount() > 0 {
		m.Status = MilestoneCertified
		return
	}
	if m.RemainingAmount() > 0 {
		m.Status = MilestoneLocked
		return
	}
	m.Status = MilestoneReleased
}

func (m *Milestone) AddReleased(amount money.Amount) error {
	next, err := m.ReleasedAmount.Add(amount)
	if err != nil {
		return err
	}
	if next > m.Amount {
		return fmt.Errorf("milestone %s release exceeds amount", m.ID)
	}
	m.ReleasedAmount = next
	if m.RemainingAmount() == 0 {
		m.Status = MilestoneReleased
	}
	return nil
}

func (m *Milestone) AddCancellation(refund, penalty money.Amount) error {
	cancelled, err := m.CancelledAmount.Add(refund)
	if err != nil {
		return err
	}
	penalized, err := m.PenaltyAmount.Add(penalty)
	if err != nil {
		return err
	}
	if cancelled.MustAdd(penalized).MustAdd(m.ReleasedAmount) > m.Amount {
		return fmt.Errorf("milestone %s cancellation exceeds amount", m.ID)
	}
	m.CancelledAmount = cancelled
	m.PenaltyAmount = penalized
	m.FrozenAmount = 0
	m.DisputeID = ""
	m.Status = MilestoneCancelled
	return nil
}

func (m *Milestone) Clone() *Milestone {
	copy := *m
	copy.Metadata = map[string]string{}
	for key, value := range m.Metadata {
		copy.Metadata[key] = value
	}
	return &copy
}

type Certificate struct {
	ID            string
	RouteID       string
	MilestoneID   string
	CustodianID   string
	DocumentHash  string
	IssuedEpoch   int
	VerifiedEpoch int
	Status        CertificateStatus
	Metadata      map[string]string
}

func NewCertificate(id, routeID, milestoneID, custodian, documentHash string, epoch int) *Certificate {
	return &Certificate{
		ID:           strings.TrimSpace(id),
		RouteID:      strings.TrimSpace(routeID),
		MilestoneID:  strings.TrimSpace(milestoneID),
		CustodianID:  strings.TrimSpace(custodian),
		DocumentHash: strings.TrimSpace(documentHash),
		IssuedEpoch:  epoch,
		Status:       CertificateIssued,
		Metadata:     map[string]string{},
	}
}

func (c *Certificate) Verify(epoch int) {
	c.VerifiedEpoch = epoch
	c.Status = CertificateVerified
}

func (c *Certificate) Clone() *Certificate {
	copy := *c
	copy.Metadata = map[string]string{}
	for key, value := range c.Metadata {
		copy.Metadata[key] = value
	}
	return &copy
}

type Dispute struct {
	ID             string
	RouteID        string
	MilestoneID    string
	ClaimantID     string
	Reason         string
	OpenedEpoch    int
	ResolvedEpoch  int
	Status         DisputeStatus
	AmountFrozen   money.Amount
	PenaltyBps     int64
	PenaltyCharged money.Amount
	Resolution     string
	Metadata       map[string]string
}

func NewDispute(id, routeID, milestoneID, claimant string, epoch int) *Dispute {
	return &Dispute{
		ID:          strings.TrimSpace(id),
		RouteID:     strings.TrimSpace(routeID),
		MilestoneID: strings.TrimSpace(milestoneID),
		ClaimantID:  strings.TrimSpace(claimant),
		OpenedEpoch: epoch,
		Status:      DisputeOpen,
		Metadata:    map[string]string{},
	}
}

func (d *Dispute) IsOpen() bool {
	return d.Status == DisputeOpen
}

func (d *Dispute) Resolve(epoch int, status DisputeStatus, resolution string) {
	d.ResolvedEpoch = epoch
	d.Status = status
	d.Resolution = strings.TrimSpace(resolution)
}

func (d *Dispute) Clone() *Dispute {
	copy := *d
	copy.Metadata = map[string]string{}
	for key, value := range d.Metadata {
		copy.Metadata[key] = value
	}
	return &copy
}

type Event struct {
	ID          string            `json:"id"`
	Type        EventType         `json:"type"`
	Epoch       int               `json:"epoch"`
	RouteID     string            `json:"routeId,omitempty"`
	MilestoneID string            `json:"milestoneId,omitempty"`
	AccountID   string            `json:"accountId,omitempty"`
	Asset       string            `json:"asset,omitempty"`
	Amount      money.Amount      `json:"amount,omitempty"`
	Message     string            `json:"message"`
	Fields      map[string]string `json:"fields,omitempty"`
}

func NewEvent(id string, eventType EventType, epoch int) Event {
	return Event{
		ID:     id,
		Type:   eventType,
		Epoch:  epoch,
		Fields: map[string]string{},
	}
}
