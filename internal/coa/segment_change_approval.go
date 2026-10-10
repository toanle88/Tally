package coa

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

const (
	SegmentChangeApprovalDecisionPermission = "finance.coa.apply.segment.change.approval.decision"
	SegmentChangeApprovalOutcomeApproved    = "approved"
	SegmentChangeApprovalOutcomeRejected    = "rejected"
	SegmentChangeApprovalAction             = "apply-approval-decision"
)

type SegmentChangeApprovalDecisionReference struct {
	ApprovalRequestID    uuid.UUID
	DecisionID           uuid.UUID
	PolicyVersion        string
	DecisionVersion      int64
	SubjectVersion       aggregateversion.AggregateVersion
	CandidateFingerprint string
	ApproverUserID       uuid.UUID
	DecidedAt            time.Time
}

func (reference SegmentChangeApprovalDecisionReference) Canonical() SegmentChangeApprovalDecisionReference {
	reference.PolicyVersion = strings.TrimSpace(reference.PolicyVersion)
	reference.CandidateFingerprint = strings.TrimSpace(reference.CandidateFingerprint)
	reference.DecidedAt = reference.DecidedAt.UTC()
	return reference
}

func (reference SegmentChangeApprovalDecisionReference) Validate() error {
	reference = reference.Canonical()
	if reference.ApprovalRequestID == uuid.Nil || reference.DecisionID == uuid.Nil || reference.ApproverUserID == uuid.Nil {
		return fmt.Errorf("%w: approval and approver identifiers are required", ErrSegmentChangeApprovalDecisionReferenceInvalid)
	}
	if reference.DecisionVersion < 1 || reference.SubjectVersion.Value() < 1 || reference.PolicyVersion == "" || reference.CandidateFingerprint == "" || reference.DecidedAt.IsZero() {
		return fmt.Errorf("%w: approval metadata is incomplete", ErrSegmentChangeApprovalDecisionReferenceInvalid)
	}
	return nil
}

type SegmentChangeApprovalDecisionCommand struct {
	ScopeID                uuid.UUID
	SegmentChangeRequestID uuid.UUID
	Outcome                string
	Approval               SegmentChangeApprovalDecisionReference
	IdempotencyKey         string
	CorrelationID          string
	CausationID            string
}

func (command SegmentChangeApprovalDecisionCommand) Canonical() SegmentChangeApprovalDecisionCommand {
	command.Outcome = strings.ToLower(strings.TrimSpace(command.Outcome))
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	command.Approval = command.Approval.Canonical()
	return command
}

func (command SegmentChangeApprovalDecisionCommand) Validate() error {
	command = command.Canonical()
	if command.ScopeID == uuid.Nil || command.SegmentChangeRequestID == uuid.Nil {
		return fmt.Errorf("%w: scope and request identifiers are required", ErrSegmentChangeApprovalDecisionInvalid)
	}
	if command.Outcome != SegmentChangeApprovalOutcomeApproved && command.Outcome != SegmentChangeApprovalOutcomeRejected {
		return fmt.Errorf("%w: outcome must be approved or rejected", ErrSegmentChangeApprovalDecisionInvalid)
	}
	if command.IdempotencyKey == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrSegmentChangeApprovalDecisionInvalid)
	}
	return command.Approval.Validate()
}

type SegmentChangeApprovalDecisionAuthorizer interface {
	AuthorizeSegmentChangeApprovalDecision(context.Context, Actor, SegmentChangeApprovalDecisionCommand, *SegmentChangeRequest) (AuthorizationDecision, error)
}

type SegmentChangeApprovalDecisionResult struct {
	SegmentChangeRequest    SafeSegmentChangeRequest               `json:"segmentChangeRequest"`
	Outcome                 string                                 `json:"outcome"`
	ApplicationStatus       string                                 `json:"applicationStatus"`
	ResultingSubjectVersion int64                                  `json:"resultingSubjectVersion"`
	AppliedSubjectVersion   int64                                  `json:"appliedSubjectVersion"`
	EffectiveDateResult     string                                 `json:"effectiveDateResult"`
	ConflictCode            string                                 `json:"conflictCode,omitempty"`
	RejectionReason         string                                 `json:"rejectionReason,omitempty"`
	Approval                SegmentChangeApprovalDecisionReference `json:"approval"`
	DecisionReference       uuid.UUID                              `json:"decisionReference"`
	PolicyReference         string                                 `json:"policyReference"`
	ValidationOutcome       string                                 `json:"validationOutcome"`
	Replayed                bool                                   `json:"replayed,omitempty"`
}

type SegmentChangeApprovalMutation struct {
	Request       SegmentChangeRequest
	BeforeSubject SegmentChangeRequestSubject
	AfterSubject  SegmentChangeRequestSubject
	Decision      SegmentChangeApprovalDecisionReference
	Outcome       string
	ReplayOnly    bool
	Audit         AuditRecord
}

type SegmentChangeApprovalRepository interface {
	CommitSegmentChangeApproval(context.Context, SegmentChangeApprovalMutation) error
}

type DurableSegmentChangeApprovalRepository interface {
	CommitSegmentChangeApprovalWithIdempotency(context.Context, SegmentChangeApprovalMutation, DurableSegmentChangeApprovalCommit) error
}

type DurableSegmentChangeApprovalCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableSegmentChangeApprovalServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type SegmentChangeApprovalDecisionService struct {
	repository  SegmentChangeApprovalRepository
	requests    SegmentChangeRequestRepository
	authorizer  SegmentChangeApprovalDecisionAuthorizer
	clock       func() time.Time
	durable     *DurableSegmentChangeApprovalServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedSegmentChangeApprovalDecision
}

type storedSegmentChangeApprovalDecision struct {
	fingerprint string
	result      SegmentChangeApprovalDecisionResult
}

func NewSegmentChangeApprovalDecisionService(repository SegmentChangeApprovalRepository, requests SegmentChangeRequestRepository, authorizer SegmentChangeApprovalDecisionAuthorizer, clock func() time.Time) (*SegmentChangeApprovalDecisionService, error) {
	return newSegmentChangeApprovalDecisionService(repository, requests, authorizer, clock, nil)
}

func NewSegmentChangeApprovalDecisionServiceWithDurableIdempotency(repository SegmentChangeApprovalRepository, requests SegmentChangeRequestRepository, authorizer SegmentChangeApprovalDecisionAuthorizer, clock func() time.Time, durable DurableSegmentChangeApprovalServiceConfig) (*SegmentChangeApprovalDecisionService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidSegmentChangeRequestService
	}
	return newSegmentChangeApprovalDecisionService(repository, requests, authorizer, clock, &durable)
}

func newSegmentChangeApprovalDecisionService(repository SegmentChangeApprovalRepository, requests SegmentChangeRequestRepository, authorizer SegmentChangeApprovalDecisionAuthorizer, clock func() time.Time, durable *DurableSegmentChangeApprovalServiceConfig) (*SegmentChangeApprovalDecisionService, error) {
	if repository == nil || requests == nil || authorizer == nil || clock == nil {
		return nil, ErrInvalidSegmentChangeRequestService
	}
	return &SegmentChangeApprovalDecisionService{repository: repository, requests: requests, authorizer: authorizer, clock: clock, durable: durable, idempotency: make(map[string]storedSegmentChangeApprovalDecision)}, nil
}

func (service *SegmentChangeApprovalDecisionService) Execute(ctx context.Context, actor Actor, command SegmentChangeApprovalDecisionCommand) (SegmentChangeApprovalDecisionResult, error) {
	if service == nil {
		return SegmentChangeApprovalDecisionResult{}, ErrInvalidSegmentChangeRequestService
	}
	if err := actor.Validate(); err != nil {
		return SegmentChangeApprovalDecisionResult{}, ErrSegmentChangeApprovalDecisionAuthorizationDenied
	}
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SegmentChangeApprovalDecisionResult{}, err
	}
	fingerprint, err := segmentChangeApprovalDecisionFingerprint(command)
	if err != nil {
		return SegmentChangeApprovalDecisionResult{}, err
	}

	key := actor.UserID.String() + ":" + command.IdempotencyKey
	var acquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		scope, marshalErr := json.Marshal(struct {
			Module string    `json:"module"`
			Actor  uuid.UUID `json:"actorId"`
			Scope  uuid.UUID `json:"scopeId"`
		}{"coa", actor.UserID, command.ScopeID})
		if marshalErr != nil {
			return SegmentChangeApprovalDecisionResult{}, marshalErr
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scope), command.IdempotencyKey)
		if err != nil {
			return SegmentChangeApprovalDecisionResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return SegmentChangeApprovalDecisionResult{}, ErrSegmentChangeApprovalDecisionIdempotencyConflict
		}
		if err != nil {
			return SegmentChangeApprovalDecisionResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return SegmentChangeApprovalDecisionResult{}, ErrSegmentChangeApprovalDecisionCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return SegmentChangeApprovalDecisionResult{}, ErrSegmentChangeApprovalDecisionDurableCommandFailed
			}
			result, decodeErr := decodeDurableSegmentChangeApprovalDecisionResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return SegmentChangeApprovalDecisionResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return SegmentChangeApprovalDecisionResult{}, ErrSegmentChangeApprovalDecisionIdempotencyConflict
			}
			result := previous.result
			result.Replayed = true
			return result, nil
		}
	}

	request, err := service.requests.Get(ctx, command.SegmentChangeRequestID)
	if errors.Is(err, ErrSegmentChangeRequestSubjectNotFound) {
		err = ErrSegmentChangeApprovalDecisionNotFound
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeApprovalDecisionResult{}, err
	}
	if request.ScopeID != command.ScopeID {
		err = ErrSegmentChangeApprovalDecisionAuthorizationDenied
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeApprovalDecisionResult{}, err
	}

	decision, err := service.authorizer.AuthorizeSegmentChangeApprovalDecision(ctx, actor, command, &request)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeApprovalDecisionResult{}, err
	}
	if err := authorizeSegmentChangeApprovalDecision(decision, command.ScopeID); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeApprovalDecisionResult{}, err
	}

	if request.ApprovalStatus != SegmentChangeRequestApprovalPending || request.ApplicationStatus != SegmentChangeRequestApplicationOpen {
		if request.DecisionFingerprint == fingerprint {
			result := segmentChangeApprovalDecisionResult(request, command.Approval, decision)
			result.Replayed = true
			mutation := SegmentChangeApprovalMutation{Request: request, Decision: command.Approval, Outcome: command.Outcome, ReplayOnly: true}
			if err := service.commit(ctx, actor, command, fingerprint, durableIdentity, acquisition, mutation, result); err != nil {
				return SegmentChangeApprovalDecisionResult{}, err
			}
			return result, nil
		}
		err = ErrSegmentChangeApprovalDecisionDuplicate
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeApprovalDecisionResult{}, err
	}
	if request.ApprovalRequestID != command.Approval.ApprovalRequestID || request.SubjectVersion.Value() != command.Approval.SubjectVersion.Value() || request.ProposedFingerprint != command.Approval.CandidateFingerprint {
		err = ErrSegmentChangeApprovalDecisionReferenceInvalid
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeApprovalDecisionResult{}, err
	}

	current, err := service.currentSubject(ctx, request)
	if errors.Is(err, ErrSegmentChangeRequestSubjectNotFound) {
		err = ErrSegmentChangeApprovalDecisionSubjectConflict
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeApprovalDecisionResult{}, err
	}
	if current.ScopeID != request.ScopeID {
		err = ErrSegmentChangeApprovalDecisionAuthorizationDenied
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeApprovalDecisionResult{}, err
	}

	now := service.clock().UTC()
	applicationStatus := SegmentChangeRequestApplicationUnchanged
	conflictCode := ""
	rejectionReason := ""
	after := current
	if current.Version.Value() != request.SubjectVersion.Value() || FingerprintSegmentChangeSubject(current) != request.SubjectFingerprint {
		applicationStatus = SegmentChangeRequestApplicationConflict
		conflictCode = "STALE_SUBJECT"
	} else if command.Outcome == SegmentChangeApprovalOutcomeApproved {
		after, applicationStatus, err = buildApprovedSegmentChangeSubject(request, current, now)
		if err != nil {
			if !isSegmentChangeApprovalConflict(err) {
				service.finalizeDurableFailure(ctx, acquisition, err)
				return SegmentChangeApprovalDecisionResult{}, err
			}
			applicationStatus = SegmentChangeRequestApplicationConflict
			conflictCode = segmentChangeApprovalConflictCode(err)
			after = current
		}
	} else {
		rejectionReason = "APPROVAL_REJECTED"
	}

	terminal := cloneSegmentChangeRequest(request)
	nextVersion, err := request.Version.Advance()
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeApprovalDecisionResult{}, err
	}
	terminal.Version = nextVersion
	terminal.RevisionNumber = request.RevisionNumber + 1
	terminal.ApprovalStatus = command.Outcome
	terminal.ApplicationStatus = applicationStatus
	terminal.ConflictCode = conflictCode
	terminal.RejectionReason = rejectionReason
	terminal.NextAction = SegmentChangeRequestNextActionCompleted
	if applicationStatus == SegmentChangeRequestApplicationConflict {
		terminal.NextAction = SegmentChangeRequestNextActionResolveConflict
	}
	terminal.ApprovalDecisionID = command.Approval.DecisionID
	terminal.ApprovalPolicyVersion = command.Approval.PolicyVersion
	terminal.ApprovalDecisionVersion = command.Approval.DecisionVersion
	terminal.ApprovalSubjectVersion = command.Approval.SubjectVersion
	terminal.ApprovalCandidateFingerprint = command.Approval.CandidateFingerprint
	terminal.ApprovalApproverUserID = command.Approval.ApproverUserID
	terminal.ApprovalDecidedAt = command.Approval.DecidedAt
	terminal.ApprovalAppliedAt = now
	terminal.AppliedSubjectVersion = current.Version
	terminal.ResultingSubjectVersion = current.Version
	if applicationStatus == SegmentChangeRequestApplicationApplied {
		terminal.ResultingSubjectVersion = after.Version
	}
	terminal.DecisionFingerprint = fingerprint
	terminal.UpdatedAt = now
	if err := terminal.Validate(); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeApprovalDecisionResult{}, err
	}

	beforeFingerprint := FingerprintSegmentChangeSubject(current)
	afterFingerprint := beforeFingerprint
	if applicationStatus == SegmentChangeRequestApplicationApplied {
		afterFingerprint = FingerprintSegmentChangeSubject(after)
	}
	audit := AuditRecord{
		SegmentChangeRequestID: terminal.ID, SegmentDefinitionID: subjectDefinitionID(current), SegmentValueID: subjectValueID(current),
		ApprovalRequestID: terminal.ApprovalRequestID, ApprovalDecisionID: command.Approval.DecisionID, ApproverUserID: command.Approval.ApproverUserID,
		ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: SegmentChangeApprovalAction, ScopeID: terminal.ScopeID,
		Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference,
		RevisionNumber: terminal.RevisionNumber, BeforeFingerprint: beforeFingerprint, AfterFingerprint: afterFingerprint,
		CorrelationID: command.CorrelationID, CausationID: command.CausationID,
	}
	result := segmentChangeApprovalDecisionResult(terminal, command.Approval, decision)
	result.Outcome = command.Outcome
	result.ApplicationStatus = applicationStatus
	result.ResultingSubjectVersion = terminal.ResultingSubjectVersion.Value()
	result.AppliedSubjectVersion = terminal.AppliedSubjectVersion.Value()
	result.EffectiveDateResult = "effective"
	result.ConflictCode = conflictCode
	result.RejectionReason = rejectionReason
	mutation := SegmentChangeApprovalMutation{Request: terminal, BeforeSubject: current, AfterSubject: after, Decision: command.Approval, Outcome: command.Outcome, Audit: audit}
	if err := service.commit(ctx, actor, command, fingerprint, durableIdentity, acquisition, mutation, result); err != nil {
		return SegmentChangeApprovalDecisionResult{}, err
	}
	return result, nil
}

func (service *SegmentChangeApprovalDecisionService) currentSubject(ctx context.Context, request SegmentChangeRequest) (SegmentChangeRequestSubject, error) {
	reader, ok := service.requests.(SegmentChangeRequestSubjectReader)
	if !ok {
		return SegmentChangeRequestSubject{}, ErrSegmentChangeApprovalDecisionSubjectConflict
	}
	return reader.GetSegmentChangeSubject(ctx, request.ChangeType, request.SubjectID)
}

func (service *SegmentChangeApprovalDecisionService) commit(ctx context.Context, actor Actor, command SegmentChangeApprovalDecisionCommand, fingerprint string, durableIdentity platformidempotency.IdempotencyIdentity, acquisition platformidempotency.DurableAcquisition, mutation SegmentChangeApprovalMutation, result SegmentChangeApprovalDecisionResult) error {
	if service.durable != nil {
		committer, ok := service.repository.(DurableSegmentChangeApprovalRepository)
		if !ok {
			service.finalizeDurableFailure(ctx, acquisition, ErrInvalidSegmentChangeRequestService)
			return ErrInvalidSegmentChangeRequestService
		}
		body, err := json.Marshal(result)
		if err != nil {
			service.finalizeDurableFailure(ctx, acquisition, err)
			return err
		}
		status := 200
		aggregateID := mutation.Request.ID
		metadata, err := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if err != nil {
			service.finalizeDurableFailure(ctx, acquisition, err)
			return err
		}
		if err := committer.CommitSegmentChangeApprovalWithIdempotency(ctx, mutation, DurableSegmentChangeApprovalCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata}); err != nil {
			service.finalizeDurableFailure(ctx, acquisition, err)
			return err
		}
		return nil
	}
	if err := service.repository.CommitSegmentChangeApproval(ctx, mutation); err != nil {
		return err
	}
	service.idempotency[actor.UserID.String()+":"+command.IdempotencyKey] = storedSegmentChangeApprovalDecision{fingerprint: fingerprint, result: result}
	return nil
}

func authorizeSegmentChangeApprovalDecision(decision AuthorizationDecision, scope uuid.UUID) error {
	if !decision.Allowed || decision.Permission != SegmentChangeApprovalDecisionPermission || decision.DecisionReference == uuid.Nil {
		return ErrSegmentChangeApprovalDecisionAuthorizationDenied
	}
	for _, allowed := range decision.ApprovedScopeIDs {
		if allowed == uuid.Nil || allowed == scope {
			return nil
		}
	}
	return ErrSegmentChangeApprovalDecisionAuthorizationDenied
}

func buildApprovedSegmentChangeSubject(request SegmentChangeRequest, current SegmentChangeRequestSubject, now time.Time) (SegmentChangeRequestSubject, string, error) {
	proposal := request.ProposedChange.Canonical(request.ChangeType)
	if err := validateSegmentChangeRequestProposal(request, current); err != nil {
		return current, SegmentChangeRequestApplicationConflict, err
	}
	switch request.ChangeType {
	case SegmentChangeTypeDefinition:
		if current.Definition == nil {
			return current, SegmentChangeRequestApplicationConflict, ErrSegmentChangeApprovalDecisionSubjectConflict
		}
		if current.Definition.SegmentType == proposal.SegmentType && current.Definition.Code == proposal.Code && current.Definition.Name == proposal.Name && current.Definition.Status == proposal.Status && dateOnly(current.Definition.EffectiveDateFrom).Equal(dateOnly(proposal.EffectiveDateFrom)) && sameDate(current.Definition.EffectiveDateTo, proposal.EffectiveDateTo) {
			return current, SegmentChangeRequestApplicationUnchanged, nil
		}
		after := cloneSegmentDefinition(*current.Definition)
		if err := after.Replace(*current.Definition, proposal.SegmentType, proposal.Code, proposal.Name, proposal.Status, proposal.EffectiveDateFrom, proposal.EffectiveDateTo, now); err != nil {
			return current, SegmentChangeRequestApplicationConflict, err
		}
		return SegmentChangeRequestSubject{ChangeType: request.ChangeType, SubjectID: after.ID, ScopeID: after.ScopeID, Version: after.Version, RevisionNumber: after.RevisionNumber, Definition: &after}, SegmentChangeRequestApplicationApplied, nil
	case SegmentChangeTypeValue:
		if current.Definition == nil || current.Value == nil {
			return current, SegmentChangeRequestApplicationConflict, ErrSegmentChangeApprovalDecisionSubjectConflict
		}
		if current.Value.Value == proposal.Value && current.Value.Description == proposal.Description && current.Value.Status == proposal.Status && dateOnly(current.Value.EffectiveDateFrom).Equal(dateOnly(proposal.EffectiveDateFrom)) && sameDate(current.Value.EffectiveDateTo, proposal.EffectiveDateTo) {
			return current, SegmentChangeRequestApplicationUnchanged, nil
		}
		after := cloneSegmentDefinition(*current.Definition)
		nextVersion, err := after.Version.Advance()
		if err != nil {
			return current, SegmentChangeRequestApplicationConflict, err
		}
		valueAfter := cloneSegmentValue(*current.Value)
		if err := valueAfter.Replace(*current.Value, after, proposal.Value, proposal.Description, proposal.Status, proposal.EffectiveDateFrom, proposal.EffectiveDateTo, now); err != nil {
			return current, SegmentChangeRequestApplicationConflict, err
		}
		after.Version = nextVersion
		after.RevisionNumber++
		after.UpdatedAt = now.UTC()
		for index := range after.Values {
			if after.Values[index].ID == valueAfter.ID {
				after.Values[index] = valueAfter
				break
			}
		}
		sortSegmentValues(after.Values)
		if err := after.Validate(); err != nil {
			return current, SegmentChangeRequestApplicationConflict, err
		}
		after.Revisions = append(cloneRevisions(current.Definition.Revisions), SegmentDefinitionRevision{RevisionNumber: after.RevisionNumber, Version: after.Version, Snapshot: after.Snapshot(), CreatedAt: after.UpdatedAt})
		return SegmentChangeRequestSubject{ChangeType: request.ChangeType, SubjectID: valueAfter.ID, ScopeID: after.ScopeID, Version: after.Version, RevisionNumber: after.RevisionNumber, Definition: &after, Value: &valueAfter}, SegmentChangeRequestApplicationApplied, nil
	default:
		return current, SegmentChangeRequestApplicationConflict, ErrSegmentChangeRequestUnsupportedSubject
	}
}

func validateSegmentChangeRequestProposal(request SegmentChangeRequest, subject SegmentChangeRequestSubject) error {
	return validateSegmentChangeProposal(SegmentChangeRequestCommand{ChangeType: request.ChangeType, SubjectID: request.SubjectID, SubjectVersion: request.SubjectVersion, RequestedEffectiveDate: request.RequestedEffectiveDate, ProposedChange: request.ProposedChange}, subject)
}

func sameDate(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return dateOnly(*left).Equal(dateOnly(*right))
}

func isSegmentChangeApprovalConflict(err error) bool {
	return errors.Is(err, ErrSegmentDefinitionDuplicate) || errors.Is(err, ErrSegmentValueDuplicate) || errors.Is(err, ErrInvalidSegmentDefinition) || errors.Is(err, ErrInvalidSegmentValue) || errors.Is(err, ErrSegmentChangeRequestVersionConflict) || errors.Is(err, ErrSegmentChangeApprovalDecisionSubjectConflict)
}

func segmentChangeApprovalConflictCode(err error) string {
	switch {
	case errors.Is(err, ErrSegmentChangeRequestVersionConflict):
		return "STALE_SUBJECT"
	case errors.Is(err, ErrSegmentDefinitionDuplicate), errors.Is(err, ErrSegmentValueDuplicate):
		return "UNIQUENESS_CONFLICT"
	case errors.Is(err, ErrInvalidSegmentDefinition), errors.Is(err, ErrInvalidSegmentValue):
		return "POLICY_CONFLICT"
	default:
		return "SUBJECT_CONFLICT"
	}
}

func segmentChangeApprovalDecisionResult(request SegmentChangeRequest, approval SegmentChangeApprovalDecisionReference, decision AuthorizationDecision) SegmentChangeApprovalDecisionResult {
	return SegmentChangeApprovalDecisionResult{
		SegmentChangeRequest: request.SafeProjection(), Outcome: request.ApprovalStatus, ApplicationStatus: request.ApplicationStatus,
		ResultingSubjectVersion: request.ResultingSubjectVersion.Value(), AppliedSubjectVersion: request.AppliedSubjectVersion.Value(),
		EffectiveDateResult: "effective", ConflictCode: request.ConflictCode, RejectionReason: request.RejectionReason,
		Approval: approval, DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference, ValidationOutcome: request.ValidationOutcome,
	}
}

func segmentChangeApprovalDecisionFingerprint(command SegmentChangeApprovalDecisionCommand) (string, error) {
	command = command.Canonical()
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (service *SegmentChangeApprovalDecisionService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code := "VALIDATION_FAILED"
	switch {
	case errors.Is(commandErr, ErrSegmentChangeApprovalDecisionAuthorizationDenied):
		code = "AUTHORIZATION_DENIED"
	case errors.Is(commandErr, ErrSegmentChangeApprovalDecisionReferenceInvalid):
		code = "INVALID_APPROVAL_REFERENCE"
	case errors.Is(commandErr, ErrSegmentChangeApprovalDecisionDuplicate):
		code = "DUPLICATE_DECISION"
	case errors.Is(commandErr, ErrSegmentChangeApprovalDecisionNotFound):
		code = "REQUEST_NOT_FOUND"
	case errors.Is(commandErr, ErrSegmentChangeApprovalDecisionSubjectConflict):
		code = "SUBJECT_CONFLICT"
	}
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{code})
	if err != nil {
		return
	}
	initial := acquisition.Result()
	metadata, err := platformidempotency.NewCommandResultMetadata(initial.Identity(), initial.Fingerprint(), initial.OperationID(), platformidempotency.StateFailed, nil, body, nil, nil)
	if err != nil {
		return
	}
	tx, err := service.durable.Database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := service.durable.Coordinator.Finalize(ctx, tx, acquisition, metadata); err != nil {
		return
	}
	if err := tx.Commit(ctx); err != nil {
		return
	}
	committed = true
}

func decodeDurableSegmentChangeApprovalDecisionResult(body []byte) (SegmentChangeApprovalDecisionResult, error) {
	var result SegmentChangeApprovalDecisionResult
	if err := json.Unmarshal(body, &result); err != nil {
		return SegmentChangeApprovalDecisionResult{}, ErrSegmentChangeApprovalDecisionCommandInProgress
	}
	return result, nil
}

func (repository *MemorySegmentChangeRequestRepository) CommitSegmentChangeApproval(ctx context.Context, mutation SegmentChangeApprovalMutation) error {
	return repository.commitSegmentChangeApproval(ctx, mutation)
}

func (repository *MemorySegmentChangeRequestRepository) GetSegmentChangeSubject(ctx context.Context, changeType string, subjectID uuid.UUID) (SegmentChangeRequestSubject, error) {
	if repository == nil || repository.subjects == nil {
		return SegmentChangeRequestSubject{}, ErrInvalidSegmentChangeRequestService
	}
	return repository.subjects.GetSegmentChangeSubject(ctx, changeType, subjectID)
}

func (authorizer MemoryAuthorizer) AuthorizeSegmentChangeApprovalDecision(context.Context, Actor, SegmentChangeApprovalDecisionCommand, *SegmentChangeRequest) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

func (repository *MemorySegmentChangeRequestRepository) CommitSegmentChangeApprovalWithIdempotency(ctx context.Context, mutation SegmentChangeApprovalMutation, _ DurableSegmentChangeApprovalCommit) error {
	return repository.commitSegmentChangeApproval(ctx, mutation)
}

func (repository *MemorySegmentChangeRequestRepository) commitSegmentChangeApproval(ctx context.Context, mutation SegmentChangeApprovalMutation) error {
	if repository == nil || repository.subjects == nil {
		return ErrInvalidSegmentChangeRequestService
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	currentRequest, ok := repository.requests[mutation.Request.ID]
	if !ok {
		return ErrSegmentChangeApprovalDecisionNotFound
	}
	if mutation.ReplayOnly {
		if currentRequest.DecisionFingerprint != mutation.Request.DecisionFingerprint {
			return ErrSegmentChangeApprovalDecisionDuplicate
		}
		return nil
	}
	if currentRequest.ApprovalStatus != SegmentChangeRequestApprovalPending || currentRequest.ApplicationStatus != SegmentChangeRequestApplicationOpen {
		return ErrSegmentChangeApprovalDecisionDuplicate
	}
	if err := mutation.Request.Validate(); err != nil {
		return err
	}
	memoryRepository, ok := repository.subjects.(*MemorySegmentDefinitionRepository)
	if !ok {
		return ErrInvalidSegmentChangeRequestService
	}
	memoryRepository.mu.Lock()
	defer memoryRepository.mu.Unlock()
	currentSubject, err := memorySegmentChangeSubjectLocked(memoryRepository, mutation.Request.ChangeType, mutation.Request.SubjectID)
	if err != nil {
		return mapSegmentChangeRequestSubjectError(err)
	}
	if mutation.Request.ApplicationStatus == SegmentChangeRequestApplicationApplied && (currentSubject.Version.Value() != mutation.BeforeSubject.Version.Value() || FingerprintSegmentChangeSubject(currentSubject) != FingerprintSegmentChangeSubject(mutation.BeforeSubject)) {
		return ErrSegmentChangeApprovalDecisionSubjectConflict
	}
	if mutation.Request.ApplicationStatus == SegmentChangeRequestApplicationApplied {
		if mutation.AfterSubject.Definition == nil {
			return ErrSegmentChangeApprovalDecisionSubjectConflict
		}
		if err := validateMemoryApprovalUniqueness(memoryRepository, mutation.Request.ChangeType, *mutation.AfterSubject.Definition); err != nil {
			return err
		}
	}
	if repository.audit == nil {
		return ErrSegmentChangeApprovalDecisionAuditUnavailable
	}
	if err := repository.audit.RecordSegmentChangeRequestMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	if mutation.Request.ApplicationStatus == SegmentChangeRequestApplicationApplied {
		memoryRepository.definitions[mutation.AfterSubject.Definition.ID] = cloneSegmentDefinition(*mutation.AfterSubject.Definition)
	}
	repository.requests[mutation.Request.ID] = cloneSegmentChangeRequest(mutation.Request)
	return nil
}

func validateMemoryApprovalUniqueness(repository *MemorySegmentDefinitionRepository, changeType string, candidate SegmentDefinition) error {
	if changeType != SegmentChangeTypeDefinition {
		return nil
	}
	for id, current := range repository.definitions {
		if id == candidate.ID || current.ScopeID != candidate.ScopeID || current.SegmentType != candidate.SegmentType || current.Code != candidate.Code {
			continue
		}
		if rangesOverlap(current.EffectiveDateFrom, current.EffectiveDateTo, candidate.EffectiveDateFrom, candidate.EffectiveDateTo) {
			return ErrSegmentDefinitionDuplicate
		}
	}
	return nil
}

func memorySegmentChangeSubjectLocked(repository *MemorySegmentDefinitionRepository, changeType string, subjectID uuid.UUID) (SegmentChangeRequestSubject, error) {
	if changeType == SegmentChangeTypeDefinition {
		definition, ok := repository.definitions[subjectID]
		if !ok {
			return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestSubjectNotFound
		}
		definition = cloneSegmentDefinition(definition)
		return SegmentChangeRequestSubject{ChangeType: changeType, SubjectID: definition.ID, ScopeID: definition.ScopeID, Version: definition.Version, RevisionNumber: definition.RevisionNumber, Definition: &definition}, nil
	}
	if changeType == SegmentChangeTypeValue {
		for _, definition := range repository.definitions {
			for _, value := range definition.Values {
				if value.ID == subjectID {
					parent := cloneSegmentDefinition(definition)
					candidate := cloneSegmentValue(value)
					return SegmentChangeRequestSubject{ChangeType: changeType, SubjectID: candidate.ID, ScopeID: parent.ScopeID, Version: parent.Version, RevisionNumber: parent.RevisionNumber, Definition: &parent, Value: &candidate}, nil
				}
			}
		}
		return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestSubjectNotFound
	}
	return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestUnsupportedSubject
}

var _ SegmentChangeApprovalRepository = (*MemorySegmentChangeRequestRepository)(nil)
var _ DurableSegmentChangeApprovalRepository = (*MemorySegmentChangeRequestRepository)(nil)
