package server

import (
	"errors"
	"presence_service/internal/domain"
	"presence_service/internal/mocks"
	"presence_service/pkg/pb"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var svcError = errors.New("service error")

func Test_MarkOnline(t *testing.T) {
	userID := uuid.NewString()

	tests := []struct {
		name       string
		userID     string
		setup      func(svc *mocks.MockPresence)
		wantStatus pb.Code
		wantErr    bool
	}{
		{
			name:   "Success",
			userID: userID,
			setup: func(svc *mocks.MockPresence) {
				svc.EXPECT().MarkOnline(mock.Anything, userID, map[string]string{}).Return(nil)
			},
			wantStatus: pb.Code_OK,
		},
		{
			name:       "Error parsing user id",
			userID:     "some-random-id",
			setup:      func(svc *mocks.MockPresence) {},
			wantErr:    true,
			wantStatus: pb.Code_INVALID_ARGUMENT,
		},
		{
			name:   "Service error",
			userID: userID,
			setup: func(svc *mocks.MockPresence) {
				svc.EXPECT().MarkOnline(mock.Anything, userID, map[string]string{}).Return(svcError)
			},
			wantErr:    true,
			wantStatus: pb.Code_INTERNAL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewMockPresence(t)
			srv := NewPresenceServer(svc)
			tt.setup(svc)

			stt, err := srv.MarkOnline(t.Context(), &pb.MarkOnlineReq{UserId: tt.userID, Metadata: []*pb.MapMetadata{}})

			if tt.wantErr {
				statusErr, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, codes.Code(tt.wantStatus), statusErr.Code())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, stt.Status)
			}
		})
	}
}

func Test_Heartbeat(t *testing.T) {
	userID := uuid.NewString()

	tests := []struct {
		name       string
		userID     string
		setup      func(svc *mocks.MockPresence)
		wantStatus pb.Code
		wantErr    bool
	}{
		{
			name:   "Success",
			userID: userID,
			setup: func(svc *mocks.MockPresence) {
				svc.EXPECT().Heartbeat(mock.Anything, userID).Return(nil)
			},
			wantStatus: pb.Code_OK,
		},
		{
			name:       "Error parsing user id",
			userID:     "some-random-id",
			setup:      func(svc *mocks.MockPresence) {},
			wantErr:    true,
			wantStatus: pb.Code_INVALID_ARGUMENT,
		},
		{
			name:   "Service error",
			userID: userID,
			setup: func(svc *mocks.MockPresence) {
				svc.EXPECT().Heartbeat(mock.Anything, userID).Return(svcError)
			},
			wantErr:    true,
			wantStatus: pb.Code_INTERNAL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewMockPresence(t)
			srv := NewPresenceServer(svc)
			tt.setup(svc)

			stt, err := srv.Heartbeat(t.Context(), &pb.HeartbeatReq{UserId: tt.userID})

			if tt.wantErr {
				statusErr, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, codes.Code(tt.wantStatus), statusErr.Code())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, stt.Status)
			}
		})
	}
}

func Test_GetStatus(t *testing.T) {
	userID := uuid.NewString()
	metadata := map[string]string{"status": domain.StatusOnline}

	tests := []struct {
		name       string
		userID     string
		setup      func(svc *mocks.MockPresence)
		wantStatus pb.Code
		wantErr    bool
	}{
		{
			name:   "Success",
			userID: userID,
			setup: func(svc *mocks.MockPresence) {
				svc.EXPECT().GetStatus(mock.Anything, userID).Return(metadata, nil)
			},
			wantStatus: pb.Code_OK,
		},
		{
			name:       "Error parsing user id",
			userID:     "some-random-id",
			setup:      func(svc *mocks.MockPresence) {},
			wantErr:    true,
			wantStatus: pb.Code_INVALID_ARGUMENT,
		},
		{
			name:   "Service error",
			userID: userID,
			setup: func(svc *mocks.MockPresence) {
				svc.EXPECT().GetStatus(mock.Anything, userID).Return(nil, svcError)
			},
			wantErr:    true,
			wantStatus: pb.Code_INTERNAL,
		},
		{
			name:   "User offline",
			userID: userID,
			setup: func(svc *mocks.MockPresence) {
				svc.EXPECT().GetStatus(mock.Anything, userID).Return(nil, domain.ErrUserOffline)
			},
			wantStatus: pb.Code_NOT_FOUND,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewMockPresence(t)
			srv := NewPresenceServer(svc)
			tt.setup(svc)

			stt, err := srv.GetStatus(t.Context(), &pb.GetStatusReq{UserId: tt.userID})

			if tt.wantErr {
				statusErr, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, codes.Code(tt.wantStatus), statusErr.Code())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, stt.Status.Status)
			}
		})
	}
}

func Test_GetBulkStatus(t *testing.T) {
	userID1 := uuid.NewString()
	userID2 := uuid.NewString()

	tests := []struct {
		name       string
		userIDs    []string
		setup      func(svc *mocks.MockPresence)
		wantStatus pb.Code
		wantErr    bool
	}{
		{
			name:    "All users online",
			userIDs: []string{userID1, userID2},
			setup: func(svc *mocks.MockPresence) {
				svc.EXPECT().GetStatus(mock.Anything, userID1).Return(map[string]string{}, nil)
				svc.EXPECT().GetStatus(mock.Anything, userID2).Return(map[string]string{}, nil)
			},
			wantStatus: pb.Code_OK,
		},
		{
			name:    "One user offline",
			userIDs: []string{userID1, userID2},
			setup: func(svc *mocks.MockPresence) {
				svc.EXPECT().GetStatus(mock.Anything, userID1).Return(map[string]string{}, nil)
				svc.EXPECT().GetStatus(mock.Anything, userID2).Return(nil, domain.ErrUserOffline)
			},
			wantStatus: pb.Code_OK,
		},
		{
			name:       "Invalid user id",
			userIDs:    []string{"some-random-id"},
			setup:      func(svc *mocks.MockPresence) {},
			wantErr:    true,
			wantStatus: pb.Code_INVALID_ARGUMENT,
		},
		{
			name:    "Service error",
			userIDs: []string{userID1},
			setup: func(svc *mocks.MockPresence) {
				svc.EXPECT().GetStatus(mock.Anything, userID1).Return(nil, svcError)
			},
			wantErr:    true,
			wantStatus: pb.Code_INTERNAL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewMockPresence(t)
			srv := NewPresenceServer(svc)
			tt.setup(svc)

			res, err := srv.GetBulkStatus(t.Context(), &pb.GetBulkStatusReq{UserIds: tt.userIDs})

			if tt.wantErr {
				statusErr, ok := status.FromError(err)
				assert.True(t, ok)
				assert.Equal(t, codes.Code(tt.wantStatus), statusErr.Code())
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStatus, res.Status.Status)
			}
		})
	}
}
