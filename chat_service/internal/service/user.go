package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"chat_service/internal/domain"
	loggerPkg "chat_service/pkg/logger"

	ws "github.com/gorilla/websocket"
)

const (
	MaxBufSize = 100
)

type User interface {
	AddRoom(room Room) error
	DeleteRoom(room Room) error
	Write(ctx context.Context, msg *domain.Message) error
	Listen(ctx context.Context, wg *sync.WaitGroup)
	ID() string
	Name() string
	Rooms() map[string]Room
	Stop()
}

type user struct {
	id          string
	name        string
	conn        *ws.Conn
	rooms       map[string]Room
	outgoingMsg chan *domain.Message
	hub         Hub
	logger      *slog.Logger
	doneCh      chan struct{}
	closeOnce   sync.Once
	mu          sync.RWMutex
}

func NewUser(
	id, name string,
	conn *ws.Conn,
	hub Hub,
	logger *slog.Logger,
) User {
	return &user{
		id:          id,
		name:        name,
		conn:        conn,
		rooms:       make(map[string]Room),
		outgoingMsg: make(chan *domain.Message, MaxBufSize),
		hub:         hub,
		logger:      logger,
		doneCh:      make(chan struct{}),
	}
}

func (u *user) AddRoom(room Room) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	_, ok := u.rooms[room.ID()]
	if ok {
		return domain.ErrRoomExists
	}
	u.rooms[room.ID()] = room
	return nil
}

func (u *user) DeleteRoom(room Room) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	_, ok := u.rooms[room.ID()]
	if !ok {
		return domain.ErrRoomNotFound
	}
	delete(u.rooms, room.ID())
	return nil
}

func (u *user) Write(ctx context.Context, msg *domain.Message) error {
	select {
	case u.outgoingMsg <- msg:
	default:
		u.closeOnce.Do(func() { close(u.doneCh) })
		return domain.ErrUserDisconnected
	}
	return nil
}

func (u *user) Listen(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(2)
	go func() { defer wg.Done(); u.listenWrite() }()
	go func() { defer wg.Done(); u.listenRead(ctx) }()
}

func (u *user) listenWrite() {
	const op = "chat.service.user.listenWrite"
	logger := u.logger.With("op", op)

	for {
		select {
		case <-u.doneCh:
			logger.Info("user done", slog.String("userID", u.id))
			return

		case msg := <-u.outgoingMsg:
			if err := u.conn.WriteJSON(msg); err != nil {
				u.closeOnce.Do(func() { close(u.doneCh) })
				logger.Info("write error user", slog.String("userID", u.id), loggerPkg.Err(err))
			}
		}
	}
}

func (u *user) listenRead(ctx context.Context) {
	defer func() {
		u.conn.Close()
		u.hub.Disconnect(ctx, u.id)
	}()

	const op = "chat.service.user.listenRead"
	logger := u.logger.With("op", op)

	for {
		select {
		case <-u.doneCh:
			logger.Info("user done", slog.String("userID", u.id))
			return

		case <-ctx.Done():
			logger.Info("user ctx done", slog.String("userID", u.id))
			return

		default:
			msg := domain.Message{}
			err := u.conn.ReadJSON(&msg)
			if err != nil {
				u.closeOnce.Do(func() { close(u.doneCh) })
				logger.Info("read error user", slog.String("userID", u.id), loggerPkg.Err(err))
				return
			} else {
				u.mu.RLock()
				room := u.rooms[msg.RoomID]
				u.mu.RUnlock()
				if room == nil {
					logger.Info("room doesnt exist")
					continue
				}
				msg.UserID = u.id
				msg.Username = u.name
				msg.Timestamp = time.Now().UTC()
				room.Broadcast(ctx, &msg)
			}
		}
	}
}

func (u *user) ID() string {
	return u.id
}

func (u *user) Name() string {
	return u.name
}

func (u *user) Rooms() map[string]Room {
	u.mu.RLock()
	defer u.mu.RUnlock()
	rooms := make(map[string]Room, len(u.rooms))
	for k, v := range u.rooms {
		rooms[k] = v
	}
	return rooms
}

func (u *user) Stop() {
	u.mu.Lock()
	u.conn.Close()
	u.mu.Unlock()
}
