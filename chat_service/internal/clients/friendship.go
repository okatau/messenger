package clients

import (
	pb "chat_service/pkg/friends_pb"
	"context"

	"google.golang.org/grpc"
)

type FriendshipClient interface {
	IsFriend(ctx context.Context, userID, friendID string) (bool, error)
}

type friendshipClient struct {
	client pb.FriendshipClient
}

func NewFriendshipClient(conn *grpc.ClientConn) FriendshipClient {
	return &friendshipClient{client: pb.NewFriendshipClient(conn)}
}

func (c *friendshipClient) IsFriend(ctx context.Context, userID, friendID string) (bool, error) {
	resp, err := c.client.IsFriend(ctx, &pb.IsFriendRequest{
		UserId:   userID,
		FriendId: friendID,
	})
	if err != nil {
		return false, err
	}
	return resp.Status, err
}
