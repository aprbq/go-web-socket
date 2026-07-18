package repositories

import (
	"sync"
	"time"
)

// message is the DB schema model — it never leaves this layer.
// RecipientID > 0 means a direct message; otherwise RoomID names the room
// and RoomID = "" means it was a broadcast message.
type message struct {
	ID            uint `gorm:"primaryKey"`
	SenderID      uint `gorm:"index"`
	SenderName    string
	RecipientID   uint `gorm:"index"`
	RecipientName string
	RoomID        string `gorm:"index"`
	Data          string
	CreatedAt     time.Time
}

type MessageRepository interface {
	SaveMessage(senderID uint, senderName, roomID, data string) error
	SaveDirectMessage(senderID uint, senderName string, recipientID uint, recipientName, data string) error
	GetRoomMessages(roomID string, limit int) ([]message, error)
	GetDirectMessages(userA, userB uint, limit int) ([]message, error)
	// GetLatestDirectMessages returns the newest message per conversation
	// partner of userID, most recent conversation first.
	GetLatestDirectMessages(userID uint) ([]message, error)
}

// messageRepositoryMock is an in-memory implementation for tests.
type messageRepositoryMock struct {
	mu       sync.Mutex
	messages []message
}

func NewMessageRepositoryMock() MessageRepository {
	return &messageRepositoryMock{}
}

func (r *messageRepositoryMock) SaveMessage(senderID uint, senderName, roomID, data string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.messages = append(r.messages, message{
		ID:         uint(len(r.messages) + 1),
		SenderID:   senderID,
		SenderName: senderName,
		RoomID:     roomID,
		Data:       data,
		CreatedAt:  time.Now(),
	})
	return nil
}

func (r *messageRepositoryMock) SaveDirectMessage(senderID uint, senderName string, recipientID uint, recipientName, data string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.messages = append(r.messages, message{
		ID:            uint(len(r.messages) + 1),
		SenderID:      senderID,
		SenderName:    senderName,
		RecipientID:   recipientID,
		RecipientName: recipientName,
		Data:          data,
		CreatedAt:     time.Now(),
	})
	return nil
}

func (r *messageRepositoryMock) GetRoomMessages(roomID string, limit int) ([]message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	msgs := []message{}
	for _, m := range r.messages {
		if m.RecipientID == 0 && m.RoomID == roomID {
			msgs = append(msgs, m)
		}
	}
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	return msgs, nil
}

func (r *messageRepositoryMock) GetDirectMessages(userA, userB uint, limit int) ([]message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	msgs := []message{}
	for _, m := range r.messages {
		if (m.SenderID == userA && m.RecipientID == userB) ||
			(m.SenderID == userB && m.RecipientID == userA) {
			msgs = append(msgs, m)
		}
	}
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	return msgs, nil
}

func (r *messageRepositoryMock) GetLatestDirectMessages(userID uint) ([]message, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	latest := []message{}
	seen := map[uint]bool{}
	for i := len(r.messages) - 1; i >= 0; i-- {
		m := r.messages[i]
		if m.RecipientID == 0 || (m.SenderID != userID && m.RecipientID != userID) {
			continue
		}
		peer := m.SenderID
		if m.SenderID == userID {
			peer = m.RecipientID
		}
		if !seen[peer] {
			seen[peer] = true
			latest = append(latest, m)
		}
	}
	return latest, nil
}
