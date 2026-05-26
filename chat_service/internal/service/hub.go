package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"chat_service/internal/clients"
	"chat_service/internal/domain"
	"chat_service/internal/pubsub"
	"chat_service/internal/repository"
	sl "chat_service/pkg/service_logger"

	"github.com/gorilla/websocket"
)

type Hub interface {
	Connect(ctx context.Context, userID string, conn *websocket.Conn) error
	Disconnect(userID string) (User, error)

	InviteUser(ctx context.Context, inviterID, inviteeID, roomID string) error
	LeaveRoom(ctx context.Context, userID, roomID string) error

	CreateRoom(ctx context.Context, roomName, userID string) (*domain.Room, error)

	GetRoomClients(ctx context.Context, roomID string) ([]*domain.User, error)
	GetRoomHistory(ctx context.Context, userID, roomID string, before time.Time) ([]*domain.Message, error)
	GetRoomsByUser(ctx context.Context, userID string) ([]*domain.Room, error)

	Shutdown()
}

type hub struct {
	rooms         map[string]Room
	users         map[string]User
	mu            sync.RWMutex
	ctx           context.Context
	userRepo      repository.UserRepository
	roomRepo      repository.RoomRepository
	msgRepo       repository.MessageRepository
	logger        *slog.Logger
	wg            sync.WaitGroup
	friendsClient clients.FriendshipClient
	ps            pubsub.PubSub
}

func NewHub(
	ctx context.Context,
	userRepo repository.UserRepository,
	roomRepo repository.RoomRepository,
	msgRepo repository.MessageRepository,
	logger *slog.Logger,
	friendsClient clients.FriendshipClient,
	ps pubsub.PubSub,
) Hub {
	return &hub{
		rooms:         make(map[string]Room),
		users:         make(map[string]User),
		ctx:           ctx,
		userRepo:      userRepo,
		roomRepo:      roomRepo,
		msgRepo:       msgRepo,
		logger:        logger,
		friendsClient: friendsClient,
		ps:            ps,
	}
}

func (h *hub) Connect(ctx context.Context, userID string, conn *websocket.Conn) error {
	const op = "chat.service.hub.connect"
	l := h.logger.With(slog.String("op", op))

	user, err := h.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		l.Error("failed to get user", sl.Err(err))
		return err
	}
	if user == nil {
		return domain.ErrUserNotFound
	}

	rooms, err := h.roomRepo.GetRoomsByUserID(ctx, userID)
	if err != nil {
		l.Error("failed to get rooms", sl.Err(err))
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	newUser := NewUser(userID, user.Username, conn, h, l)
	h.users[user.ID] = newUser

	for _, roomDTO := range rooms {
		room, exists := h.rooms[roomDTO.ID]
		if !exists {
			room = NewRoom(roomDTO.ID, h.msgRepo, h.ps, h.logger)
			h.rooms[roomDTO.ID] = room
			go room.Run(ctx)
		}

		if err := newUser.AddRoom(room); err != nil {
			l.Error("error adding room", sl.Err(err))
		}
		if err := room.AddUser(newUser); err != nil {
			l.Error("error adding user", sl.Err(err))
		}
	}

	newUser.Listen(ctx, &h.wg)
	return nil
}

func (h *hub) Disconnect(userID string) (User, error) {
	const op = "chat.service.hub.disconnect"
	l := h.logger.With(slog.String("op", op))

	h.mu.Lock()
	defer h.mu.Unlock()

	user, ok := h.users[userID]
	if !ok {
		return nil, domain.ErrUserNotFound
	}

	delete(h.users, user.ID())

	for _, room := range user.Rooms() {
		if err := room.RemoveUser(user); err != nil {
			l.Error("error removing user", sl.Err(err))
		}

		if room.IsEmpty() {
			room.Stop()
			delete(h.rooms, room.ID())
		}
	}

	return user, nil
}

// TODO semantics not invites user but add to chat.
func (h *hub) InviteUser(ctx context.Context, inviterID, inviteeID, roomID string) error {
	const op = "chat.service.hub.inviteuser"
	l := h.logger.With(slog.String("op", op))

	isMember, err := h.roomRepo.IsMember(ctx, inviterID, roomID)
	if err != nil {
		l.Error("failed to check user", sl.Err(err))
		return err
	}
	if !isMember {
		return domain.ErrUserForbidden
	}

	isFriend, err := h.friendsClient.IsFriend(ctx, inviterID, inviteeID)
	if err != nil {
		l.Error("failed to check friendship", sl.Err(err))
		return err
	}
	if !isFriend {
		return domain.ErrUserForbidden
	}

	if err := h.roomRepo.AddUser(ctx, inviteeID, roomID); err != nil {
		l.Error("failed to add user", sl.Err(err))
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	user, online := h.users[inviteeID]
	if !online {
		return nil
	}

	room, exists := h.rooms[roomID]
	if !exists {
		room = NewRoom(roomID, h.msgRepo, h.ps, h.logger)
		h.rooms[roomID] = room
		//nolint:contextcheck // using connection context
		go room.Run(h.ctx)
	}

	if err := user.AddRoom(room); err != nil {
		l.Error("error adding room", sl.Err(err))
	}
	if err := room.AddUser(user); err != nil {
		l.Error("error adding user", sl.Err(err))
	}

	return nil
}

func (h *hub) LeaveRoom(ctx context.Context, userID, roomID string) error {
	const op = "chat.service.hub.leaveroom"
	l := h.logger.With(slog.String("op", op))

	isMember, err := h.roomRepo.IsMember(ctx, userID, roomID)
	if err != nil {
		l.Error("failed to check user", sl.Err(err))
		return err
	}
	if !isMember {
		return domain.ErrUserForbidden
	}

	if err = h.roomRepo.RemoveUser(ctx, userID, roomID); err != nil {
		l.Error("failed to remove user", sl.Err(err))
		return err
	}

	isEmpty, err := h.roomRepo.IsEmpty(ctx, roomID)
	if err != nil {
		l.Error("failed to IsEmpty", sl.Err(err))
		return err
	}

	if isEmpty {
		if _, err := h.roomRepo.DeleteRoom(ctx, roomID); err != nil {
			l.Error("failed to delete room", sl.Err(err))
			return err
		}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	user, online := h.users[userID]
	if !online {
		return nil
	}

	if room, exists := h.rooms[roomID]; exists {
		if err := room.RemoveUser(user); err != nil {
			l.Error("error removing user", sl.Err(err))
		}
		if err := user.DeleteRoom(room); err != nil {
			l.Error("error deleting room", sl.Err(err))
		}

		if room.IsEmpty() {
			room.Stop()
			delete(h.rooms, roomID)
		}
	}

	return nil
}

func (h *hub) Shutdown() {
	const op = "chat.service.hub.shutdown"
	l := h.logger.With(slog.String("op", op))
	l.Info("shutting down hub")

	h.mu.Lock()
	for _, user := range h.users {
		user.Stop()
	}
	h.mu.Unlock()

	h.wg.Wait()
}

func (h *hub) CreateRoom(ctx context.Context, roomName, userID string) (*domain.Room, error) {
	const op = "chat.service.hub.createroom"
	l := h.logger.With(slog.String("op", op))

	roomDTO, err := h.roomRepo.CreateRoom(ctx, roomName, userID)
	if err != nil {
		l.Error("failed to create room", "userID", userID, "roomName", roomName)
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	newRoom := NewRoom(roomDTO.ID, h.msgRepo, h.ps, h.logger)
	h.rooms[roomDTO.ID] = newRoom
	//nolint:contextcheck // using connection context
	go newRoom.Run(h.ctx)

	if user, online := h.users[userID]; online {
		if err := user.AddRoom(newRoom); err != nil {
			l.Error("error adding room", sl.Err(err))
		}
		if err := newRoom.AddUser(user); err != nil {
			l.Error("error adding user", sl.Err(err))
		}
	}

	return roomDTO, nil
}

func (h *hub) GetRoomClients(ctx context.Context, roomID string) ([]*domain.User, error) {
	_, exists := h.rooms[roomID]
	if !exists {
		return nil, nil
	}

	return h.roomRepo.GetUsersByRoomID(ctx, roomID)
}

func (h *hub) GetRoomHistory(ctx context.Context, userID, roomID string, before time.Time) ([]*domain.Message, error) {
	const op = "chat.service.hub.getroomhistory"
	l := h.logger.With(slog.String("op", op))

	isMember, err := h.roomRepo.IsMember(ctx, userID, roomID)
	if err != nil {
		l.Error("failed to check user", "userID", userID)
		return nil, err
	}
	if !isMember {
		return nil, domain.ErrUserForbidden
	}

	if before.IsZero() {
		before = time.Now().Add(time.Second)
	}

	messages, err := h.msgRepo.GetMessagesBefore(ctx, roomID, before)

	if err != nil {
		return nil, err
	}

	return messages, err
}

func (h *hub) GetRoomsByUser(ctx context.Context, userID string) ([]*domain.Room, error) {
	return h.roomRepo.GetRoomsByUserID(ctx, userID)
}
