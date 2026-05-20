package grpcserver

import (
	"context"
	"friends_service/internal/service"
	pb "friends_service/pkg/friends_pb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedFriendshipServer
	svc service.Friendship
}

func New(svc service.Friendship) *Server {
	return &Server{
		svc: svc,
	}
}

func (s *Server) IsFriend(ctx context.Context, req *pb.IsFriendRequest) (*pb.IsFriendResponse, error) {
	isFriend, err := s.svc.IsFriend(ctx, req.GetUserId(), req.GetFriendId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check friendship: %v", err)
	}
	return &pb.IsFriendResponse{Status: isFriend}, nil
}
