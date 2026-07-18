package repositories

import (
	"sync"
	"time"
)

// message is the DB schema model for a direct message — it never leaves this layer.
type message struct {
	ID            uint `gorm:"primaryKey"`
	SenderID      uint `gorm:"index"`
	SenderName    string
	RecipientID   uint `gorm:"index"`
	RecipientName string
	Data          string
	CreatedAt     time.Time
}

type MessageRepository interface {
	SaveDirectMessage(senderID uint, senderName string, recipientID uint, recipientName, data string) error
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
		if m.SenderID != userID && m.RecipientID != userID {
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
