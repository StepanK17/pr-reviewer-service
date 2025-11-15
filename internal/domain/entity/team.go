package entity

import "time"

type Team struct {
	TeamName  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TeamMember struct {
	UserID   string
	Username string
	IsActive bool
}

type TeamWithMembers struct {
	TeamName string
	Members  []TeamMember
}
