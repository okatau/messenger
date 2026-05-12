package grpc

import (
	"context"
	"friends_service/internal/service"
	pb "friends_service/pkg/friendspb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCServer struct {
	pb.UnimplementedFriendshipServer
	Svc service.Friendship
}

func (s *GRPCServer) IsFriend(ctx context.Context, req *pb.IsFriendRequest) (*pb.IsFriendResponse, error) {
	isFriend, err := s.Svc.IsFriend(ctx, req.GetUserId(), req.GetFriendId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check friendship: %v", err)
	}
	return &pb.IsFriendResponse{Status: isFriend}, nil
}
