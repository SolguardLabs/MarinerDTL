package risk

import (
	"fmt"
	"sort"
	"strings"

	"marinerdtl/src/domain"
	"marinerdtl/src/money"
)

// Severity expresses an operational risk tier. Values are ordered from the
// least restrictive tier to the most restrictive one.
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityModerate Severity = "moderate"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Scenario defines the deterministic shocks used to calculate required
// capital. All percentages are expressed in basis points.
type Scenario struct {
	Name                       string `json:"name"`
	EscrowHaircutBps           int64  `json:"escrowHaircutBps"`
	FrozenValueLossBps         int64  `json:"frozenValueLossBps"`
	UnsettledValueLossBps      int64  `json:"unsettledValueLossBps"`
	RebateReserveBps           int64  `json:"rebateReserveBps"`
	PenaltyRecoveryBps         int64  `json:"penaltyRecoveryBps"`
	CustodianConcentrationBps  int64  `json:"custodianConcentrationBps"`
	OperatorConcentrationBps   int64  `json:"operatorConcentrationBps"`
	CorridorConcentrationBps   int64  `json:"corridorConcentrationBps"`
	MinimumCoverageBps         int64  `json:"minimumCoverageBps"`
	MaximumSingleRouteShareBps int64  `json:"maximumSingleRouteShareBps"`
}

func OperatingScenario() Scenario {
	return Scenario{
		Name:                       "operating",
		EscrowHaircutBps:           250,
		FrozenValueLossBps:         2000,
		UnsettledValueLossBps:      350,
		RebateReserveBps:           10000,
		PenaltyRecoveryBps:         6000,
		CustodianConcentrationBps:  1200,
		OperatorConcentrationBps:   900,
		CorridorConcentrationBps:   650,
		MinimumCoverageBps:         12000,
		MaximumSingleRouteShareBps: 3500,
	}
}

func SevereScenario() Scenario {
	return Scenario{
		Name:                       "severe",
		EscrowHaircutBps:           900,
		FrozenValueLossBps:         6500,
		UnsettledValueLossBps:      1250,
		RebateReserveBps:           10000,
		PenaltyRecoveryBps:         2500,
		CustodianConcentrationBps:  2600,
		OperatorConcentrationBps:   2200,
		CorridorConcentrationBps:   1800,
		MinimumCoverageBps:         14500,
		MaximumSingleRouteShareBps: 2500,
	}
}

func (s Scenario) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("scenario name is required")
	}
	fields := map[string]int64{
		"escrow haircut":             s.EscrowHaircutBps,
		"frozen value loss":          s.FrozenValueLossBps,
		"unsettled value loss":       s.UnsettledValueLossBps,
		"rebate reserve":             s.RebateReserveBps,
		"penalty recovery":           s.PenaltyRecoveryBps,
		"custodian concentration":    s.CustodianConcentrationBps,
		"operator concentration":     s.OperatorConcentrationBps,
		"corridor concentration":     s.CorridorConcentrationBps,
		"single route concentration": s.MaximumSingleRouteShareBps,
	}
	for name, value := range fields {
		if value < 0 || value > money.BasisMax {
			return fmt.Errorf("%s bps out of range", name)
		}
	}
	if s.MinimumCoverageBps < money.BasisMax || s.MinimumCoverageBps > 30000 {
		return fmt.Errorf("minimum coverage bps must be between 10000 and 30000")
	}
	return nil
}

// RoutePosition is a normalized, read-only projection of a route and its
// milestones. It intentionally contains no command methods.
type RoutePosition struct {
	RouteID           string                  `json:"routeId"`
	Asset             string                  `json:"asset"`
	Corridor          string                  `json:"corridor"`
	OperatorID        string                  `json:"operatorId"`
	GrossValue        money.Amount            `json:"grossValue"`
	EscrowRemaining   money.Amount            `json:"escrowRemaining"`
	ReleasedValue     money.Amount            `json:"releasedValue"`
	CancelledValue    money.Amount            `json:"cancelledValue"`
	FrozenValue       money.Amount            `json:"frozenValue"`
	PenaltyValue      money.Amount            `json:"penaltyValue"`
	RebateOutstanding money.Amount            `json:"rebateOutstanding"`
	UnsettledValue    money.Amount            `json:"unsettledValue"`
	CustodianValues   map[string]money.Amount `json:"custodianValues"`
}

func ProjectState(state *domain.State) ([]RoutePosition, error) {
	if state == nil {
		return nil, fmt.Errorf("state is required")
	}
	positions := make([]RoutePosition, 0, len(state.Routes))
	for _, routeID := range state.SortedRouteIDs() {
		route := state.Routes[routeID]
		position := RoutePosition{
			RouteID:           route.ID,
			Asset:             route.Asset,
			Corridor:          corridor(route.OriginPort, route.DestinationPort),
			OperatorID:        route.OperatorID,
			GrossValue:        state.RouteAmount(route.ID),
			EscrowRemaining:   route.EscrowRemaining,
			ReleasedValue:     state.ReleasedAmount(route.ID),
			CancelledValue:    state.CancelledAmount(route.ID),
			FrozenValue:       state.FrozenAmount(route.ID),
			PenaltyValue:      state.PenaltyAmount(route.ID),
			RebateOutstanding: route.OutstandingRebate(),
			CustodianValues:   map[string]money.Amount{},
		}
		used := position.ReleasedValue.MustAdd(position.CancelledValue).MustAdd(position.PenaltyValue)
		if used < position.GrossValue {
			position.UnsettledValue = position.GrossValue - used
		}
		for _, milestone := range state.RouteMilestones(route.ID) {
			custodian := strings.TrimSpace(milestone.CustodianID)
			if custodian == "" {
				custodian = "unassigned"
			}
			current := position.CustodianValues[custodian]
			next, err := current.Add(milestone.RemainingAmount())
			if err != nil {
				return nil, fmt.Errorf("route %s custodian exposure: %w", route.ID, err)
			}
			position.CustodianValues[custodian] = next
		}
		positions = append(positions, position)
	}
	return positions, nil
}

func corridor(origin, destination string) string {
	origin = strings.ToUpper(strings.TrimSpace(origin))
	destination = strings.ToUpper(strings.TrimSpace(destination))
	if origin == "" {
		origin = "UNSPECIFIED"
	}
	if destination == "" {
		destination = "UNSPECIFIED"
	}
	return origin + ">" + destination
}

func sortedKeys(values map[string]money.Amount) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
