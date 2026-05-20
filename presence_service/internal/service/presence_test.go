package service

import (
	"context"
	"errors"
	"log/slog"
	"presence_service/internal/domain"
	"presence_service/internal/mocks"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	redisError = errors.New("redis error")
)

func Test_MarkOnline(t *testing.T) {
	repo := mocks.NewMockPresenceRepository(t)
	svc := NewPresenceService(repo, slog.Default())

	t.Run("Successfully addede status", func(t *testing.T) {
		userID := "test-case-1"
		key := presenceKey(userID)
		emptyMD := map[string]string{}
		repo.EXPECT().Add(mock.Anything, key, mock.Anything).Return(nil)

		err := svc.MarkOnline(t.Context(), userID, emptyMD)
		require.NoError(t, err)
	})

	t.Run("redis error", func(t *testing.T) {
		userID := "test-case-2"
		key := presenceKey(userID)
		emptyMD := map[string]string{}

		repo.EXPECT().Add(mock.Anything, key, mock.Anything).Return(redisError)

		err := svc.MarkOnline(t.Context(), userID, emptyMD)
		require.ErrorIs(t, err, redisError)
	})
}

func Test_Heartbeat(t *testing.T) {
	userID := "test-heartbeat"
	key := presenceKey(userID)

	tests := []struct {
		name    string
		key     string
		setup   func(repo *mocks.MockPresenceRepository)
		wantErr error
	}{
		{
			name: "Successfully updated",
			setup: func(repo *mocks.MockPresenceRepository) {
				repo.EXPECT().Update(mock.Anything, key).Return(true, nil)
			},
		},
		{
			name: "user offline",
			setup: func(repo *mocks.MockPresenceRepository) {
				repo.EXPECT().Update(mock.Anything, key).Return(false, nil)
			},
			wantErr: domain.ErrUserOffline,
		},
		{
			name: "redis error",
			setup: func(repo *mocks.MockPresenceRepository) {
				repo.EXPECT().Update(mock.Anything, key).Return(false, redisError)
			},
			wantErr: redisError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockPresenceRepository(t)
			svc := NewPresenceService(repo, slog.Default())
			tt.setup(repo)

			err := svc.Heartbeat(t.Context(), userID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_GetStatus(t *testing.T) {
	userID := "test-getstatus"
	key := presenceKey(userID)
	onlineMap := map[string]string{"status": domain.StatusOnline}

	tests := []struct {
		name    string
		key     string
		setup   func(repo *mocks.MockPresenceRepository)
		wantErr error
	}{
		{
			name: "Successfully got",
			setup: func(repo *mocks.MockPresenceRepository) {
				repo.EXPECT().Get(mock.Anything, key).Return(onlineMap, nil)
			},
		},
		{
			name: "user offline",
			setup: func(repo *mocks.MockPresenceRepository) {
				repo.EXPECT().Get(mock.Anything, key).Return(map[string]string{}, nil)
			},
			wantErr: domain.ErrUserOffline,
		},
		{
			name: "redis error",
			setup: func(repo *mocks.MockPresenceRepository) {
				repo.EXPECT().Get(mock.Anything, key).Return(nil, redisError)
			},
			wantErr: redisError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockPresenceRepository(t)
			svc := NewPresenceService(repo, slog.Default())
			tt.setup(repo)

			ans, err := svc.GetStatus(t.Context(), userID)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, ans)
			}
		})
	}
}

func Test_GetBulkStatus(t *testing.T) {
	userID := "test-getbulkstatus"
	key := presenceKey(userID)

	tests := []struct {
		name    string
		key     string
		setup   func(repo *mocks.MockPresenceRepository) []*redis.MapStringStringCmd
		wantErr error
	}{
		{
			name: "Successfully got bulk status",
			setup: func(repo *mocks.MockPresenceRepository) []*redis.MapStringStringCmd {
				cmd0 := redis.NewMapStringStringCmd(context.Background())
				cmd1 := redis.NewMapStringStringCmd(context.Background())
				cmd0.SetVal(map[string]string{"status": domain.StatusOnline})
				cmd1.SetVal(map[string]string{"status": domain.StatusOnline})
				redisMap := []*redis.MapStringStringCmd{cmd0, cmd1}

				repo.EXPECT().GetBulk(mock.Anything, []string{key, key}).Return(redisMap, nil)

				return redisMap
			},
		},
		{
			name: "redis error",
			setup: func(repo *mocks.MockPresenceRepository) []*redis.MapStringStringCmd {
				repo.EXPECT().GetBulk(mock.Anything, []string{key, key}).Return(nil, redisError)
				return nil
			},
			wantErr: redisError,
		},
		{
			name: "1 user offline",
			setup: func(repo *mocks.MockPresenceRepository) []*redis.MapStringStringCmd {
				cmd0 := redis.NewMapStringStringCmd(context.Background())
				cmd1 := redis.NewMapStringStringCmd(context.Background())
				cmd0.SetVal(map[string]string{"status": domain.StatusOnline})
				cmd1.SetVal(map[string]string{})
				redisMap := []*redis.MapStringStringCmd{cmd0, cmd1}

				repo.EXPECT().GetBulk(mock.Anything, []string{key, key}).Return(redisMap, nil)

				return redisMap
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := mocks.NewMockPresenceRepository(t)
			svc := NewPresenceService(repo, slog.Default())
			rmap := tt.setup(repo)
			_ = rmap
			ans, err := svc.GetBulkStatus(t.Context(), []string{userID, userID})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, ans)
			}
		})
	}
}
