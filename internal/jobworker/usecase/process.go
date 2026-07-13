package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aralary/edgeguard/internal/jobworker/domain"
	platformjobs "github.com/aralary/edgeguard/internal/platform/jobs"
)

var ErrInvalidJobPayload = errors.New("invalid job payload")

func (u *Usecase) Process(ctx context.Context, envelope platformjobs.Envelope) (domain.Result, error) {
	switch envelope.Type {
	case platformjobs.TypeWebhookDeliver:
		var payload platformjobs.WebhookPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return domain.Result{}, err
		}
		if err := u.webhookSender.Deliver(ctx, payload); err != nil {
			return domain.Result{}, err
		}

		return domain.Result{Message: "webhook delivered"}, nil

	case platformjobs.TypeReportGenerate:
		var payload platformjobs.ReportPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return domain.Result{}, err
		}
		outputPath, rowCount, err := u.reportGenerator.Generate(ctx, envelope.ID, payload)
		if err != nil {
			return domain.Result{}, err
		}

		return domain.Result{
			Message:      "report generated",
			OutputPath:   outputPath,
			AffectedRows: rowCount,
		}, nil

	case platformjobs.TypeCleanupExpiredTokens:
		var payload platformjobs.CleanupPayload
		if err := decodePayload(envelope.Payload, &payload); err != nil {
			return domain.Result{}, err
		}
		if payload.Before.IsZero() || payload.BatchSize < 1 || payload.BatchSize > 5000 {
			return domain.Result{}, domain.Permanent(ErrInvalidJobPayload)
		}
		deleted, err := u.tokenCleaner.CleanupExpiredTokens(ctx, payload)
		if err != nil {
			return domain.Result{}, err
		}

		return domain.Result{
			Message:      "expired refresh tokens cleaned",
			AffectedRows: deleted,
		}, nil

	default:
		return domain.Result{}, domain.Permanent(fmt.Errorf("unsupported job type %q", envelope.Type))
	}
}

func decodePayload(raw json.RawMessage, destination any) error {
	if len(raw) == 0 || !json.Valid(raw) {
		return domain.Permanent(ErrInvalidJobPayload)
	}
	if err := json.Unmarshal(raw, destination); err != nil {
		return domain.Permanent(fmt.Errorf("%w: %v", ErrInvalidJobPayload, err))
	}

	return nil
}
