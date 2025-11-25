package usecase

import (
	"context"

	"github.com/AlexOFF1/avito-backend-trainee-task-autumn/internal/models"
)

type PRService interface {
	CreateTeam(ctx context.Context, team models.Team) (*models.Team, error)
	GetTeam(ctx context.Context, teamName string) (*models.Team, error)
	SetUserActive(ctx context.Context, userID string, isActive bool) (*models.User, error)

	CreatePR(ctx context.Context, pr models.PullRequest) (*models.PullRequest, error)
	MergePR(ctx context.Context, prID string) (*models.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldUserID string) (*models.PullRequest, string, error)
	GetUserReviews(ctx context.Context, userID string) ([]models.PRShort, error)
}
