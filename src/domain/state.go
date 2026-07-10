package domain

import (
	"fmt"
	"sort"

	"marinerdtl/src/money"
)

type State struct {
	Name         string
	Epoch        int
	Assets       map[string]*Asset
	Accounts     map[string]*Account
	Routes       map[string]*Route
	Milestones   map[string]*Milestone
	Certificates map[string]*Certificate
	Disputes     map[string]*Dispute
	Events       []Event
	Metrics      map[string]money.Amount
	nextEvent    int
}

func NewState(name string) *State {
	if name == "" {
		name = "MarinerDTL"
	}
	return &State{
		Name:         name,
		Epoch:        1,
		Assets:       map[string]*Asset{},
		Accounts:     map[string]*Account{},
		Routes:       map[string]*Route{},
		Milestones:   map[string]*Milestone{},
		Certificates: map[string]*Certificate{},
		Disputes:     map[string]*Dispute{},
		Events:       []Event{},
		Metrics:      map[string]money.Amount{},
		nextEvent:    1,
	}
}

func (s *State) AddAsset(asset *Asset) error {
	if asset == nil || asset.ID == "" {
		return fmt.Errorf("asset is required")
	}
	if _, exists := s.Assets[asset.ID]; exists {
		return fmt.Errorf("asset %s already exists", asset.ID)
	}
	s.Assets[asset.ID] = asset
	return nil
}

func (s *State) RequireAsset(id string) (*Asset, error) {
	asset, ok := s.Assets[id]
	if !ok {
		return nil, fmt.Errorf("asset %s not found", id)
	}
	return asset, nil
}

func (s *State) AddAccount(account *Account) error {
	if account == nil || account.ID == "" {
		return fmt.Errorf("account is required")
	}
	if _, exists := s.Accounts[account.ID]; exists {
		return fmt.Errorf("account %s already exists", account.ID)
	}
	s.Accounts[account.ID] = account
	return nil
}

func (s *State) RequireAccount(id string) (*Account, error) {
	account, ok := s.Accounts[id]
	if !ok {
		return nil, fmt.Errorf("account %s not found", id)
	}
	return account, nil
}

func (s *State) RequireRole(id string, roles ...AccountRole) (*Account, error) {
	account, err := s.RequireAccount(id)
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if account.Role == role {
			return account, nil
		}
	}
	return nil, fmt.Errorf("account %s has role %s", id, account.Role)
}

func (s *State) AddRoute(route *Route) error {
	if route == nil || route.ID == "" {
		return fmt.Errorf("route is required")
	}
	if _, exists := s.Routes[route.ID]; exists {
		return fmt.Errorf("route %s already exists", route.ID)
	}
	s.Routes[route.ID] = route
	return nil
}

func (s *State) RequireRoute(id string) (*Route, error) {
	route, ok := s.Routes[id]
	if !ok {
		return nil, fmt.Errorf("route %s not found", id)
	}
	return route, nil
}

func (s *State) AddMilestone(milestone *Milestone) error {
	if milestone == nil || milestone.ID == "" {
		return fmt.Errorf("milestone is required")
	}
	if _, exists := s.Milestones[milestone.ID]; exists {
		return fmt.Errorf("milestone %s already exists", milestone.ID)
	}
	route, err := s.RequireRoute(milestone.RouteID)
	if err != nil {
		return err
	}
	s.Milestones[milestone.ID] = milestone
	route.AddMilestone(milestone.ID)
	return nil
}

func (s *State) RequireMilestone(id string) (*Milestone, error) {
	milestone, ok := s.Milestones[id]
	if !ok {
		return nil, fmt.Errorf("milestone %s not found", id)
	}
	return milestone, nil
}

func (s *State) AddCertificate(certificate *Certificate) error {
	if certificate == nil || certificate.ID == "" {
		return fmt.Errorf("certificate is required")
	}
	if _, exists := s.Certificates[certificate.ID]; exists {
		return fmt.Errorf("certificate %s already exists", certificate.ID)
	}
	s.Certificates[certificate.ID] = certificate
	return nil
}

func (s *State) RequireCertificate(id string) (*Certificate, error) {
	certificate, ok := s.Certificates[id]
	if !ok {
		return nil, fmt.Errorf("certificate %s not found", id)
	}
	return certificate, nil
}

func (s *State) AddDispute(dispute *Dispute) error {
	if dispute == nil || dispute.ID == "" {
		return fmt.Errorf("dispute is required")
	}
	if _, exists := s.Disputes[dispute.ID]; exists {
		return fmt.Errorf("dispute %s already exists", dispute.ID)
	}
	s.Disputes[dispute.ID] = dispute
	return nil
}

func (s *State) RequireDispute(id string) (*Dispute, error) {
	dispute, ok := s.Disputes[id]
	if !ok {
		return nil, fmt.Errorf("dispute %s not found", id)
	}
	return dispute, nil
}

func (s *State) RouteMilestones(routeID string) []*Milestone {
	route, ok := s.Routes[routeID]
	if !ok {
		return nil
	}
	out := make([]*Milestone, 0, len(route.Milestones))
	for _, id := range route.Milestones {
		if milestone, ok := s.Milestones[id]; ok {
			out = append(out, milestone)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Sequence == out[j].Sequence {
			return out[i].ID < out[j].ID
		}
		return out[i].Sequence < out[j].Sequence
	})
	return out
}

func (s *State) RouteDisputes(routeID string) []*Dispute {
	out := []*Dispute{}
	for _, dispute := range s.Disputes {
		if dispute.RouteID == routeID {
			out = append(out, dispute)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *State) OpenRouteDisputes(routeID string) []*Dispute {
	out := []*Dispute{}
	for _, dispute := range s.RouteDisputes(routeID) {
		if dispute.IsOpen() {
			out = append(out, dispute)
		}
	}
	return out
}

func (s *State) HasOpenRouteDispute(routeID string) bool {
	return len(s.OpenRouteDisputes(routeID)) > 0
}

func (s *State) MilestoneByCertificate(certificateID string) *Milestone {
	for _, milestone := range s.Milestones {
		if milestone.CertificateID == certificateID {
			return milestone
		}
	}
	return nil
}

func (s *State) AddMetric(name string, amount money.Amount) {
	current := s.Metrics[name]
	s.Metrics[name] = current.MustAdd(amount)
}

func (s *State) SetMetric(name string, amount money.Amount) {
	s.Metrics[name] = amount
}

func (s *State) NextEvent(eventType EventType) Event {
	id := fmt.Sprintf("evt-%06d", s.nextEvent)
	s.nextEvent++
	return NewEvent(id, eventType, s.Epoch)
}

func (s *State) AppendEvent(event Event) {
	s.Events = append(s.Events, event)
}

func (s *State) Advance(to int) error {
	if to <= 0 {
		s.Epoch++
		return nil
	}
	if to < s.Epoch {
		return fmt.Errorf("cannot move epoch backwards")
	}
	s.Epoch = to
	return nil
}

func (s *State) SortedAssetIDs() []string {
	out := make([]string, 0, len(s.Assets))
	for id := range s.Assets {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *State) SortedAccountIDs() []string {
	out := make([]string, 0, len(s.Accounts))
	for id := range s.Accounts {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *State) SortedRouteIDs() []string {
	out := make([]string, 0, len(s.Routes))
	for id := range s.Routes {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *State) SortedMilestoneIDs() []string {
	out := make([]string, 0, len(s.Milestones))
	for id := range s.Milestones {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool {
		left := s.Milestones[out[i]]
		right := s.Milestones[out[j]]
		if left.RouteID == right.RouteID {
			if left.Sequence == right.Sequence {
				return left.ID < right.ID
			}
			return left.Sequence < right.Sequence
		}
		return left.RouteID < right.RouteID
	})
	return out
}

func (s *State) SortedCertificateIDs() []string {
	out := make([]string, 0, len(s.Certificates))
	for id := range s.Certificates {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *State) SortedDisputeIDs() []string {
	out := make([]string, 0, len(s.Disputes))
	for id := range s.Disputes {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (s *State) RouteAmount(routeID string) money.Amount {
	var total money.Amount
	for _, milestone := range s.RouteMilestones(routeID) {
		total = total.MustAdd(milestone.Amount)
	}
	return total
}

func (s *State) ReleasedAmount(routeID string) money.Amount {
	var total money.Amount
	for _, milestone := range s.RouteMilestones(routeID) {
		total = total.MustAdd(milestone.ReleasedAmount)
	}
	return total
}

func (s *State) CancelledAmount(routeID string) money.Amount {
	var total money.Amount
	for _, milestone := range s.RouteMilestones(routeID) {
		total = total.MustAdd(milestone.CancelledAmount)
	}
	return total
}

func (s *State) PenaltyAmount(routeID string) money.Amount {
	var total money.Amount
	for _, milestone := range s.RouteMilestones(routeID) {
		total = total.MustAdd(milestone.PenaltyAmount)
	}
	return total
}

func (s *State) FrozenAmount(routeID string) money.Amount {
	var total money.Amount
	for _, milestone := range s.RouteMilestones(routeID) {
		total = total.MustAdd(milestone.FrozenAmount)
	}
	return total
}

func (s *State) DeliveredProgressBps(routeID string) int64 {
	total := s.RouteAmount(routeID)
	released := s.ReleasedAmount(routeID)
	return money.RatioBps(released, total)
}

func (s *State) IsRouteComplete(routeID string) bool {
	milestones := s.RouteMilestones(routeID)
	if len(milestones) == 0 {
		return false
	}
	for _, milestone := range milestones {
		if milestone.Status != MilestoneReleased {
			return false
		}
	}
	return true
}

func (s *State) Clone() *State {
	copy := NewState(s.Name)
	copy.Epoch = s.Epoch
	copy.nextEvent = s.nextEvent
	for id, asset := range s.Assets {
		assetCopy := *asset
		assetCopy.Metadata = map[string]string{}
		for key, value := range asset.Metadata {
			assetCopy.Metadata[key] = value
		}
		copy.Assets[id] = &assetCopy
	}
	for id, account := range s.Accounts {
		copy.Accounts[id] = account.Clone()
	}
	for id, route := range s.Routes {
		copy.Routes[id] = route.Clone()
	}
	for id, milestone := range s.Milestones {
		copy.Milestones[id] = milestone.Clone()
	}
	for id, certificate := range s.Certificates {
		copy.Certificates[id] = certificate.Clone()
	}
	for id, dispute := range s.Disputes {
		copy.Disputes[id] = dispute.Clone()
	}
	copy.Events = append([]Event(nil), s.Events...)
	copy.Metrics = money.MapCopy(s.Metrics)
	return copy
}
