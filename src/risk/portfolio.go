package risk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"marinerdtl/src/domain"
	"marinerdtl/src/money"
)

type CapitalBreakdown struct {
	EscrowHaircut    money.Amount `json:"escrowHaircut"`
	FrozenValue      money.Amount `json:"frozenValue"`
	UnsettledValue   money.Amount `json:"unsettledValue"`
	RebateReserve    money.Amount `json:"rebateReserve"`
	PenaltyRecovery  money.Amount `json:"penaltyRecovery"`
	CustodianAddOn   money.Amount `json:"custodianAddOn"`
	OperatorAddOn    money.Amount `json:"operatorAddOn"`
	CorridorAddOn    money.Amount `json:"corridorAddOn"`
	RequiredCapital  money.Amount `json:"requiredCapital"`
	TargetCapital    money.Amount `json:"targetCapital"`
	AvailableCapital money.Amount `json:"availableCapital"`
	CapitalSurplus   money.Amount `json:"capitalSurplus"`
	CapitalShortfall money.Amount `json:"capitalShortfall"`
	CoverageBps      int64        `json:"coverageBps"`
}

type Concentration struct {
	ID       string       `json:"id"`
	Exposure money.Amount `json:"exposure"`
	ShareBps int64        `json:"shareBps"`
}

type RouteRisk struct {
	RouteID           string       `json:"routeId"`
	Corridor          string       `json:"corridor"`
	OperatorID        string       `json:"operatorId"`
	GrossValue        money.Amount `json:"grossValue"`
	AtRiskValue       money.Amount `json:"atRiskValue"`
	CapitalCharge     money.Amount `json:"capitalCharge"`
	PortfolioShareBps int64        `json:"portfolioShareBps"`
	Severity          Severity     `json:"severity"`
}

type Finding struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Subject  string   `json:"subject"`
	Message  string   `json:"message"`
}

type Report struct {
	Scenario    string           `json:"scenario"`
	GrossValue  money.Amount     `json:"grossValue"`
	AtRiskValue money.Amount     `json:"atRiskValue"`
	Capital     CapitalBreakdown `json:"capital"`
	Routes      []RouteRisk      `json:"routes"`
	Custodians  []Concentration  `json:"custodians"`
	Operators   []Concentration  `json:"operators"`
	Corridors   []Concentration  `json:"corridors"`
	Findings    []Finding        `json:"findings"`
	Digest      string           `json:"digest"`
}

type Analyzer struct {
	scenario Scenario
}

func NewAnalyzer(scenario Scenario) (*Analyzer, error) {
	if err := scenario.Validate(); err != nil {
		return nil, err
	}
	return &Analyzer{scenario: scenario}, nil
}

func MustAnalyzer(scenario Scenario) *Analyzer {
	analyzer, err := NewAnalyzer(scenario)
	if err != nil {
		panic(err)
	}
	return analyzer
}

func (a *Analyzer) AnalyzeState(state *domain.State, availableCapital money.Amount) (Report, error) {
	positions, err := ProjectState(state)
	if err != nil {
		return Report{}, err
	}
	return a.Analyze(positions, availableCapital)
}

func (a *Analyzer) Analyze(positions []RoutePosition, availableCapital money.Amount) (Report, error) {
	if a == nil {
		return Report{}, fmt.Errorf("analyzer is required")
	}
	if err := availableCapital.Validate(); err != nil {
		return Report{}, fmt.Errorf("available capital: %w", err)
	}
	report := Report{Scenario: a.scenario.Name}
	custodians := map[string]money.Amount{}
	operators := map[string]money.Amount{}
	corridors := map[string]money.Amount{}

	for _, position := range positions {
		if err := validatePosition(position); err != nil {
			return Report{}, err
		}
		report.GrossValue = report.GrossValue.MustAdd(position.GrossValue)
		atRisk := routeAtRisk(position)
		report.AtRiskValue = report.AtRiskValue.MustAdd(atRisk)
		operators[position.OperatorID] = operators[position.OperatorID].MustAdd(atRisk)
		corridors[position.Corridor] = corridors[position.Corridor].MustAdd(atRisk)
		for custodian, value := range position.CustodianValues {
			custodians[custodian] = custodians[custodian].MustAdd(value)
		}
	}

	report.Custodians = concentrations(custodians, report.AtRiskValue)
	report.Operators = concentrations(operators, report.AtRiskValue)
	report.Corridors = concentrations(corridors, report.AtRiskValue)
	report.Capital = a.capital(positions, report.Custodians, report.Operators, report.Corridors, availableCapital)
	report.Routes = a.routes(positions, report.GrossValue)
	report.Findings = a.findings(report)
	digest, err := reportDigest(report)
	if err != nil {
		return Report{}, err
	}
	report.Digest = digest
	return report, nil
}

func validatePosition(position RoutePosition) error {
	if position.RouteID == "" {
		return fmt.Errorf("route id is required")
	}
	if position.OperatorID == "" {
		return fmt.Errorf("route %s operator is required", position.RouteID)
	}
	if position.Corridor == "" {
		return fmt.Errorf("route %s corridor is required", position.RouteID)
	}
	amounts := map[string]money.Amount{
		"gross": position.GrossValue, "escrow": position.EscrowRemaining,
		"released": position.ReleasedValue, "cancelled": position.CancelledValue,
		"frozen": position.FrozenValue, "penalty": position.PenaltyValue,
		"rebate": position.RebateOutstanding, "unsettled": position.UnsettledValue,
	}
	for name, amount := range amounts {
		if err := amount.Validate(); err != nil {
			return fmt.Errorf("route %s %s value: %w", position.RouteID, name, err)
		}
	}
	for custodian, value := range position.CustodianValues {
		if custodian == "" {
			return fmt.Errorf("route %s has empty custodian id", position.RouteID)
		}
		if err := value.Validate(); err != nil {
			return fmt.Errorf("route %s custodian %s: %w", position.RouteID, custodian, err)
		}
	}
	return nil
}

func routeAtRisk(position RoutePosition) money.Amount {
	return position.UnsettledValue.MustAdd(position.FrozenValue).MustAdd(position.RebateOutstanding)
}

func concentrations(values map[string]money.Amount, total money.Amount) []Concentration {
	out := make([]Concentration, 0, len(values))
	for _, id := range sortedKeys(values) {
		out = append(out, Concentration{ID: id, Exposure: values[id], ShareBps: money.RatioBps(values[id], total)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Exposure == out[j].Exposure {
			return out[i].ID < out[j].ID
		}
		return out[i].Exposure > out[j].Exposure
	})
	return out
}

func (a *Analyzer) capital(positions []RoutePosition, custodians, operators, corridors []Concentration, available money.Amount) CapitalBreakdown {
	var out CapitalBreakdown
	out.AvailableCapital = available
	for _, position := range positions {
		out.EscrowHaircut = out.EscrowHaircut.MustAdd(position.EscrowRemaining.MustMulBps(a.scenario.EscrowHaircutBps))
		out.FrozenValue = out.FrozenValue.MustAdd(position.FrozenValue.MustMulBps(a.scenario.FrozenValueLossBps))
		out.UnsettledValue = out.UnsettledValue.MustAdd(position.UnsettledValue.MustMulBps(a.scenario.UnsettledValueLossBps))
		out.RebateReserve = out.RebateReserve.MustAdd(position.RebateOutstanding.MustMulBps(a.scenario.RebateReserveBps))
		recoverable := position.PenaltyValue.MustMulBps(a.scenario.PenaltyRecoveryBps)
		if position.PenaltyValue > recoverable {
			out.PenaltyRecovery = out.PenaltyRecovery.MustAdd(position.PenaltyValue - recoverable)
		}
	}
	out.CustodianAddOn = concentrationAddOn(custodians, a.scenario.CustodianConcentrationBps)
	out.OperatorAddOn = concentrationAddOn(operators, a.scenario.OperatorConcentrationBps)
	out.CorridorAddOn = concentrationAddOn(corridors, a.scenario.CorridorConcentrationBps)
	out.RequiredCapital = out.EscrowHaircut.
		MustAdd(out.FrozenValue).
		MustAdd(out.UnsettledValue).
		MustAdd(out.RebateReserve).
		MustAdd(out.PenaltyRecovery).
		MustAdd(out.CustodianAddOn).
		MustAdd(out.OperatorAddOn).
		MustAdd(out.CorridorAddOn)
	out.TargetCapital = out.RequiredCapital.MustMulBps(a.scenario.MinimumCoverageBps)
	out.CoverageBps = coverageBps(available, out.RequiredCapital)
	if available >= out.TargetCapital {
		out.CapitalSurplus = available - out.TargetCapital
	} else {
		out.CapitalShortfall = out.TargetCapital - available
	}
	return out
}

func coverageBps(available, required money.Amount) int64 {
	if required <= 0 {
		if available > 0 {
			return 30000
		}
		return 0
	}
	quotient := int64(available / required)
	if quotient >= 3 {
		return 30000
	}
	remainder := available % required
	return quotient*money.BasisMax + int64(remainder)*money.BasisMax/int64(required)
}

func concentrationAddOn(values []Concentration, factorBps int64) money.Amount {
	if len(values) == 0 {
		return 0
	}
	threshold := int64(2500)
	var charge money.Amount
	for _, item := range values {
		if item.ShareBps <= threshold {
			continue
		}
		excessBps := item.ShareBps - threshold
		excess := item.Exposure.MustMulBps(excessBps)
		charge = charge.MustAdd(excess.MustMulBps(factorBps))
	}
	return charge
}

func (a *Analyzer) routes(positions []RoutePosition, gross money.Amount) []RouteRisk {
	out := make([]RouteRisk, 0, len(positions))
	for _, position := range positions {
		atRisk := routeAtRisk(position)
		charge := position.EscrowRemaining.MustMulBps(a.scenario.EscrowHaircutBps).
			MustAdd(position.FrozenValue.MustMulBps(a.scenario.FrozenValueLossBps)).
			MustAdd(position.UnsettledValue.MustMulBps(a.scenario.UnsettledValueLossBps)).
			MustAdd(position.RebateOutstanding.MustMulBps(a.scenario.RebateReserveBps))
		share := money.RatioBps(position.GrossValue, gross)
		severity := SeverityLow
		switch {
		case share > a.scenario.MaximumSingleRouteShareBps || position.FrozenValue > 0:
			severity = SeverityHigh
		case share > a.scenario.MaximumSingleRouteShareBps*3/4:
			severity = SeverityModerate
		}
		out = append(out, RouteRisk{
			RouteID: position.RouteID, Corridor: position.Corridor, OperatorID: position.OperatorID,
			GrossValue: position.GrossValue, AtRiskValue: atRisk, CapitalCharge: charge,
			PortfolioShareBps: share, Severity: severity,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CapitalCharge == out[j].CapitalCharge {
			return out[i].RouteID < out[j].RouteID
		}
		return out[i].CapitalCharge > out[j].CapitalCharge
	})
	return out
}

func reportDigest(report Report) (string, error) {
	copy := report
	copy.Digest = ""
	encoded, err := json.Marshal(copy)
	if err != nil {
		return "", fmt.Errorf("encode risk report: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
