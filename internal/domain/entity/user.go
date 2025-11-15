package entity

import "time"

type User struct {
	UserID    string
	Username  string
	TeamName  string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
