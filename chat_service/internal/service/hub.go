package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"chat_service/internal/domain"
	"chat_service/internal/repository"
	loggerPkg "chat_service/pkg/logger"

	"github.com/gorilla/websocket"
)

type Hub interface {
	Connect(ctx context.Context, userID string, conn *websocket.Conn) error
	Disconnect(ctx context.Context, userID string) (User, error)
	InviteUser(ctx context.Context, inviterID, inviteeID, roomID string) error
	LeaveRoom(ctx context.Context, userID, roomID string) error
	Shutdown(ctx context.Context)
	CreateRoom(ctx context.Context, roomName, userID string) (*domain.Room, error)
	GetRoomClients(roomID string) []string
	GetRoomHistory(ctx context.Context, userID, roomID string, before time.Time) ([]*domain.Message, error)
	GetRoomsByUser(ctx context.Context, userID string) ([]*domain.Room, error)
}

type hub struct {
	rooms    map[string]Room
	users    map[string]User
	mu       sync.RWMutex
	ctx      context.Context
	userRepo repository.UserRepository
	roomRepo repository.RoomRepository
	msgRepo  repository.MessageRepository
	logger   *slog.Logger
	wg       sync.WaitGroup
}

func NewHub(
	ctx context.Context,
	userRepo repository.UserRepository,
	roomRepo repository.RoomRepository,
	msgRepo repository.MessageRepository,
	logger *slog.Logger,
) Hub {
	return &hub{
		rooms:    make(map[string]Room),
		users:    make(map[string]User),
		ctx:      ctx,
		userRepo: userRepo,
		roomRepo: roomRepo,
		msgRepo:  msgRepo,
		logger:   logger,
	}
}

func (h *hub) Connect(ctx context.Context, userID string, conn *websocket.Conn) error {
	const op = "chat.service.hub.connect"
	logger := h.logger.With(slog.String("op", op))

	user, err := h.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		logger.Error("failed to get user", loggerPkg.Err(err))
		return err
	}
	if user == nil {
		return domain.ErrUserNotFound
	}

	rooms, err := h.roomRepo.GetRoomsByUserID(ctx, userID)
	if err != nil {
		logger.Error("failed to get rooms", loggerPkg.Err(err))
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	newUser := NewUser(userID, user.Username, conn, h, logger)
	h.users[user.ID] = newUser

	for _, roomDTO := range rooms {
		room, exists := h.rooms[roomDTO.ID]
		if !exists {
			room = NewRoom(roomDTO.ID, h.msgRepo, h.logger)
			h.rooms[roomDTO.ID] = room
			go room.Run(ctx)
		}

		newUser.AddRoom(room)
		room.AddUser(newUser)
	}

	newUser.Listen(ctx, &h.wg)
	return nil
}

func (h *hub) Disconnect(ctx context.Context, userID string) (User, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	user, ok := h.users[userID]
	if !ok {
		return nil, domain.ErrUserNotFound
	}

	delete(h.users, user.ID())

	for _, room := range user.Rooms() {
		room.RemoveUser(user)

		if room.IsEmpty() {
			room.Stop()
			delete(h.rooms, room.ID())
		}
	}

	return user, nil
}

// TODO semantics not invites user but add to chat
func (h *hub) InviteUser(ctx context.Context, inviterID, inviteeID, roomID string) error {
	const op = "chat.service.hub.inviteuser"
	logger := h.logger.With(slog.String("op", op))

	isMember, err := h.isMember(ctx, inviterID, roomID)
	if err != nil {
		logger.Error("failed to check user", loggerPkg.Err(err))
		return err
	}
	if !isMember {
		return domain.ErrUserForbidden
	}

	if err := h.roomRepo.AddUser(ctx, inviteeID, roomID); err != nil {
		logger.Error("failed to add user", loggerPkg.Err(err))
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
		room = NewRoom(roomID, h.msgRepo, h.logger)
		h.rooms[roomID] = room
		go room.Run(h.ctx)
	}

	user.AddRoom(room)
	room.AddUser(user)

	return nil
}

func (h *hub) LeaveRoom(ctx context.Context, userID, roomID string) error {
	const op = "chat.service.hub.leaveroom"
	logger := h.logger.With(slog.String("op", op))

	isMember, err := h.isMember(ctx, userID, roomID)
	if err != nil {
		logger.Error("failed to check user", loggerPkg.Err(err))
		return err
	}
	if !isMember {
		return domain.ErrUserForbidden
	}

	if err := h.roomRepo.RemoveUser(ctx, userID, roomID); err != nil {
		logger.Error("failed to remove user", loggerPkg.Err(err))
		return err
	}

	isEmpty, err := h.roomRepo.IsEmpty(ctx, roomID)
	if err != nil {
		logger.Error("failed to IsEmpty", loggerPkg.Err(err))
		return err
	}

	if isEmpty {
		if _, err := h.roomRepo.DeleteRoom(ctx, roomID); err != nil {
			logger.Error("failed to delete room", loggerPkg.Err(err))
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
		room.RemoveUser(user)
		user.DeleteRoom(room)

		if room.IsEmpty() {
			room.Stop()
			delete(h.rooms, roomID)
		}
	}

	return nil
}

func (h *hub) Shutdown(ctx context.Context) {
	const op = "chat.service.hub.shutdown"
	logger := h.logger.With(slog.String("op", op))
	logger.Info("shutting down hub")

	h.mu.Lock()
	for _, user := range h.users {
		user.Stop()
	}
	h.mu.Unlock()

	h.wg.Wait()
}

func (h *hub) CreateRoom(ctx context.Context, roomName, userID string) (*domain.Room, error) {
	const op = "chat.service.hub.createroom"
	logger := h.logger.With(slog.String("op", op))

	roomDTO, err := h.roomRepo.CreateRoom(ctx, roomName, userID)
	if err != nil {
		logger.Error("failed to create room", "userID", userID, "roomName", roomName)
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	newRoom := NewRoom(roomDTO.ID, h.msgRepo, h.logger)
	h.rooms[roomDTO.ID] = newRoom
	go newRoom.Run(h.ctx)

	if user, online := h.users[userID]; online {
		user.AddRoom(newRoom)
		newRoom.AddUser(user)
	}

	return roomDTO, nil
}

func (h *hub) GetRoomClients(roomID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, exists := h.rooms[roomID]
	if !exists {
		return []string{}
	}

	return room.GetUsernames()
}

func (h *hub) GetRoomHistory(ctx context.Context, userID, roomID string, before time.Time) ([]*domain.Message, error) {
	const op = "chat.service.hub.getroomhistory"
	logger := h.logger.With(slog.String("op", op))

	isMember, err := h.isMember(ctx, userID, roomID)
	if err != nil {
		logger.Error("failed to check user", "userID", userID)
		return nil, err
	}
	if !isMember {
		return nil, domain.ErrUserForbidden
	}

	if before.After(time.Now()) {
		before = time.Time{}
	}
	var messages []*domain.Message
	if before.IsZero() {
		messages, err = h.msgRepo.GetMessages(ctx, roomID)
	} else {
		messages, err = h.msgRepo.GetMessagesBefore(ctx, roomID, before)
	}

	if err != nil {
		return nil, err
	}

	return messages, err
}

func (h *hub) GetRoomsByUser(ctx context.Context, userID string) ([]*domain.Room, error) {
	return h.roomRepo.GetRoomsByUserID(ctx, userID)
}

func (h *hub) isMember(ctx context.Context, userID, roomID string) (bool, error) {
	return h.roomRepo.IsMember(ctx, userID, roomID)
}
