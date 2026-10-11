package main

import (
	"time"

	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/platform/money"
)

func newMemoryPostingService() (*gl.PostingService, error) {
	postingService, _, _, err := newMemoryPostingServices()
	return postingService, err
}

func newMemoryPostingServices() (*gl.PostingService, *gl.PostingApprovalDecisionService, *gl.JournalReversalService, error) {
	currencyRegistry, err := money.NewCurrencyRegistry([]money.CurrencyMetadata{
		{Code: "USD", Scale: 2},
		{Code: "EUR", Scale: 2},
		{Code: "GBP", Scale: 2},
		{Code: "SGD", Scale: 2},
		{Code: "VND", Scale: 0},
		{Code: "JPY", Scale: 0},
	})
	if err != nil {
		return nil, nil, nil, err
	}
	repository := gl.NewMemoryPostingRepository()
	audit := &gl.MemoryPostingAuditRecorder{}
	postingAuthorizer := gl.MemoryPostingAuthorizer{Decision: gl.PostingAuthorizationDecision{Allowed: true, PolicyReference: "memory-permissive", PolicyVersion: "memory-v1"}}
	postingService, err := gl.NewPostingService(
		repository,
		postingAuthorizer,
		gl.AllowAllPostingReferenceValidator{},
		gl.MemoryPostingGateValidator{},
		gl.AllowAllPostingApprovalValidator{},
		audit,
		currencyRegistry,
		time.Now,
		gl.AllowAllSoftClosePolicyEvaluator{},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	approvalService, err := gl.NewPostingApprovalDecisionService(
		repository,
		gl.MemoryPostingApprovalDecisionVerifier{},
		gl.MemoryPostingApprovalDecisionAuthorizer{Decision: gl.PostingApprovalDecisionAuthorization{Allowed: true, PolicyReference: "memory-permissive", PolicyVersion: "memory-v1"}},
		postingAuthorizer,
		gl.AllowAllPostingReferenceValidator{},
		gl.MemoryPostingGateValidator{},
		currencyRegistry,
		time.Now,
		gl.AllowAllSoftClosePolicyEvaluator{},
	)
	if err != nil {
		return nil, nil, nil, err
	}
	reversalService, err := gl.NewJournalReversalService(
		repository,
		gl.MemoryJournalReversalAuthorizer{Decision: postingAuthorizer.Decision},
		gl.AllowAllPostingReferenceValidator{},
		gl.MemoryPostingGateValidator{},
		currencyRegistry,
		time.Now,
	)
	if err != nil {
		return nil, nil, nil, err
	}
	return postingService, approvalService, reversalService, nil
}
