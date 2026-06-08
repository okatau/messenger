package service

import (
	"context"
	"fmt"
	"log/slog"
	"maps"

	"presence_service/internal/domain"
	"presence_service/internal/repository"
	sl "presence_service/pkg/service_logger"
)

const svcName = "service.presence"

type Presence interface {
	MarkOnline(ctx context.Context, userID string, metadata map[string]string) error
	Heartbeat(ctx context.Context, userID string) error
	GetStatus(ctx context.Context, userID string) (map[string]string, error)
	GetBulkStatus(ctx context.Context, userIDs []string) ([]*domain.UserStatus, error)
}

type presence struct {
	pRepo  repository.PresenceRepository
	logger *slog.Logger
}

func New(pRepo repository.PresenceRepository, logger *slog.Logger) Presence {
	return &presence{
		pRepo:  pRepo,
		logger: logger,
	}
}

func (svc *presence) MarkOnline(ctx context.Context, userID string, metadata map[string]string) error {
	l := svc.loggerWith(".markonline")

	key := presenceKey(userID)

	fields := make(map[string]string, len(metadata)+1)
	maps.Copy(fields, metadata)
	fields["status"] = domain.StatusOnline

	err := svc.pRepo.Add(ctx, key, fields)
	if err != nil {
		l.Error("failed to mark as online", sl.Err(err))
		return err
	}

	return nil
}

func (svc *presence) Heartbeat(ctx context.Context, userID string) error {
	l := svc.loggerWith(".heartbeat")

	key := presenceKey(userID)

	ok, err := svc.pRepo.Update(ctx, key)
	if err != nil {
		l.Error("failed to update heartbeat", sl.Err(err))
		return err
	}
	if !ok {
		return domain.ErrUserOffline
	}
	return nil
}

func (svc *presence) GetStatus(ctx context.Context, userID string) (map[string]string, error) {
	l := svc.loggerWith(".getstatus")

	key := presenceKey(userID)

	metadata, err := svc.pRepo.Get(ctx, key)
	if err != nil {
		l.Error("failed to get user status", sl.Err(err))
		return nil, err
	}
	if len(metadata) == 0 {
		return nil, domain.ErrUserOffline
	}

	return metadata, nil
}

func (svc *presence) GetBulkStatus(ctx context.Context, userIDs []string) ([]*domain.UserStatus, error) {
	l := svc.loggerWith(".getbulkstatus")

	keys := make([]string, len(userIDs))
	for i := range userIDs {
		keys[i] = presenceKey(userIDs[i])
	}

	res, err := svc.pRepo.GetBulk(ctx, keys)
	if err != nil {
		l.Error("failed to get status bulk", sl.Err(err))
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

func (svc *presence) loggerWith(fnName string) *slog.Logger {
	return svc.logger.With("op", svcName+fnName)
}

func presenceKey(userID string) string {
	return fmt.Sprintf("presence:%s:status", userID)
}
