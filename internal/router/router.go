package router

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/AlexOFF1/avito-backend-trainee-task-autumn/internal/delivery"
)

func Router(h *delivery.Handler) *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/team/add", h.AddTeam).Methods("POST")
	r.HandleFunc("/team/get", h.GetTeam).Methods("GET")
	r.HandleFunc("/users/setIsActive", h.SetIsActive).Methods("POST")
	r.HandleFunc("/users/getReview", h.GetReviews).Methods("GET")

	r.HandleFunc("/pullRequest/create", h.CreatePR).Methods("POST")
	r.HandleFunc("/pullRequest/merge", h.MergePR).Methods("POST")
	r.HandleFunc("/pullRequest/reassign", h.Reassign).Methods("POST")

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }).Methods("GET")

	return r
}
