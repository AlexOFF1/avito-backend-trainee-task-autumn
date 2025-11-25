package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"

	"github.com/AlexOFF1/avito-backend-trainee-task-autumn/internal/models"
	"github.com/AlexOFF1/avito-backend-trainee-task-autumn/internal/repository"
)

type prService struct {
	repo   repository.PRRepository
	logger *slog.Logger
}

func NewPRService(repo repository.PRRepository, logger *slog.Logger) PRService {
	return &prService{repo: repo, logger: logger}
}

func (s *prService) CreateTeam(ctx context.Context, team models.Team) (*models.Team, error) {
	if team.Name == "" {
		s.logger.Warn("invalid team name")
		return nil, errors.New("team name required")
	}
	if len(team.Members) == 0 {
		s.logger.Warn("no members")
		return nil, errors.New("members required")
	}
	for _, m := range team.Members {
		if m.ID == "" || m.Username == "" {
			s.logger.Warn("invalid member", "id", m.ID)
			return nil, errors.New("invalid member data")
		}
	}

	_, err := s.repo.GetTeam(ctx, team.Name)
	if err == nil {
		s.logger.Warn("team already exists", "name", team.Name)
		return nil, errors.New("TEAM_EXISTS")
	}
	if !errors.Is(err, repository.ErrTeamNotFound) {
		s.logger.Error("failed to check team existence", "err", err)
		return nil, fmt.Errorf("check team exists: %w", err)
	}

	if err := s.repo.CreateOrUpdateTeam(ctx, team); err != nil {
		s.logger.Error("create team failed", "err", err)
		return nil, fmt.Errorf("create team: %w", err)
	}

	newTeam, err := s.repo.GetTeam(ctx, team.Name)
	if err != nil {
		s.logger.Error("get created team failed", "err", err)
		return nil, fmt.Errorf("get team: %w", err)
	}
	return newTeam, nil
}

func (s *prService) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	if teamName == "" {
		s.logger.Warn("invalid team name")
		return nil, errors.New("team name required")
	}
	team, err := s.repo.GetTeam(ctx, teamName)
	if err != nil {
		s.logger.Error("get team failed", "err", err)
		return nil, fmt.Errorf("get team: %w", err)
	}
	return team, nil
}

func (s *prService) SetUserActive(ctx context.Context, userID string, isActive bool) (*models.User, error) {
	if userID == "" {
		s.logger.Warn("invalid user id")
		return nil, errors.New("user id required")
	}
	user, err := s.repo.SetUserActive(ctx, userID, isActive)
	if err != nil {
		s.logger.Error("set active failed", "err", err)
		return nil, fmt.Errorf("set active: %w", err)
	}
	return user, nil
}

func (s *prService) CreatePR(ctx context.Context, pr models.PullRequest) (*models.PullRequest, error) {
	if pr.ID == "" || pr.Name == "" || pr.AuthorID == "" {
		s.logger.Warn("invalid pr data")
		return nil, errors.New("pr fields required")
	}

	teamName, err := s.repo.GetUserTeam(ctx, pr.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("get author team: %w", err)
	}

	members, err := s.repo.GetActiveMembersExcluding(ctx, teamName, pr.AuthorID)
	if err != nil {
		return nil, fmt.Errorf("get active members: %w", err)
	}

	rand.Shuffle(len(members), func(i, j int) { members[i], members[j] = members[j], members[i] })
	if len(members) > 2 {
		pr.AssignedReviewers = members[:2]
	} else {
		pr.AssignedReviewers = members
	}
	pr.Status = "OPEN"

	if _, err := s.repo.GetPR(ctx, pr.ID); err == nil {
		return nil, errors.New("PR_EXISTS")
	}

	if err := s.repo.CreatePR(ctx, pr); err != nil {
		return nil, fmt.Errorf("create pr: %w", err)
	}

	return s.repo.GetPR(ctx, pr.ID)
}

func (s *prService) MergePR(ctx context.Context, prID string) (*models.PullRequest, error) {
	if prID == "" {
		s.logger.Warn("invalid pr id")
		return nil, errors.New("pr id required")
	}
	pr, err := s.repo.MergePR(ctx, prID)
	if err != nil {
		s.logger.Error("merge pr failed", "err", err)
		return nil, fmt.Errorf("merge pr: %w", err)
	}
	return pr, nil
}

func (s *prService) ReassignReviewer(ctx context.Context, prID, oldUserID string) (*models.PullRequest, string, error) {
	if prID == "" || oldUserID == "" {
		s.logger.Warn("invalid reassign data")
		return nil, "", errors.New("fields required")
	}

	pr, err := s.repo.GetPR(ctx, prID)
	if err != nil {
		s.logger.Error("get pr failed", "err", err)
		return nil, "", fmt.Errorf("get pr: %w", err)
	}

	if pr.Status == "MERGED" {
		return nil, "", errors.New("PR_MERGED")
	}

	found := false
	for _, r := range pr.AssignedReviewers {
		if r == oldUserID {
			found = true
			break
		}
	}
	if !found {
		return nil, "", errors.New("NOT_ASSIGNED")
	}

	teamName, err := s.repo.GetUserTeam(ctx, oldUserID)
	if err != nil {
		s.logger.Error("get old user team failed", "err", err)
		return nil, "", fmt.Errorf("get old team: %w", err)
	}

	newUserID, err := s.repo.RandomActiveMemberFromTeam(ctx, teamName, oldUserID)
	if err != nil {
		s.logger.Error("get new reviewer failed", "err", err)
		return nil, "", fmt.Errorf("get new reviewer: %w", err)
	}

	pr, err = s.repo.ReassignReviewer(ctx, prID, oldUserID, newUserID)
	if err != nil {
		s.logger.Error("reassign failed", "err", err)
		return nil, "", fmt.Errorf("reassign: %w", err)
	}
	return pr, newUserID, nil
}

func (s *prService) GetUserReviews(ctx context.Context, userID string) ([]models.PRShort, error) {
	if userID == "" {
		s.logger.Warn("invalid user id")
		return nil, errors.New("user id required")
	}

	_, err := s.repo.GetUserTeam(ctx, userID)
	if err != nil {
		s.logger.Error("user not found", "err", err)
		return nil, fmt.Errorf("check user: %w", err)
	}

	prs, err := s.repo.GetUserReviewPRs(ctx, userID)
	if err != nil {
		s.logger.Error("get reviews failed", "err", err)
		return nil, fmt.Errorf("get reviews: %w", err)
	}
	return prs, nil
}
