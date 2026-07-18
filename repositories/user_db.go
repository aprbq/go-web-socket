package repositories

import (
	"errors"

	"gorm.io/gorm"
)

type userRepositoryDB struct {
	db *gorm.DB
}

func NewUserRepositoryDB(db *gorm.DB) UserRepository {
	db.AutoMigrate(&user{})
	return userRepositoryDB{db: db}
}

func (r userRepositoryDB) CreateUser(username, passwordHash string) (*user, error) {
	u := &user{Username: username, PasswordHash: passwordHash}
	err := r.db.Create(u).Error
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r userRepositoryDB) SearchUsers(q string, excludeID uint, limit int) (users []user, err error) {
	err = r.db.Where("username ILIKE ? AND id <> ?", "%"+q+"%", excludeID).
		Order("username ASC").
		Limit(limit).
		Find(&users).Error
	return users, err
}

func (r userRepositoryDB) GetUserByUsername(username string) (*user, error) {
	u := new(user)
	err := r.db.Where("username = ?", username).First(u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}
