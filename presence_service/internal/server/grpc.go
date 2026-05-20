package server

import (
	"context"
	"errors"
	"presence_service/internal/domain"
	"presence_service/internal/service"
	pb "presence_service/pkg/pb"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type PresenceServer struct {
	pb.UnimplementedPresenceServer
	svc service.Presence
}

func NewPresenceServer(svc service.Presence) *PresenceServer {
	return &PresenceServer{
		svc: svc,
	}
}

func (srv *PresenceServer) MarkOnline(ctx context.Context, req *pb.MarkOnlineReq) (*pb.Status, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	err = srv.svc.MarkOnline(ctx, userID.String(), convertToMap(req.Metadata))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.Status{Status: pb.Code_OK}, nil
}

func (srv *PresenceServer) Heartbeat(ctx context.Context, req *pb.HeartbeatReq) (*pb.Status, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	err = srv.svc.Heartbeat(ctx, userID.String())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.Status{Status: pb.Code_OK}, nil
}

func (srv *PresenceServer) GetStatus(ctx context.Context, req *pb.GetStatusReq) (*pb.GetStatusRes, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}

	metadata, err := srv.svc.GetStatus(ctx, userID.String())
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrUserOffline):
			return &pb.GetStatusRes{
				Status: &pb.Status{
					Status:  pb.Code_NOT_FOUND,
					Message: "user offline",
				},
			}, nil
		default:
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &pb.GetStatusRes{
		Status: &pb.Status{
			Status: pb.Code_OK,
		},
		Metadata: convertToMapMetadata(metadata),
	}, nil
}

func (srv *PresenceServer) GetBulkStatus(ctx context.Context, req *pb.GetBulkStatusReq) (*pb.GetBulkStatusRes, error) {
	userIDs := req.UserIds
	userInfo := make([]*pb.UserInfo, len(userIDs))
	for i := range userIDs {
		userID, err := uuid.Parse(userIDs[i])
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user id")
		}

		metadata, err := srv.svc.GetStatus(ctx, userID.String())
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrUserOffline):
				userInfo = append(userInfo, &pb.UserInfo{Status: false})
				continue
			default:
				return nil, status.Error(codes.Internal, err.Error())
			}
		}

		userInfo = append(userInfo, &pb.UserInfo{
			Status:   true,
			Metadata: convertToMapMetadata(metadata),
		})
	}

	return &pb.GetBulkStatusRes{
		Status:   &pb.Status{Status: pb.Code_OK},
		UserInfo: userInfo,
	}, nil
}

func convertToMap(reqMD []*pb.MapMetadata) map[string]string {
	metadata := make(map[string]string, len(reqMD))
	for i := range reqMD {
		metadata[reqMD[i].Key] = reqMD[i].Value
	}

	return metadata
}

func convertToMapMetadata(metadata map[string]string) []*pb.MapMetadata {
	mm := make([]*pb.MapMetadata, len(metadata))

	i := 0
	for k, v := range metadata {
		var temp pb.MapMetadata
		temp.Key = k
		temp.Value = v

		mm[i] = &temp

		i++
	}

	return mm
}
