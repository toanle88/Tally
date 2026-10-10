package main

import (
	"time"

	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/platform/money"
)

func newMemoryPostingService() (*gl.PostingService, error) {
	currencyRegistry, err := money.NewCurrencyRegistry([]money.CurrencyMetadata{
		{Code: "USD", Scale: 2},
		{Code: "EUR", Scale: 2},
		{Code: "GBP", Scale: 2},
		{Code: "SGD", Scale: 2},
		{Code: "VND", Scale: 0},
		{Code: "JPY", Scale: 0},
	})
	if err != nil {
		return nil, err
	}
	repository := gl.NewMemoryPostingRepository()
	audit := &gl.MemoryPostingAuditRecorder{}
	return gl.NewPostingService(
		repository,
		gl.MemoryPostingAuthorizer{Decision: gl.PostingAuthorizationDecision{Allowed: true, PolicyReference: "memory-permissive", PolicyVersion: "memory-v1"}},
		gl.AllowAllPostingReferenceValidator{},
		gl.MemoryPostingGateValidator{},
		gl.AllowAllPostingApprovalValidator{},
		audit,
		currencyRegistry,
		time.Now,
	)
}
