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

const hubSvcName = "chat.service.hub"

type Hub interface {
	Connect(ctx context.Context, userID string, conn *websocket.Conn) error
	Disconnect(userID string) (User, error)

	InviteUser(ctx context.Context, userID, inviteeID, roomID string) error
	LeaveRoom(ctx context.Context, userID, roomID string) error

	CreateRoom(ctx context.Context, roomName, userID string) (*domain.Room, error)
	CreateDM(ctx context.Context, userID, inviteeID string) (*domain.Room, error)
	JoinRoom(ctx context.Context, userID, roomID string) (Room, error)

	GetRoomClients(ctx context.Context, roomID string) ([]*domain.User, error)
	GetRoomHistory(ctx context.Context, roomID, userID string, before time.Time) ([]*domain.Message, error)
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
	l := h.loggerWith(".connect")

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
	dms, err := h.roomRepo.GetDMsByUserID(ctx, userID)
	if err != nil {
		l.Error("failed to get dms", sl.Err(err))
		return err
	}

	rooms = append(rooms, dms...)

	h.mu.Lock()
	defer h.mu.Unlock()

	newUser := NewUser(userID, user.Username, conn, h, l)
	h.users[user.ID] = newUser

	for _, room := range rooms {
		h.connectToRoom(ctx, newUser, room.ID, l)
	}

	newUser.Listen(ctx, &h.wg)
	return nil
}

func (h *hub) Disconnect(userID string) (User, error) {
	l := h.loggerWith(".disconnect")

	h.mu.Lock()
	defer h.mu.Unlock()

	user, ok := h.users[userID]
	if !ok {
		return nil, domain.ErrUserNotFound
	}

	delete(h.users, user.ID())

	for _, room := range user.Rooms() {
		if err := room.RemoveUser(user); err != nil {
			l.Error("failed to remove user", sl.Err(err))
		}

		if room.IsEmpty() {
			room.Stop()
			delete(h.rooms, room.ID())
		}
	}

	return user, nil
}

// TODO semantics not invites user but add to chat.
func (h *hub) InviteUser(ctx context.Context, userID, inviteeID, roomID string) error {
	l := h.loggerWith(".inviteuser")

	err := h.isMemberValidation(ctx, userID, roomID)
	if err != nil {
		l.Error("failed to validate membership", sl.Err(err))
		return err
	}

	isFriend, err := h.friendsClient.IsFriend(ctx, userID, inviteeID)
	if err != nil {
		l.Error("failed to check friendship", sl.Err(err))
		return err
	}
	if !isFriend {
		return domain.ErrUserForbidden
	}
	roomType, err := h.roomRepo.GetRoomType(ctx, roomID)
	if err != nil {
		l.Error("failed to get room type", sl.Err(err))
		return err
	}
	if roomType == "direct" {
		l.Error("invalid room type")
		return domain.ErrRoomNotFound
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

	h.connectToRoom(h.ctx, user, roomID, l) //nolint:contextcheck // uses server cancel context

	return nil
}

func (h *hub) LeaveRoom(ctx context.Context, userID, roomID string) error {
	l := h.loggerWith(".leaveroom")

	err := h.isMemberValidation(ctx, userID, roomID)
	if err != nil {
		l.Error("failed to validate membership", sl.Err(err))
		return err
	}

	if err = h.roomRepo.RemoveUser(ctx, userID, roomID); err != nil {
		l.Error("failed to remove user", sl.Err(err))
		return err
	}

	isEmpty, err := h.roomRepo.IsEmpty(ctx, roomID)
	if err != nil {
		l.Error("failed to check emptyness", sl.Err(err))
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
			l.Error("failed to remove user", sl.Err(err))
		}
		if err := user.DeleteRoom(room); err != nil {
			l.Error("failed to delete room", sl.Err(err))
		}

		if room.IsEmpty() {
			room.Stop()
			delete(h.rooms, roomID)
		}
	}

	return nil
}

func (h *hub) Shutdown() {
	h.mu.Lock()
	for _, user := range h.users {
		user.Stop()
	}
	h.mu.Unlock()

	h.wg.Wait()
}

func (h *hub) CreateRoom(ctx context.Context, roomName, userID string) (*domain.Room, error) {
	l := h.loggerWith(".createroom")

	room, err := h.roomRepo.CreateRoom(ctx, roomName, userID)
	if err != nil {
		l.Error("failed to create room", slog.String("userID", userID), sl.Err(err))
		return nil, err
	}

	h.runRoom(l, room.ID, userID)

	return room, nil
}

func (h *hub) CreateDM(ctx context.Context, userID, inviteeID string) (*domain.Room, error) {
	l := h.loggerWith(".createdm")

	room, err := h.roomRepo.CreateDM(ctx, userID, inviteeID)
	if err != nil {
		l.Error("failed to create room", "userID", userID, sl.Err(err))
		return nil, err
	}

	h.runRoom(l, room.ID, userID)

	invitee, online := h.users[inviteeID]
	if !online {
		return room, nil
	}
	h.connectToRoom(h.ctx, invitee, room.ID, l) //nolint:contextcheck // uses server cancel context

	return room, nil
}

func (h *hub) GetRoomClients(ctx context.Context, roomID string) ([]*domain.User, error) {
	_, exists := h.rooms[roomID]
	if !exists {
		return nil, nil
	}

	return h.roomRepo.GetUsersByRoomID(ctx, roomID)
}

func (h *hub) GetRoomHistory(ctx context.Context, roomID, userID string, before time.Time) ([]*domain.Message, error) {
	l := h.loggerWith(".getroomhistory")

	err := h.isMemberValidation(ctx, userID, roomID)
	if err != nil {
		l.Error("failed to validate membership", sl.Err(err))
		return nil, err
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
	l := h.loggerWith(".getroomsbyuser")

	rooms, err := h.roomRepo.GetRoomsByUserID(ctx, userID)
	if err != nil {
		l.Error("failed to get rooms", sl.Err(err))
		return nil, err
	}

	dms, err := h.roomRepo.GetDMsByUserID(ctx, userID)
	if err != nil {
		l.Error("failed to get dms", sl.Err(err))
		return nil, err
	}

	rooms = append(rooms, dms...)
	return rooms, nil
}

func (h *hub) runRoom(l *slog.Logger, roomID, userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	newRoom := NewRoom(roomID, h.msgRepo, h.ps, h.logger)
	h.rooms[roomID] = newRoom
	go newRoom.Run(h.ctx)

	if user, online := h.users[userID]; online {
		if err := user.AddRoom(newRoom); err != nil {
			l.Error("failed to add room", sl.Err(err))
		}
		if err := newRoom.AddUser(user); err != nil {
			l.Error("failed to add user", sl.Err(err))
		}
	}
}

func (h *hub) JoinRoom(ctx context.Context, userID, roomID string) (Room, error) {
	isMember, err := h.roomRepo.IsMember(ctx, userID, roomID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, domain.ErrUserForbidden
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	user, online := h.users[userID]
	if !online {
		return nil, domain.ErrUserNotFound
	}

	h.connectToRoom(ctx, user, roomID, h.logger.With("op", hubSvcName+".joinroom"))
	return h.rooms[roomID], nil
}

func (h *hub) isMemberValidation(ctx context.Context, inviterID, roomID string) error {
	isMember, err := h.roomRepo.IsMember(ctx, inviterID, roomID)
	if err != nil {
		return err
	}
	if !isMember {
		return domain.ErrUserForbidden
	}
	return nil
}

func (h *hub) loggerWith(fnName string) *slog.Logger {
	return h.logger.With("op", hubSvcName+fnName)
}

func (h *hub) connectToRoom(ctx context.Context, user User, roomID string, l *slog.Logger) {
	room, exists := h.rooms[roomID]
	if !exists {
		room = NewRoom(roomID, h.msgRepo, h.ps, h.logger)
		h.rooms[roomID] = room
		go room.Run(ctx)
	}

	if err := user.AddRoom(room); err != nil {
		l.Error("failed to add room", sl.Err(err))
	}
	if err := room.AddUser(user); err != nil {
		l.Error("failed to add user", sl.Err(err))
	}
}
