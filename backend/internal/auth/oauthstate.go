package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// PendingOAuth holds short-lived OAuth state between auth redirect and callback.
type PendingOAuth struct {
	Provider    string
	Next        string // optional post-login redirect (validated on callback)
	AccountType string // affiliate | merchant
	CompanyName string // required when AccountType is merchant
	Expires     time.Time
}

var oauthStates sync.Map // state string -> PendingOAuth

const oauthStateTTL = 15 * time.Minute

// NewOAuthState registers a pending OAuth flow and returns the opaque state token.
func NewOAuthState(provider, next, accountType, companyName string) (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := hex.EncodeToString(b)
	if accountType == "" {
		accountType = "affiliate"
	}
	oauthStates.Store(state, PendingOAuth{
		Provider:    provider,
		Next:        next,
		AccountType: accountType,
		CompanyName: companyName,
		Expires:     time.Now().Add(oauthStateTTL),
	})
	return state, nil
}

// ConsumeOAuthState validates and removes state.
func ConsumeOAuthState(state string) (provider, next, accountType, companyName string, ok bool) {
	if state == "" {
		return "", "", "", "", false
	}
	v, loaded := oauthStates.LoadAndDelete(state)
	if !loaded {
		return "", "", "", "", false
	}
	p, ok := v.(PendingOAuth)
	if !ok {
		return "", "", "", "", false
	}
	if time.Now().After(p.Expires) {
		return "", "", "", "", false
	}
	at := p.AccountType
	if at == "" {
		at = "affiliate"
	}
	return p.Provider, p.Next, at, p.CompanyName, true
}
