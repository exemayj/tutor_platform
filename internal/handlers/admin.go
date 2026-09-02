package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutor_platform/internal/models"
	"tutor_platform/web/templates/layouts"
	"tutor_platform/web/templates/pages"
)

type AdminHandler struct {
	DB *pgxpool.Pool
}

func NewAdminHandler(db *pgxpool.Pool) *AdminHandler {
	return &AdminHandler{DB: db}
}

func (h *AdminHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(),
		`SELECT tp.id, tp.headline, tp.is_active, tp.is_verified, u.name
		FROM tutor_profiles tp
		JOIN users u ON tp.user_id = u.id
		ORDER BY tp.created_at DESC`,
	)
	if err != nil {
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tutors []models.TutorProfile
	for rows.Next() {
		var t models.TutorProfile
		rows.Scan(&t.ID, &t.Headline, &t.IsActive, &t.IsVerified, &t.TutorName)
		tutors = append(tutors, t)
	}

	userInfo := layouts.GetUserInfo(r.Context())
	pages.AdminDashboard(tutors, userInfo).Render(r.Context(), w)
}

func (h *AdminHandler) BlockTutor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.DB.Exec(r.Context(), "UPDATE tutor_profiles SET is_active = false WHERE id = $1", id)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *AdminHandler) UnblockTutor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.DB.Exec(r.Context(), "UPDATE tutor_profiles SET is_active = true WHERE id = $1", id)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *AdminHandler) VerifyTutor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.DB.Exec(r.Context(), "UPDATE tutor_profiles SET is_verified = true WHERE id = $1", id)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *AdminHandler) UnverifyTutor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.DB.Exec(r.Context(), "UPDATE tutor_profiles SET is_verified = false WHERE id = $1", id)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
