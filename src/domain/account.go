package domain

import (
	"fmt"
	"sort"
	"strings"

	"marinerdtl/src/money"
)

type Account struct {
	ID          string
	Role        AccountRole
	DisplayName string
	Free        map[string]money.Amount
	Reserved    map[string]money.Amount
	Settled     map[string]money.Amount
	Rebates     map[string]money.Amount
	Penalties   map[string]money.Amount
	Metadata    map[string]string
}

func NewAccount(id string, role AccountRole) *Account {
	return &Account{
		ID:        strings.TrimSpace(id),
		Role:      role,
		Free:      map[string]money.Amount{},
		Reserved:  map[string]money.Amount{},
		Settled:   map[string]money.Amount{},
		Rebates:   map[string]money.Amount{},
		Penalties: map[string]money.Amount{},
		Metadata:  map[string]string{},
	}
}

func (a *Account) EnsureAsset(asset string) {
	if _, ok := a.Free[asset]; !ok {
		a.Free[asset] = 0
	}
	if _, ok := a.Reserved[asset]; !ok {
		a.Reserved[asset] = 0
	}
	if _, ok := a.Settled[asset]; !ok {
		a.Settled[asset] = 0
	}
	if _, ok := a.Rebates[asset]; !ok {
		a.Rebates[asset] = 0
	}
	if _, ok := a.Penalties[asset]; !ok {
		a.Penalties[asset] = 0
	}
}

func (a *Account) Available(asset string) money.Amount {
	a.EnsureAsset(asset)
	return a.Free[asset]
}

func (a *Account) ReservedBalance(asset string) money.Amount {
	a.EnsureAsset(asset)
	return a.Reserved[asset]
}

func (a *Account) SettledBalance(asset string) money.Amount {
	a.EnsureAsset(asset)
	return a.Settled[asset]
}

func (a *Account) RebateBalance(asset string) money.Amount {
	a.EnsureAsset(asset)
	return a.Rebates[asset]
}

func (a *Account) PenaltyBalance(asset string) money.Amount {
	a.EnsureAsset(asset)
	return a.Penalties[asset]
}

func (a *Account) CreditAvailable(asset string, amount money.Amount) error {
	if err := amount.Validate(); err != nil {
		return err
	}
	a.EnsureAsset(asset)
	next, err := a.Free[asset].Add(amount)
	if err != nil {
		return err
	}
	a.Free[asset] = next
	return nil
}

func (a *Account) DebitAvailable(asset string, amount money.Amount) error {
	if err := amount.Validate(); err != nil {
		return err
	}
	a.EnsureAsset(asset)
	next, err := a.Free[asset].Sub(amount)
	if err != nil {
		return fmt.Errorf("account %s free %s cannot cover %s", a.ID, asset, amount)
	}
	a.Free[asset] = next
	return nil
}

func (a *Account) CreditReserved(asset string, amount money.Amount) error {
	if err := amount.Validate(); err != nil {
		return err
	}
	a.EnsureAsset(asset)
	next, err := a.Reserved[asset].Add(amount)
	if err != nil {
		return err
	}
	a.Reserved[asset] = next
	return nil
}

func (a *Account) DebitReserved(asset string, amount money.Amount) error {
	if err := amount.Validate(); err != nil {
		return err
	}
	a.EnsureAsset(asset)
	next, err := a.Reserved[asset].Sub(amount)
	if err != nil {
		return fmt.Errorf("account %s reserved %s cannot cover %s", a.ID, asset, amount)
	}
	a.Reserved[asset] = next
	return nil
}

func (a *Account) Reserve(asset string, amount money.Amount) error {
	if err := a.DebitAvailable(asset, amount); err != nil {
		return err
	}
	return a.CreditReserved(asset, amount)
}

func (a *Account) Unreserve(asset string, amount money.Amount) error {
	if err := a.DebitReserved(asset, amount); err != nil {
		return err
	}
	return a.CreditAvailable(asset, amount)
}

func (a *Account) CreditSettled(asset string, amount money.Amount) error {
	if err := amount.Validate(); err != nil {
		return err
	}
	a.EnsureAsset(asset)
	next, err := a.Settled[asset].Add(amount)
	if err != nil {
		return err
	}
	a.Settled[asset] = next
	return nil
}

func (a *Account) CreditRebate(asset string, amount money.Amount) error {
	if err := amount.Validate(); err != nil {
		return err
	}
	a.EnsureAsset(asset)
	next, err := a.Rebates[asset].Add(amount)
	if err != nil {
		return err
	}
	a.Rebates[asset] = next
	return a.CreditAvailable(asset, amount)
}

func (a *Account) CreditPenalty(asset string, amount money.Amount) error {
	if err := amount.Validate(); err != nil {
		return err
	}
	a.EnsureAsset(asset)
	next, err := a.Penalties[asset].Add(amount)
	if err != nil {
		return err
	}
	a.Penalties[asset] = next
	return a.CreditAvailable(asset, amount)
}

func (a *Account) Assets() []string {
	seen := map[string]struct{}{}
	for key := range a.Free {
		seen[key] = struct{}{}
	}
	for key := range a.Reserved {
		seen[key] = struct{}{}
	}
	for key := range a.Settled {
		seen[key] = struct{}{}
	}
	for key := range a.Rebates {
		seen[key] = struct{}{}
	}
	for key := range a.Penalties {
		seen[key] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for key := range seen {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func (a *Account) Clone() *Account {
	copy := NewAccount(a.ID, a.Role)
	copy.DisplayName = a.DisplayName
	copy.Free = money.MapCopy(a.Free)
	copy.Reserved = money.MapCopy(a.Reserved)
	copy.Settled = money.MapCopy(a.Settled)
	copy.Rebates = money.MapCopy(a.Rebates)
	copy.Penalties = money.MapCopy(a.Penalties)
	for key, value := range a.Metadata {
		copy.Metadata[key] = value
	}
	return copy
}
