package repository

import (
	"context"

	"github.com/AlexOFF1/avito-backend-trainee-task-autumn/internal/models"
)

type PRRepository interface {
	CreateOrUpdateTeam(ctx context.Context, team models.Team) error
	GetTeam(ctx context.Context, teamName string) (*models.Team, error)
	SetUserActive(ctx context.Context, userID string, isActive bool) (*models.User, error)
	GetUserTeam(ctx context.Context, userID string) (string, error)
	RandomActiveMemberFromTeam(ctx context.Context, teamName, excludeID string) (string, error)

	GetUserReviewPRs(ctx context.Context, userID string) ([]models.PRShort, error)
	CreatePR(ctx context.Context, pr models.PullRequest) error
	GetPR(ctx context.Context, prID string) (*models.PullRequest, error)
	MergePR(ctx context.Context, prID string) (*models.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldUserID, newUserID string) (*models.PullRequest, error)
	GetActiveMembersExcluding(ctx context.Context, teamName string, excludeID string) ([]string, error)
}
