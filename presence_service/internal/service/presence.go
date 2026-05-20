package service

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"presence_service/internal/domain"
	"presence_service/internal/repository"
	"presence_service/pkg/service_logger"
)

type Presence interface {
	MarkOnline(ctx context.Context, userID string, metadata map[string]string) error
	Heartbeat(ctx context.Context, userID string) error
	GetStatus(ctx context.Context, userID string) (map[string]string, error)
	GetBulkStatus(ctx context.Context, userIDs []string) ([]*domain.UserStatus, error)
}

type presenceService struct {
	pRepo  repository.PresenceRepository
	logger *slog.Logger
}

func NewPresenceService(pRepo repository.PresenceRepository, logger *slog.Logger) Presence {
	return &presenceService{
		pRepo:  pRepo,
		logger: logger,
	}
}

func (svc *presenceService) MarkOnline(ctx context.Context, userID string, metadata map[string]string) error {
	const op = "presence.markonline"
	logger := svc.logger.With(slog.String("op", op))

	key := presenceKey(userID)

	fields := make(map[string]string, len(metadata)+1)
	maps.Copy(fields, metadata)
	fields["status"] = domain.StatusOnline

	err := svc.pRepo.Add(ctx, key, fields)
	if err != nil {
		logger.Error("failed to mark as online", service_logger.Err(err))
		return err
	}

	return nil
}

func (svc *presenceService) Heartbeat(ctx context.Context, userID string) error {
	const op = "presence.heartbeat"
	logger := svc.logger.With(slog.String("op", op))

	key := presenceKey(userID)

	ok, err := svc.pRepo.Update(ctx, key)
	if err != nil {
		logger.Error("failed to update heartbeat", service_logger.Err(err))
		return err
	}
	if !ok {
		return domain.ErrUserOffline
	}
	return nil
}

func (svc *presenceService) GetStatus(ctx context.Context, userID string) (map[string]string, error) {
	const op = "presence.getstatus"
	logger := svc.logger.With(slog.String("op", op))

	key := presenceKey(userID)

	metadata, err := svc.pRepo.Get(ctx, key)
	if err != nil {
		logger.Error("failed to get user status", service_logger.Err(err))
		return nil, err
	}
	if len(metadata) == 0 {
		return nil, domain.ErrUserOffline
	}

	return metadata, nil
}

func (svc *presenceService) GetBulkStatus(ctx context.Context, userIDs []string) ([]*domain.UserStatus, error) {
	const op = "presence.getstatus"
	logger := svc.logger.With(slog.String("op", op))

	keys := make([]string, len(userIDs))
	for i := range userIDs {
		keys[i] = presenceKey(userIDs[i])
	}

	res, err := svc.pRepo.GetBulk(ctx, keys)
	if err != nil {
		logger.Error("failed to get status bulk", service_logger.Err(err))
		return nil, err
	}

	statuses := make([]*domain.UserStatus, len(res))
	for i := range res {
		vals, err := res[i].Result()
		if err != nil || len(vals) == 0 {
			statuses[i] = &domain.UserStatus{Status: domain.StatusOffline}
			continue
		}

		metadata := make(map[string]any, len(vals))
		for k, v := range vals {
			if k != "status" {
				metadata[k] = v
			}
		}

		statuses[i] = &domain.UserStatus{Status: domain.StatusOnline, Metadata: metadata}
	}

	return statuses, nil
}

func presenceKey(userID string) string {
	return fmt.Sprintf("presence:%s:status", userID)
}
