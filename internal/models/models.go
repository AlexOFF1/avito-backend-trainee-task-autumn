package models

import "time"

type Team struct {
	Name    string
	Members []Member
}

type Member struct {
	ID       string
	Username string
	IsActive bool
}

type User struct {
	ID       string
	Username string
	TeamName string
	IsActive bool
}

type PullRequest struct {
	ID                string
	Name              string
	AuthorID          string
	Status            string
	AssignedReviewers []string
	CreatedAt         *time.Time
	MergedAt          *time.Time
}

type PRShort struct {
	ID       string
	Name     string
	AuthorID string
	Status   string
}
