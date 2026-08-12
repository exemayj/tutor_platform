package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutor_platform/internal/middleware"
	"tutor_platform/internal/models"
	"tutor_platform/web/templates/layouts"
	"tutor_platform/web/templates/pages"
)

type OrdersHandler struct {
	DB *pgxpool.Pool
}

func NewOrdersHandler(db *pgxpool.Pool) *OrdersHandler {
	return &OrdersHandler{DB: db}
}

func (h *OrdersHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	r.ParseForm()

	tutorProfileID := strings.TrimSpace(r.FormValue("tutor_profile_id"))
	studentName := strings.TrimSpace(r.FormValue("student_name"))
	studentPhone := strings.TrimSpace(r.FormValue("student_phone"))
	goal := strings.TrimSpace(r.FormValue("goal"))

	if tutorProfileID == "" || studentName == "" || studentPhone == "" {
		http.Error(w, "Все поля обязательны", http.StatusBadRequest)
		return
	}

	var tutorExists bool
	err := h.DB.QueryRow(r.Context(),
		"SELECT EXISTS(SELECT 1 FROM tutor_profiles WHERE id = $1 AND is_active = true)",
		tutorProfileID,
	).Scan(&tutorExists)

	if err != nil || !tutorExists {
		http.Error(w, "Репетитор не найден", http.StatusNotFound)
		return
	}

	_, err = h.DB.Exec(r.Context(),
		`INSERT INTO orders (student_id, tutor_profile_id, student_name, student_phone, goal, status)
		 VALUES ($1, $2, $3, $4, $5, 'new')`,
		claims.UserID, tutorProfileID, studentName, studentPhone, goal,
	)

	if err != nil {
		http.Error(w, "Ошибка создания заявки", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/tutor/"+tutorProfileID+"?sent=ok", http.StatusSeeOther)
}

func (h *OrdersHandler) StudentOrdersPage(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	rows, err := h.DB.Query(r.Context(),
		`SELECT o.id, o.student_id, o.tutor_profile_id, o.student_name, o.student_phone,
		        COALESCE(u.name, ''), COALESCE(s.name, ''), COALESCE(o.goal, ''),
		        o.status, o.created_at
		 FROM orders o
		 JOIN tutor_profiles tp ON o.tutor_profile_id = tp.id
		 JOIN users u ON tp.user_id = u.id
		 LEFT JOIN subjects s ON o.subject_id = s.id
		 WHERE o.student_id = $1
		 ORDER BY o.created_at DESC`,
		claims.UserID,
	)
	if err != nil {
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []models.OrderView
	for rows.Next() {
		var o models.OrderView
		rows.Scan(&o.ID, &o.StudentID, &o.TutorProfileID, &o.StudentName, &o.StudentPhone,
			&o.TutorName, &o.SubjectName, &o.Goal, &o.Status, &o.CreatedAt)
		orders = append(orders, o)
	}

	userInfo := layouts.GetUserInfo(r.Context())
	pages.StudentOrdersPage(orders, userInfo).Render(r.Context(), w)
}

func (h *OrdersHandler) TutorOrdersPage(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "tutor" {
		http.Error(w, "Только для репетиторов", http.StatusForbidden)
		return
	}

	var profileID string
	err := h.DB.QueryRow(r.Context(),
		"SELECT id FROM tutor_profiles WHERE user_id = $1",
		claims.UserID,
	).Scan(&profileID)

	if err != nil {
		http.Error(w, "Сначала создайте анкету", http.StatusBadRequest)
		return
	}

	rows, err := h.DB.Query(r.Context(),
		`SELECT o.id, o.student_id, o.tutor_profile_id, o.student_name, o.student_phone,
		        COALESCE(u.name, ''), COALESCE(s.name, ''), COALESCE(o.goal, ''),
		        o.status, o.created_at
		 FROM orders o
		 LEFT JOIN users u ON o.student_id = u.id
		 LEFT JOIN subjects s ON o.subject_id = s.id
		 WHERE o.tutor_profile_id = $1
		 ORDER BY o.created_at DESC`,
		profileID,
	)
	if err != nil {
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []models.OrderView
	for rows.Next() {
		var o models.OrderView
		rows.Scan(&o.ID, &o.StudentID, &o.TutorProfileID, &o.StudentName, &o.StudentPhone,
			&o.TutorName, &o.SubjectName, &o.Goal, &o.Status, &o.CreatedAt)
		orders = append(orders, o)
	}

	userInfo := layouts.GetUserInfo(r.Context())
	pages.TutorOrdersPage(orders, userInfo).Render(r.Context(), w)
}

func (h *OrdersHandler) AcceptOrder(w http.ResponseWriter, r *http.Request) {
	h.changeOrderStatus(w, r, "accepted")
}

func (h *OrdersHandler) DeclineOrder(w http.ResponseWriter, r *http.Request) {
	h.changeOrderStatus(w, r, "declined")
}

func (h *OrdersHandler) changeOrderStatus(w http.ResponseWriter, r *http.Request, status string) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "tutor" {
		http.Error(w, "Только для репетиторов", http.StatusForbidden)
		return
	}

	orderID := chi.URLParam(r, "id")

	var profileID string
	h.DB.QueryRow(r.Context(),
		"SELECT id FROM tutor_profiles WHERE user_id = $1", claims.UserID,
	).Scan(&profileID)

	h.DB.Exec(r.Context(),
		"UPDATE orders SET status = $1 WHERE id = $2 AND tutor_profile_id = $3",
		status, orderID, profileID,
	)

	http.Redirect(w, r, "/orders/received", http.StatusSeeOther)
}

func (h *OrdersHandler) CompleteOrder(w http.ResponseWriter, r *http.Request) {
	h.changeOrderStatus(w, r, "completed")
}
