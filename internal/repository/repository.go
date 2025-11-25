package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlexOFF1/avito-backend-trainee-task-autumn/internal/models"
)

var (
	ErrTeamNotFound = errors.New("team not found")
	ErrUserNotFound = errors.New("user not found")
	ErrPRNotFound   = errors.New("pr not found")
)

type repo struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewRepository(pool *pgxpool.Pool, logger *slog.Logger) PRRepository {
	return &repo{pool: pool, logger: logger}
}

func (r *repo) CreateOrUpdateTeam(ctx context.Context, team models.Team) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `INSERT INTO teams (team_name) VALUES ($1) ON CONFLICT DO NOTHING`, team.Name)
	if err != nil {
		return fmt.Errorf("insert team: %w", err)
	}

	for _, m := range team.Members {
		_, err = tx.Exec(ctx,
			`INSERT INTO users (user_id, username, team_name, is_active)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (user_id) DO UPDATE SET username = $2, team_name = $3, is_active = $4`,
			m.ID, m.Username, team.Name, m.IsActive)
		if err != nil {
			return fmt.Errorf("upsert user: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *repo) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)`, teamName,
	).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("check team exists in teams table: %w", err)
	}
	if !exists {
		return nil, ErrTeamNotFound
	}

	team := &models.Team{Name: teamName}

	rows, err := r.pool.Query(ctx,
		`SELECT user_id, username, is_active FROM users WHERE team_name = $1 ORDER BY user_id`, teamName)
	if err != nil {
		return nil, fmt.Errorf("query team members: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var m models.Member
		if err := rows.Scan(&m.ID, &m.Username, &m.IsActive); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		team.Members = append(team.Members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return team, nil
}

func (r *repo) SetUserActive(ctx context.Context, userID string, isActive bool) (*models.User, error) {
	var u models.User
	err := r.pool.QueryRow(ctx, `
		UPDATE users SET is_active = $2 WHERE user_id = $1
		RETURNING user_id, username, team_name, is_active`,
		userID, isActive).Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("set active: %w", err)
	}
	return &u, nil
}

func (r *repo) GetUserTeam(ctx context.Context, userID string) (string, error) {
	var teamName string
	err := r.pool.QueryRow(ctx, `SELECT team_name FROM users WHERE user_id = $1`, userID).Scan(&teamName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("user not found")
		}
		return "", fmt.Errorf("get user team: %w", err)
	}
	return teamName, nil
}

func (r *repo) RandomActiveMemberFromTeam(ctx context.Context, teamName, excludeID string) (string, error) {
	var userID string
	err := r.pool.QueryRow(ctx, `
		SELECT user_id FROM users
		WHERE team_name = $1 AND is_active = true AND user_id != $2
		ORDER BY RANDOM() LIMIT 1`, teamName, excludeID).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("no candidate")
		}
		return "", fmt.Errorf("random member: %w", err)
	}
	return userID, nil
}

func (r *repo) GetActiveMembersExcluding(ctx context.Context, teamName, excludeID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id FROM users 
		WHERE team_name = $1 AND is_active = true AND user_id != $2`, teamName, excludeID)
	if err != nil {
		return nil, fmt.Errorf("query active members: %w", err)
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		members = append(members, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return members, nil
}

func (r *repo) GetUserReviewPRs(ctx context.Context, userID string) ([]models.PRShort, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT p.pull_request_id, p.pull_request_name, p.author_id, p.status
		FROM pull_requests p, unnest(p.assigned_reviewers) AS r
		WHERE r = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("query reviews: %w", err)
	}
	defer rows.Close()

	var prs []models.PRShort
	for rows.Next() {
		var pr models.PRShort
		if err := rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status); err != nil {
			return nil, fmt.Errorf("scan pr: %w", err)
		}
		prs = append(prs, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return prs, nil
}

func (r *repo) CreatePR(ctx context.Context, pr models.PullRequest) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, assigned_reviewers, created_at)
		VALUES ($1, $2, $3, 'OPEN', $4, NOW())`,
		pr.ID, pr.Name, pr.AuthorID, pr.AssignedReviewers)
	if err != nil {
		return fmt.Errorf("create pr: %w", err)
	}
	return nil
}

func (r *repo) GetPR(ctx context.Context, prID string) (*models.PullRequest, error) {
	pr := &models.PullRequest{}
	var createdAt, mergedAt *time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT pull_request_id, pull_request_name, author_id, status, assigned_reviewers, created_at, merged_at
		FROM pull_requests WHERE pull_request_id = $1`, prID).
		Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.AssignedReviewers, &createdAt, &mergedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("pr not found")
		}
		return nil, fmt.Errorf("get pr: %w", err)
	}

	pr.CreatedAt = createdAt
	pr.MergedAt = mergedAt
	return pr, nil
}

func (r *repo) MergePR(ctx context.Context, prID string) (*models.PullRequest, error) {
	pr := &models.PullRequest{}
	var createdAt, mergedAt *time.Time

	err := r.pool.QueryRow(ctx, `
		UPDATE pull_requests SET status = 'MERGED', merged_at = COALESCE(merged_at, NOW())
		WHERE pull_request_id = $1 AND status = 'OPEN'
		RETURNING pull_request_id, pull_request_name, author_id, status, assigned_reviewers, created_at, merged_at`, prID).
		Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.AssignedReviewers, &createdAt, &mergedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return r.GetPR(ctx, prID)
		}
		return nil, fmt.Errorf("merge pr: %w", err)
	}

	pr.CreatedAt = createdAt
	pr.MergedAt = mergedAt
	return pr, nil
}

func (r *repo) ReassignReviewer(ctx context.Context, prID, oldUserID, newUserID string) (*models.PullRequest, error) {
	pr := &models.PullRequest{}
	var createdAt, mergedAt *time.Time

	err := r.pool.QueryRow(ctx, `
		UPDATE pull_requests 
		SET assigned_reviewers = array_replace(assigned_reviewers, $2, $3)
		WHERE pull_request_id = $1 AND status = 'OPEN' AND $2 = ANY(assigned_reviewers)
		RETURNING pull_request_id, pull_request_name, author_id, status, assigned_reviewers, created_at, merged_at`,
		prID, oldUserID, newUserID).
		Scan(&pr.ID, &pr.Name, &pr.AuthorID, &pr.Status, &pr.AssignedReviewers, &createdAt, &mergedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("cannot reassign")
		}
		return nil, fmt.Errorf("reassign: %w", err)
	}

	pr.CreatedAt = createdAt
	pr.MergedAt = mergedAt
	return pr, nil
}
