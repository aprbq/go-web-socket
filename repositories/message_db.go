package repositories

import (
	"slices"

	"gorm.io/gorm"
)

type messageRepositoryDB struct {
	db *gorm.DB
}

func NewMessageRepositoryDB(db *gorm.DB) MessageRepository {
	db.AutoMigrate(&message{})
	return messageRepositoryDB{db: db}
}

func (r messageRepositoryDB) SaveDirectMessage(senderID uint, senderName string, recipientID uint, recipientName, data string) error {
	m := &message{
		SenderID:      senderID,
		SenderName:    senderName,
		RecipientID:   recipientID,
		RecipientName: recipientName,
		Data:          data,
	}
	return r.db.Create(m).Error
}

// GetDirectMessages returns the last `limit` messages between the two users, oldest first.
func (r messageRepositoryDB) GetDirectMessages(userA, userB uint, limit int) (msgs []message, err error) {
	err = r.db.Where(
		"(sender_id = ? AND recipient_id = ?) OR (sender_id = ? AND recipient_id = ?)",
		userA, userB, userB, userA,
	).
		Order("id DESC").
		Limit(limit).
		Find(&msgs).Error
	if err != nil {
		return nil, err
	}

	slices.Reverse(msgs)
	return msgs, nil
}

func (r messageRepositoryDB) GetLatestDirectMessages(userID uint) ([]message, error) {
	all := []message{}
	err := r.db.Where("sender_id = ? OR recipient_id = ?", userID, userID).
		Order("id DESC").
		Find(&all).Error
	if err != nil {
		return nil, err
	}

	// keep only the newest message per conversation partner
	latest := []message{}
	seen := map[uint]bool{}
	for _, m := range all {
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
