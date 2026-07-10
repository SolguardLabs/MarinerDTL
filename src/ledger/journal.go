package ledger

import (
	"fmt"
	"sort"

	"marinerdtl/src/domain"
	"marinerdtl/src/money"
)

type EntryType string

const (
	EntryDeposit           EntryType = "deposit"
	EntryEscrowReserve     EntryType = "escrow_reserve"
	EntryEscrowRelease     EntryType = "escrow_release"
	EntryEscrowRefund      EntryType = "escrow_refund"
	EntryPenalty           EntryType = "penalty"
	EntryRebate            EntryType = "rebate"
	EntryCustodyFee        EntryType = "custody_fee"
	EntryMilestoneLock     EntryType = "milestone_lock"
	EntryMilestoneDispute  EntryType = "milestone_dispute"
	EntryMilestoneResolve  EntryType = "milestone_resolve"
	EntryRouteCancellation EntryType = "route_cancellation"
)

type Entry struct {
	ID           string       `json:"id"`
	Type         EntryType    `json:"type"`
	Epoch        int          `json:"epoch"`
	RouteID      string       `json:"routeId,omitempty"`
	MilestoneID  string       `json:"milestoneId,omitempty"`
	Asset        string       `json:"asset"`
	AccountID    string       `json:"accountId,omitempty"`
	Counterparty string       `json:"counterparty,omitempty"`
	Debit        money.Amount `json:"debit"`
	Credit       money.Amount `json:"credit"`
	Memo         string       `json:"memo"`
}

type Journal struct {
	entries []Entry
	next    int
}

func NewJournal() *Journal {
	return &Journal{entries: []Entry{}, next: 1}
}

func (j *Journal) Append(entry Entry) Entry {
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("jnl-%06d", j.next)
		j.next++
	}
	j.entries = append(j.entries, entry)
	return entry
}

func (j *Journal) AppendAll(entries []Entry) {
	for _, entry := range entries {
		j.Append(entry)
	}
}

func (j *Journal) Entries() []Entry {
	return append([]Entry(nil), j.entries...)
}

func (j *Journal) EntriesByRoute(routeID string) []Entry {
	out := []Entry{}
	for _, entry := range j.entries {
		if entry.RouteID == routeID {
			out = append(out, entry)
		}
	}
	return out
}

func (j *Journal) EntriesByAccount(accountID string) []Entry {
	out := []Entry{}
	for _, entry := range j.entries {
		if entry.AccountID == accountID || entry.Counterparty == accountID {
			out = append(out, entry)
		}
	}
	return out
}

func (j *Journal) NetByAccount(accountID, asset string) money.Amount {
	var total money.Amount
	for _, entry := range j.EntriesByAccount(accountID) {
		if entry.Asset != asset {
			continue
		}
		if entry.AccountID == accountID {
			if entry.Credit > entry.Debit {
				total = total.MustAdd(entry.Credit.MustSub(entry.Debit))
			}
		}
		if entry.Counterparty == accountID {
			if entry.Debit > entry.Credit {
				total = total.MustAdd(entry.Debit.MustSub(entry.Credit))
			}
		}
	}
	return total
}

func Deposit(epoch int, account *domain.Account, asset string, amount money.Amount, memo string) ([]Entry, error) {
	if err := account.CreditAvailable(asset, amount); err != nil {
		return nil, err
	}
	return []Entry{{
		Type:      EntryDeposit,
		Epoch:     epoch,
		Asset:     asset,
		AccountID: account.ID,
		Credit:    amount,
		Memo:      memo,
	}}, nil
}

func FundEscrow(epoch int, shipper *domain.Account, route *domain.Route, amount money.Amount) ([]Entry, error) {
	if err := shipper.Reserve(route.Asset, amount); err != nil {
		return nil, err
	}
	if err := route.Fund(amount); err != nil {
		return nil, err
	}
	return []Entry{{
		Type:      EntryEscrowReserve,
		Epoch:     epoch,
		RouteID:   route.ID,
		Asset:     route.Asset,
		AccountID: shipper.ID,
		Debit:     amount,
		Memo:      "route escrow funded",
	}}, nil
}

func LockMilestone(epoch int, route *domain.Route, milestone *domain.Milestone) ([]Entry, error) {
	if route.EscrowRemaining < milestone.Amount {
		return nil, fmt.Errorf("route %s escrow cannot cover milestone %s", route.ID, milestone.ID)
	}
	milestone.MarkLocked()
	if route.Status == domain.RouteFunded {
		route.Status = domain.RouteInTransit
	}
	return []Entry{{
		Type:        EntryMilestoneLock,
		Epoch:       epoch,
		RouteID:     route.ID,
		MilestoneID: milestone.ID,
		Asset:       route.Asset,
		AccountID:   route.ShipperID,
		Debit:       milestone.Amount,
		Memo:        "milestone locked against route escrow",
	}}, nil
}

func ReleaseMilestone(epoch int, shipper, carrier *domain.Account, route *domain.Route, milestone *domain.Milestone, amount money.Amount) ([]Entry, error) {
	if amount == 0 {
		amount = milestone.RemainingAmount()
	}
	if amount <= 0 {
		return nil, fmt.Errorf("release amount is required")
	}
	if amount > milestone.RemainingAmount() {
		return nil, fmt.Errorf("release exceeds milestone remaining amount")
	}
	if err := shipper.DebitReserved(route.Asset, amount); err != nil {
		return nil, err
	}
	if err := carrier.CreditAvailable(route.Asset, amount); err != nil {
		return nil, err
	}
	if err := carrier.CreditSettled(route.Asset, amount); err != nil {
		return nil, err
	}
	if err := route.DebitEscrow(amount); err != nil {
		return nil, err
	}
	if err := milestone.AddReleased(amount); err != nil {
		return nil, err
	}
	return []Entry{{
		Type:         EntryEscrowRelease,
		Epoch:        epoch,
		RouteID:      route.ID,
		MilestoneID:  milestone.ID,
		Asset:        route.Asset,
		AccountID:    shipper.ID,
		Counterparty: carrier.ID,
		Debit:        amount,
		Credit:       amount,
		Memo:         "certified milestone released",
	}}, nil
}

func RefundMilestone(epoch int, shipper *domain.Account, route *domain.Route, milestone *domain.Milestone, amount money.Amount) ([]Entry, error) {
	if amount == 0 {
		return nil, nil
	}
	if err := shipper.Unreserve(route.Asset, amount); err != nil {
		return nil, err
	}
	if err := route.DebitEscrow(amount); err != nil {
		return nil, err
	}
	return []Entry{{
		Type:        EntryEscrowRefund,
		Epoch:       epoch,
		RouteID:     route.ID,
		MilestoneID: milestone.ID,
		Asset:       route.Asset,
		AccountID:   shipper.ID,
		Credit:      amount,
		Memo:        "milestone escrow refunded",
	}}, nil
}

func ChargePenalty(epoch int, shipper, beneficiary *domain.Account, route *domain.Route, milestone *domain.Milestone, amount money.Amount) ([]Entry, error) {
	if amount == 0 {
		return nil, nil
	}
	if err := shipper.DebitReserved(route.Asset, amount); err != nil {
		return nil, err
	}
	if err := beneficiary.CreditPenalty(route.Asset, amount); err != nil {
		return nil, err
	}
	if err := route.DebitEscrow(amount); err != nil {
		return nil, err
	}
	return []Entry{{
		Type:         EntryPenalty,
		Epoch:        epoch,
		RouteID:      route.ID,
		MilestoneID:  milestone.ID,
		Asset:        route.Asset,
		AccountID:    shipper.ID,
		Counterparty: beneficiary.ID,
		Debit:        amount,
		Credit:       amount,
		Memo:         "milestone penalty charged",
	}}, nil
}

func PayRebate(epoch int, treasury, recipient *domain.Account, route *domain.Route, amount money.Amount) ([]Entry, error) {
	if amount == 0 {
		return nil, fmt.Errorf("rebate amount is required")
	}
	if err := treasury.DebitAvailable(route.Asset, amount); err != nil {
		return nil, err
	}
	if err := recipient.CreditRebate(route.Asset, amount); err != nil {
		return nil, err
	}
	if err := route.CreditRebateClaim(amount); err != nil {
		return nil, err
	}
	return []Entry{{
		Type:         EntryRebate,
		Epoch:        epoch,
		RouteID:      route.ID,
		Asset:        route.Asset,
		AccountID:    treasury.ID,
		Counterparty: recipient.ID,
		Debit:        amount,
		Credit:       amount,
		Memo:         "route rebate paid",
	}}, nil
}

func PayCustodyFee(epoch int, treasury, custodian *domain.Account, route *domain.Route, milestone *domain.Milestone, amount money.Amount) ([]Entry, error) {
	if amount == 0 {
		return nil, nil
	}
	if err := treasury.DebitAvailable(route.Asset, amount); err != nil {
		return nil, err
	}
	if err := custodian.CreditAvailable(route.Asset, amount); err != nil {
		return nil, err
	}
	return []Entry{{
		Type:         EntryCustodyFee,
		Epoch:        epoch,
		RouteID:      route.ID,
		MilestoneID:  milestone.ID,
		Asset:        route.Asset,
		AccountID:    treasury.ID,
		Counterparty: custodian.ID,
		Debit:        amount,
		Credit:       amount,
		Memo:         "custody fee paid",
	}}, nil
}

func MarkDispute(epoch int, route *domain.Route, milestone *domain.Milestone, dispute *domain.Dispute) ([]Entry, error) {
	if dispute.AmountFrozen > milestone.RemainingAmount() {
		return nil, fmt.Errorf("dispute freeze exceeds milestone remaining amount")
	}
	milestone.MarkDisputed(dispute.ID, dispute.AmountFrozen)
	return []Entry{{
		Type:        EntryMilestoneDispute,
		Epoch:       epoch,
		RouteID:     route.ID,
		MilestoneID: milestone.ID,
		Asset:       route.Asset,
		AccountID:   dispute.ClaimantID,
		Debit:       dispute.AmountFrozen,
		Memo:        "milestone moved to dispute queue",
	}}, nil
}

func ResolveDispute(epoch int, route *domain.Route, milestone *domain.Milestone, dispute *domain.Dispute, status domain.DisputeStatus, resolution string) ([]Entry, error) {
	dispute.Resolve(epoch, status, resolution)
	milestone.ClearDispute()
	return []Entry{{
		Type:        EntryMilestoneResolve,
		Epoch:       epoch,
		RouteID:     route.ID,
		MilestoneID: milestone.ID,
		Asset:       route.Asset,
		AccountID:   dispute.ClaimantID,
		Credit:      dispute.AmountFrozen,
		Memo:        "milestone dispute resolved",
	}}, nil
}

func CancelRemainder(epoch int, shipper, beneficiary *domain.Account, route *domain.Route, milestone *domain.Milestone, refund, penalty money.Amount) ([]Entry, error) {
	entries := []Entry{}
	if penalty > 0 {
		penaltyEntries, err := ChargePenalty(epoch, shipper, beneficiary, route, milestone, penalty)
		if err != nil {
			return nil, err
		}
		entries = append(entries, penaltyEntries...)
	}
	if refund > 0 {
		refundEntries, err := RefundMilestone(epoch, shipper, route, milestone, refund)
		if err != nil {
			return nil, err
		}
		entries = append(entries, refundEntries...)
	}
	if err := milestone.AddCancellation(refund, penalty); err != nil {
		return nil, err
	}
	entries = append(entries, Entry{
		Type:        EntryRouteCancellation,
		Epoch:       epoch,
		RouteID:     route.ID,
		MilestoneID: milestone.ID,
		Asset:       route.Asset,
		AccountID:   shipper.ID,
		Debit:       penalty,
		Credit:      refund,
		Memo:        "milestone remainder reconciled",
	})
	return entries, nil
}

type Report struct {
	Entries []Entry `json:"entries"`
}

func (j *Journal) Report() Report {
	entries := j.Entries()
	sort.Slice(entries, func(i, k int) bool {
		return entries[i].ID < entries[k].ID
	})
	return Report{Entries: entries}
}
