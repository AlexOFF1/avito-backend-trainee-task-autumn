package delivery

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	d "github.com/AlexOFF1/avito-backend-trainee-task-autumn/internal/delivery/dto"
	"github.com/AlexOFF1/avito-backend-trainee-task-autumn/internal/models"
	"github.com/AlexOFF1/avito-backend-trainee-task-autumn/internal/usecase"
)

type Handler struct {
	service usecase.PRService
	logger  *slog.Logger
}

func NewHandler(service usecase.PRService, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (h *Handler) sendError(w http.ResponseWriter, status int, code, msg string) {
	resp := errorResponse{}
	resp.Error.Code = code
	resp.Error.Message = msg
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) AddTeam(w http.ResponseWriter, r *http.Request) {
	var req d.TeamDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid json")
		return
	}

	team := models.Team{Name: req.TeamName}
	for _, m := range req.Members {
		team.Members = append(team.Members, models.Member{
			ID:       m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}

	created, err := h.service.CreateTeam(r.Context(), team)
	if err != nil {
		if errors.Is(err, errors.New("TEAM_EXISTS")) {
			h.sendError(w, http.StatusBadRequest, "TEAM_EXISTS", "team_name already exists")
		} else {
			h.logger.Error("create team failed", "err", err)
			h.sendError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		}
		return
	}

	resp := d.TeamResponse{TeamName: created.Name}
	for _, m := range created.Members {
		resp.Members = append(resp.Members, d.MemberDTO{
			UserID:   m.ID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"team": resp})
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		h.sendError(w, http.StatusBadRequest, "INVALID_REQUEST", "team_name required")
		return
	}

	team, err := h.service.GetTeam(r.Context(), teamName)
	if err != nil {
		if err.Error() == "team not found" {
			h.sendError(w, http.StatusNotFound, "NOT_FOUND", "team not found")
		} else {
			h.logger.Error("get team failed", "err", err)
			h.sendError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		}
		return
	}

	resp := d.TeamResponse{TeamName: team.Name}
	for _, m := range team.Members {
		resp.Members = append(resp.Members, d.MemberDTO{
			UserID:   m.ID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) SetIsActive(w http.ResponseWriter, r *http.Request) {
	var req d.UserActiveDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid json")
		return
	}

	user, err := h.service.SetUserActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		if err.Error() == "user not found" {
			h.sendError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		} else {
			h.logger.Error("set active failed", "err", err)
			h.sendError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		}
		return
	}

	resp := d.UserResponse{
		UserID:   user.ID,
		Username: user.Username,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"user": resp})
}

func (h *Handler) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req d.PRCreateDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid json")
		return
	}

	pr := models.PullRequest{
		ID:       req.PullRequestID,
		Name:     req.PullRequestName,
		AuthorID: req.AuthorID,
	}

	created, err := h.service.CreatePR(r.Context(), pr)
	if err != nil {
		if errors.Is(err, errors.New("PR_EXISTS")) {
			h.sendError(w, http.StatusConflict, "PR_EXISTS", "PR id already exists")
		} else if err.Error() == "user not found" {
			h.sendError(w, http.StatusNotFound, "NOT_FOUND", "author not found")
		} else {
			h.logger.Error("create pr failed", "err", err)
			h.sendError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		}
		return
	}

	resp := d.PRResponse{
		PullRequestID:     created.ID,
		PullRequestName:   created.Name,
		AuthorID:          created.AuthorID,
		Status:            created.Status,
		AssignedReviewers: created.AssignedReviewers,
		CreatedAt:         created.CreatedAt,
		MergedAt:          created.MergedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"pr": resp})
}

func (h *Handler) MergePR(w http.ResponseWriter, r *http.Request) {
	var req d.PRMergeDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid json")
		return
	}

	pr, err := h.service.MergePR(r.Context(), req.PullRequestID)
	if err != nil {
		if err.Error() == "pr not found" {
			h.sendError(w, http.StatusNotFound, "NOT_FOUND", "PR not found")
		} else {
			h.logger.Error("merge pr failed", "err", err)
			h.sendError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		}
		return
	}

	resp := d.PRResponse{
		PullRequestID:     pr.ID,
		PullRequestName:   pr.Name,
		AuthorID:          pr.AuthorID,
		Status:            pr.Status,
		AssignedReviewers: pr.AssignedReviewers,
		CreatedAt:         pr.CreatedAt,
		MergedAt:          pr.MergedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"pr": resp})
}

func (h *Handler) Reassign(w http.ResponseWriter, r *http.Request) {
	var req d.PRReassignDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid json")
		return
	}

	pr, replacedBy, err := h.service.ReassignReviewer(r.Context(), req.PullRequestID, req.OldUserID)
	if err != nil {
		switch {
		case errors.Is(err, errors.New("PR_MERGED")):
			h.sendError(w, http.StatusConflict, "PR_MERGED", "cannot reassign on merged PR")
		case errors.Is(err, errors.New("NOT_ASSIGNED")):
			h.sendError(w, http.StatusConflict, "NOT_ASSIGNED", "reviewer is not assigned to this PR")
		case err.Error() == "no candidate":
			h.sendError(w, http.StatusConflict, "NO_CANDIDATE", "no active replacement candidate in team")
		default:
			h.sendError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		}
		return
	}

	resp := d.PRResponse{
		PullRequestID:     pr.ID,
		PullRequestName:   pr.Name,
		AuthorID:          pr.AuthorID,
		Status:            pr.Status,
		AssignedReviewers: pr.AssignedReviewers,
		CreatedAt:         pr.CreatedAt,
		MergedAt:          pr.MergedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"pr": resp, "replaced_by": replacedBy})
}

func (h *Handler) GetReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.sendError(w, http.StatusBadRequest, "INVALID_REQUEST", "user_id required")
		return
	}

	prs, err := h.service.GetUserReviews(r.Context(), userID)
	if err != nil {
		if err.Error() == "user not found" {
			h.sendError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
		} else {
			h.logger.Error("get reviews failed", "err", err)
			h.sendError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		}
		return
	}

	resp := d.UserReviewsResponse{UserID: userID}
	for _, pr := range prs {
		resp.PullRequests = append(resp.PullRequests, d.PRShortResponse{
			PullRequestID:   pr.ID,
			PullRequestName: pr.Name,
			AuthorID:        pr.AuthorID,
			Status:          pr.Status,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
