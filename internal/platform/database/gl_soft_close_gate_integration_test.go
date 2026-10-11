//go:build integration

package database

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/gl"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

func TestGLSoftCloseGateRepositoryEntersExitsAtomicallyAndReplays(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	scopeID, periodID, runID := uuid.New(), uuid.New(), uuid.New()
	gate := gl.NewOpenMemorySoftCloseGate(scopeID, periodID, 1)
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO gl.period_posting_gate (period_posting_gate_id, accounting_scope_id, fiscal_period_id, gate_mode, period_state_version, gate_version, aggregate_version, next_ledger_position, soft_close_restrictions, active_gate_admission_counters, frozen_admission_summary_history, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10::jsonb,$11::jsonb,$12,$12)`, gate.ID, gate.AccountingScopeID, gate.FiscalPeriodID, gate.GateMode, gate.PeriodStateVersion, gate.GateVersion, gate.AggregateVersion, gate.NextLedgerPosition, `{}`, `{}`, `[]`, gate.CreatedAt); err != nil {
		t.Fatal(err)
	}

	auditReference := uuid.New()
	auditCalls := 0
	repository, err := gl.NewPostgresSoftCloseGateRepository(fixture.pool, func(_ context.Context, _ pgx.Tx, _ gl.SoftCloseGateAuditRecord) (uuid.UUID, error) {
		auditCalls++
		return auditReference, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := gl.NewSoftCloseGateServiceWithDurableIdempotency(
		repository,
		gl.MemorySoftCloseGateAuthorizer{Decision: gl.AuthorizationDecision{Allowed: true, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}, PolicyReference: "integration-gate-policy", PolicyVersion: "integration-gate-v1"}},
		gl.AllowAllSoftClosePolicyEvaluator{},
		func() time.Time { return time.Date(2026, 10, 11, 7, 30, 0, 0, time.UTC) },
		gl.DurableSoftCloseGateServiceConfig{Database: fixture.pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, EnterOperationID: gl.SoftCloseGateEnterCommandIdentity, ExitOperationID: gl.SoftCloseGateExitCommandIdentity},
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := gl.Actor{UserID: uuid.New(), SubjectReference: "soft-close-integration-actor"}
	enter := gl.EnterSoftCloseGateCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 1, SoftCloseRunID: runID, ControlEpoch: 4, PolicyVersion: "soft-close-policy-v1", Restrictions: json.RawMessage(`{"ordinaryPosting":"evaluator-controlled"}`), IdempotencyKey: "integration-soft-close-enter", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	first, err := service.EnterSoftCloseGate(ctx, actor, enter)
	if err != nil {
		t.Fatal(err)
	}
	if first.Gate.GateMode != gl.PostingGateSoftClosePolicy || first.Gate.GateVersion != 2 || first.AuditReference != auditReference {
		t.Fatalf("entry result = %#v", first)
	}
	if auditCalls != 1 {
		t.Fatalf("audit calls after entry = %d, want 1", auditCalls)
	}
	var mode string
	var storedRun uuid.UUID
	var epoch, gateVersion int64
	var storedRestrictions []byte
	if err := fixture.pool.QueryRow(ctx, `SELECT gate_mode, active_soft_close_run_id, active_soft_close_control_epoch, gate_version, soft_close_restrictions FROM gl.period_posting_gate WHERE accounting_scope_id = $1 AND fiscal_period_id = $2`, scopeID, periodID).Scan(&mode, &storedRun, &epoch, &gateVersion, &storedRestrictions); err != nil {
		t.Fatal(err)
	}
	var storedPolicy map[string]string
	if err := json.Unmarshal(storedRestrictions, &storedPolicy); err != nil {
		t.Fatal(err)
	}
	if mode != gl.PostingGateSoftClosePolicy || storedRun != runID || epoch != 4 || gateVersion != 2 || storedPolicy["ordinaryPosting"] != "evaluator-controlled" {
		t.Fatalf("stored entry = mode %s, run %s, epoch %d, version %d, restrictions %s", mode, storedRun, epoch, gateVersion, storedRestrictions)
	}
	var enteredOutbox int
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM integration.outbox WHERE aggregate_id = $1 AND event_type = $2 AND source_context = 'gl'`, gate.ID, gl.SoftCloseGateEnteredEventType).Scan(&enteredOutbox); err != nil {
		t.Fatal(err)
	}
	if enteredOutbox != 1 {
		t.Fatalf("entry outbox rows = %d, want 1", enteredOutbox)
	}

	replay, err := service.EnterSoftCloseGate(ctx, actor, enter)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.Gate.GateVersion != first.Gate.GateVersion || auditCalls != 1 {
		t.Fatalf("replay = %#v, audit calls = %d", replay, auditCalls)
	}
	changed := enter
	changed.PolicyVersion = "different-policy"
	if _, err := service.EnterSoftCloseGate(ctx, actor, changed); !errors.Is(err, gl.ErrSoftCloseGateIdempotencyConflict) {
		t.Fatalf("changed retry error = %v", err)
	}

	exit, err := service.ExitSoftCloseGate(ctx, actor, gl.ExitSoftCloseGateCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 2, SoftCloseRunID: runID, ControlEpoch: 4, IdempotencyKey: "integration-soft-close-exit", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	if exit.Gate.GateMode != gl.PostingGateOpen || exit.Gate.GateVersion != 3 || exit.Gate.LastFrozenAdmissionSummary == nil || exit.Gate.LastFrozenAdmissionSummary.FrozenAtGateVersion != 3 {
		t.Fatalf("exit result = %#v", exit)
	}
	var activeRun *uuid.UUID
	var activeEpoch *int64
	var frozenSummary []byte
	if err := fixture.pool.QueryRow(ctx, `SELECT gate_mode, active_soft_close_run_id, active_soft_close_control_epoch, last_frozen_admission_summary FROM gl.period_posting_gate WHERE accounting_scope_id = $1 AND fiscal_period_id = $2`, scopeID, periodID).Scan(&mode, &activeRun, &activeEpoch, &frozenSummary); err != nil {
		t.Fatal(err)
	}
	if mode != gl.PostingGateOpen || activeRun != nil || activeEpoch != nil || len(frozenSummary) == 0 || string(frozenSummary) == "null" {
		t.Fatalf("stored exit = mode %s, run %v, epoch %v, summary %s", mode, activeRun, activeEpoch, frozenSummary)
	}
	var exitedOutbox int
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM integration.outbox WHERE aggregate_id = $1 AND event_type = $2 AND source_context = 'gl'`, gate.ID, gl.SoftCloseGateExitedEventType).Scan(&exitedOutbox); err != nil {
		t.Fatal(err)
	}
	if exitedOutbox != 1 {
		t.Fatalf("exit outbox rows = %d, want 1", exitedOutbox)
	}
}

func TestGLSoftCloseGateRepositoryRollsBackOnAuditFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	scopeID, periodID, runID := uuid.New(), uuid.New(), uuid.New()
	gate := gl.NewOpenMemorySoftCloseGate(scopeID, periodID, 1)
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO gl.period_posting_gate (period_posting_gate_id, accounting_scope_id, fiscal_period_id, gate_mode, period_state_version, gate_version, aggregate_version, next_ledger_position, soft_close_restrictions, active_gate_admission_counters, frozen_admission_summary_history, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10::jsonb,$11::jsonb,$12,$12)`, gate.ID, gate.AccountingScopeID, gate.FiscalPeriodID, gate.GateMode, gate.PeriodStateVersion, gate.GateVersion, gate.AggregateVersion, gate.NextLedgerPosition, `{}`, `{}`, `[]`, gate.CreatedAt); err != nil {
		t.Fatal(err)
	}
	repository, err := gl.NewPostgresSoftCloseGateRepository(fixture.pool, func(context.Context, pgx.Tx, gl.SoftCloseGateAuditRecord) (uuid.UUID, error) {
		return uuid.Nil, gl.ErrSoftCloseGateAuditUnavailable
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := gl.NewSoftCloseGateService(repository, gl.MemorySoftCloseGateAuthorizer{Decision: gl.AuthorizationDecision{Allowed: true, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, gl.AllowAllSoftClosePolicyEvaluator{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.EnterSoftCloseGate(ctx, gl.Actor{UserID: uuid.New(), SubjectReference: "audit-failure-actor"}, gl.EnterSoftCloseGateCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 1, SoftCloseRunID: runID, ControlEpoch: 1, PolicyVersion: "policy-v1", Restrictions: json.RawMessage(`{}`), IdempotencyKey: "audit-failure", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()})
	if !errors.Is(err, gl.ErrSoftCloseGateAuditUnavailable) {
		t.Fatalf("audit failure = %v", err)
	}
	var mode string
	var gateVersion int64
	if err := fixture.pool.QueryRow(ctx, `SELECT gate_mode, gate_version FROM gl.period_posting_gate WHERE accounting_scope_id = $1 AND fiscal_period_id = $2`, scopeID, periodID).Scan(&mode, &gateVersion); err != nil {
		t.Fatal(err)
	}
	if mode != gl.PostingGateOpen || gateVersion != 1 {
		t.Fatalf("gate after audit failure = mode %s, version %d", mode, gateVersion)
	}
}

func TestGLPostingBarrierRepositoryTransfersReleasesAndFinalizes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	scopeID, periodID := uuid.New(), uuid.New()
	softCloseRunID, firstCloseRunID, secondCloseRunID := uuid.New(), uuid.New(), uuid.New()
	gate := gl.NewOpenMemorySoftCloseGate(scopeID, periodID, 1)
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO gl.period_posting_gate (period_posting_gate_id, accounting_scope_id, fiscal_period_id, gate_mode, period_state_version, gate_version, aggregate_version, next_ledger_position, soft_close_restrictions, active_gate_admission_counters, frozen_admission_summary_history, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10::jsonb,$11::jsonb,$12,$12)`, gate.ID, gate.AccountingScopeID, gate.FiscalPeriodID, gate.GateMode, gate.PeriodStateVersion, gate.GateVersion, gate.AggregateVersion, gate.NextLedgerPosition, `{}`, `{}`, `[]`, gate.CreatedAt); err != nil {
		t.Fatal(err)
	}

	auditReference := uuid.New()
	auditCalls := 0
	repository, err := gl.NewPostgresSoftCloseGateRepository(fixture.pool, func(_ context.Context, _ pgx.Tx, _ gl.SoftCloseGateAuditRecord) (uuid.UUID, error) {
		auditCalls++
		return auditReference, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	service, err := gl.NewSoftCloseGateServiceWithDurableIdempotency(
		repository,
		gl.MemorySoftCloseGateAuthorizer{Decision: gl.AuthorizationDecision{Allowed: true, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}, PolicyReference: "integration-gate-policy", PolicyVersion: "integration-gate-v1"}},
		gl.AllowAllSoftClosePolicyEvaluator{},
		func() time.Time { return time.Date(2026, 10, 11, 7, 30, 0, 0, time.UTC) },
		gl.DurableSoftCloseGateServiceConfig{
			Database: fixture.pool, Coordinator: platformidempotency.NewPostgresCoordinator(),
			Policy:           platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute},
			EnterOperationID: gl.SoftCloseGateEnterCommandIdentity, ExitOperationID: gl.SoftCloseGateExitCommandIdentity,
			AcquireOperationID: gl.PostingBarrierAcquireCommandIdentity, ReleaseOperationID: gl.PostingBarrierReleaseCommandIdentity, FinalizeOperationID: gl.PostingBarrierFinalizeCommandIdentity,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := gl.Actor{UserID: uuid.New(), SubjectReference: "posting-barrier-integration-actor"}

	if _, err := service.EnterSoftCloseGate(ctx, actor, gl.EnterSoftCloseGateCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 1, SoftCloseRunID: softCloseRunID, ControlEpoch: 1, PolicyVersion: "soft-close-policy-v1", Restrictions: json.RawMessage(`{"ordinaryPosting":"evaluator-controlled"}`), IdempotencyKey: "integration-barrier-enter", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}); err != nil {
		t.Fatal(err)
	}
	acquire := gl.AcquirePostingBarrierCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 2, SoftCloseRunID: softCloseRunID, SoftCloseControlEpoch: 1, SoftClosePolicyVersion: "soft-close-policy-v1", CloseRunID: firstCloseRunID, CloseType: gl.CloseTypeInitialHardClose, IdempotencyKey: "integration-barrier-acquire-1", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	acquired, err := service.AcquirePostingBarrier(ctx, actor, acquire)
	if err != nil {
		t.Fatal(err)
	}
	if acquired.Gate.GateMode != gl.PostingGateCloseOnly || acquired.Gate.ActiveCloseRunID == nil || *acquired.Gate.ActiveCloseRunID != firstCloseRunID || acquired.Gate.PriorControlProcessID == nil || *acquired.Gate.PriorControlProcessID != softCloseRunID || acquired.Gate.BarrierLedgerPosition != 0 {
		t.Fatalf("acquired gate = %#v", acquired.Gate)
	}

	release, err := service.ReleasePostingBarrier(ctx, actor, gl.ReleasePostingBarrierCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 3, CloseRunID: firstCloseRunID, IdempotencyKey: "integration-barrier-release-1", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	if release.Gate.GateMode != gl.PostingGateSoftClosePolicy || release.Gate.ActiveSoftCloseRunID == nil || *release.Gate.ActiveSoftCloseRunID != softCloseRunID || release.Gate.ActiveSoftCloseControlEpoch != 2 || release.Gate.ActiveCloseRunID != nil {
		t.Fatalf("released gate = %#v", release.Gate)
	}

	secondAcquire, err := service.AcquirePostingBarrier(ctx, actor, gl.AcquirePostingBarrierCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 4, SoftCloseRunID: softCloseRunID, SoftCloseControlEpoch: 2, SoftClosePolicyVersion: "soft-close-policy-v1", CloseRunID: secondCloseRunID, CloseType: gl.CloseTypeInitialHardClose, IdempotencyKey: "integration-barrier-acquire-2", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	if secondAcquire.Gate.GateVersion != 5 || len(secondAcquire.Gate.FrozenAdmissionSummaryHistory) != 3 {
		t.Fatalf("second acquisition = %#v", secondAcquire.Gate)
	}

	finalize := gl.FinalizePostingGateCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 5, CloseRunID: secondCloseRunID, FinalLedgerWatermark: 3, IdempotencyKey: "integration-barrier-finalize-1", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	finalized, err := service.FinalizePostingGate(ctx, actor, finalize)
	if err != nil {
		t.Fatal(err)
	}
	if finalized.Gate.GateMode != gl.PostingGateHardClosed || finalized.Gate.GateVersion != 6 || finalized.Gate.ActiveCloseRunID != nil || finalized.Gate.FinalizedCloseRunID == nil || *finalized.Gate.FinalizedCloseRunID != secondCloseRunID || finalized.Gate.FinalLedgerWatermark == nil || *finalized.Gate.FinalLedgerWatermark != 3 {
		t.Fatalf("finalized gate = %#v", finalized.Gate)
	}
	if auditCalls != 5 {
		t.Fatalf("audit calls before replay = %d, want 5", auditCalls)
	}

	replay, err := service.FinalizePostingGate(ctx, actor, finalize)
	if err != nil || !replay.Replayed || replay.Gate.FinalizationCommandFingerprint != finalized.Gate.FinalizationCommandFingerprint {
		t.Fatalf("finalize replay = %#v, err = %v", replay, err)
	}
	if auditCalls != 5 {
		t.Fatalf("audit calls after replay = %d, want 5", auditCalls)
	}

	var mode string
	var activeClose, finalizedOwner *uuid.UUID
	var finalWatermark *int64
	var finalizationFingerprint string
	var historyJSON []byte
	if err := fixture.pool.QueryRow(ctx, `SELECT gate_mode, active_close_run_id, final_ledger_watermark, finalized_close_run_id, finalization_command_fingerprint, frozen_admission_summary_history FROM gl.period_posting_gate WHERE accounting_scope_id = $1 AND fiscal_period_id = $2`, scopeID, periodID).Scan(&mode, &activeClose, &finalWatermark, &finalizedOwner, &finalizationFingerprint, &historyJSON); err != nil {
		t.Fatal(err)
	}
	var history []gl.FrozenGateAdmissionSummary
	if err := json.Unmarshal(historyJSON, &history); err != nil {
		t.Fatal(err)
	}
	if mode != gl.PostingGateHardClosed || activeClose != nil || finalWatermark == nil || *finalWatermark != 3 || finalizedOwner == nil || *finalizedOwner != secondCloseRunID || finalizationFingerprint == "" || len(history) != 4 {
		t.Fatalf("stored final gate = mode %s, active close %v, watermark %v, finalized owner %v, fingerprint %q, history %d", mode, activeClose, finalWatermark, finalizedOwner, finalizationFingerprint, len(history))
	}
	var finalEvents int
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM integration.outbox WHERE aggregate_id = $1 AND source_context = 'gl' AND event_type = $2`, gate.ID, gl.PostingGateFinalizedEventType).Scan(&finalEvents); err != nil {
		t.Fatal(err)
	}
	if finalEvents != 1 {
		t.Fatalf("finalization outbox rows = %d, want 1", finalEvents)
	}
}
