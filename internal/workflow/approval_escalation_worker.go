package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	ApprovalEscalationWorkerInterval  = time.Minute
	ApprovalEscalationWorkerBatchSize = 100
)

var ErrInvalidApprovalEscalationWorker = errors.New("invalid approval escalation worker")

type ApprovalEscalationWorkerConfig struct {
	Interval         time.Duration
	BatchSize        int
	ProcessReference string
}

type ApprovalEscalationWorkerBatchResult struct {
	Examined  int
	Escalated int
	Replayed  int
	Skipped   int
}

type ApprovalEscalationWorker struct {
	service          *ApprovalRequestService
	candidates       ApprovalEscalationCandidateReader
	clock            func() time.Time
	interval         time.Duration
	batchSize        int
	processReference string
}

func NewApprovalEscalationWorker(service *ApprovalRequestService, candidates ApprovalEscalationCandidateReader, clock func() time.Time, config ApprovalEscalationWorkerConfig) (*ApprovalEscalationWorker, error) {
	if service == nil || candidates == nil || clock == nil || config.Interval <= 0 || config.BatchSize < 1 || config.BatchSize > 100 || config.ProcessReference != ApprovalEscalationProcess {
		return nil, ErrInvalidApprovalEscalationWorker
	}
	return &ApprovalEscalationWorker{service: service, candidates: candidates, clock: clock, interval: config.Interval, batchSize: config.BatchSize, processReference: config.ProcessReference}, nil
}

func (worker *ApprovalEscalationWorker) Run(ctx context.Context) error {
	if worker == nil || ctx == nil {
		return ErrInvalidApprovalEscalationWorker
	}
	if _, err := worker.RunBatch(ctx); err != nil {
		return err
	}
	ticker := time.NewTicker(worker.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := worker.RunBatch(ctx); err != nil {
				return err
			}
		}
	}
}

func (worker *ApprovalEscalationWorker) RunBatch(ctx context.Context) (ApprovalEscalationWorkerBatchResult, error) {
	if worker == nil || worker.service == nil || worker.candidates == nil || worker.clock == nil || ctx == nil {
		return ApprovalEscalationWorkerBatchResult{}, ErrInvalidApprovalEscalationWorker
	}
	candidates, err := worker.candidates.ListDueApprovalEscalationCandidates(ctx, worker.clock().UTC(), worker.batchSize)
	if err != nil {
		return ApprovalEscalationWorkerBatchResult{}, err
	}
	result := ApprovalEscalationWorkerBatchResult{Examined: len(candidates)}
	for _, candidate := range candidates {
		command := worker.command(candidate)
		escalation, err := worker.service.EscalateApprovalAsProcess(ctx, worker.processReference, command)
		if err != nil {
			if approvalEscalationWorkerRace(err) {
				result.Skipped++
				continue
			}
			return result, fmt.Errorf("escalate approval request %s: %w", candidate.RequestID, err)
		}
		if escalation.Replayed {
			result.Replayed++
		} else {
			result.Escalated++
		}
	}
	return result, nil
}

func (worker *ApprovalEscalationWorker) command(candidate ApprovalEscalationCandidate) EscalateApprovalCommand {
	seed := fmt.Sprintf("%s:%s:%d:%s", candidate.RequestID, candidate.StepID, candidate.ExpectedVersion.Value(), candidate.DueAt.UTC().Format(time.RFC3339Nano))
	identity := "approval-escalation:" + seed
	expected := candidate.ExpectedVersion
	return EscalateApprovalCommand{
		RequestID:            candidate.RequestID,
		ExpectedVersion:      &expected,
		EscalationID:         uuid.NewSHA1(uuid.Nil, []byte(identity+":escalation")),
		StepID:               candidate.StepID,
		Reason:               ApprovalEscalationReasonOverdue,
		PolicyVersion:        candidate.PolicyVersion,
		SubjectVersion:       candidate.SubjectVersion,
		CandidateFingerprint: candidate.CandidateFingerprint,
		IdempotencyKey:       identity,
		CorrelationID:        uuid.NewSHA1(uuid.Nil, []byte(identity+":correlation")).String(),
		CausationID:          uuid.NewSHA1(uuid.Nil, []byte(identity+":causation")).String(),
	}
}

func approvalEscalationWorkerRace(err error) bool {
	return errors.Is(err, ErrApprovalPolicyVersionConflict) ||
		errors.Is(err, ErrApprovalPolicySubjectConflict) ||
		errors.Is(err, ErrApprovalPolicyStepConflict) ||
		errors.Is(err, ErrApprovalPolicyEscalationNotEligible) ||
		errors.Is(err, ErrApprovalPolicyAlreadyEscalated) ||
		errors.Is(err, ErrApprovalPolicyApprovalRejected)
}
