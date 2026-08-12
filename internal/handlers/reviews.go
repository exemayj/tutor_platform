package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutor_platform/internal/middleware"
	"tutor_platform/internal/models"
)

type ReviewsHandler struct {
	DB *pgxpool.Pool
}

func NewReviewsHandler(db *pgxpool.Pool) *ReviewsHandler {
	return &ReviewsHandler{DB: db}
}

// CreateReview — создать отзыв
func (h *ReviewsHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Требуется авторизация", http.StatusUnauthorized)
		return
	}

	r.ParseForm()

	orderID := strings.TrimSpace(r.FormValue("order_id"))
	tutorProfileID := strings.TrimSpace(r.FormValue("tutor_profile_id"))
	ratingStr := r.FormValue("rating")
	comment := strings.TrimSpace(r.FormValue("comment"))

	rating, err := strconv.Atoi(ratingStr)
	if err != nil || rating < 1 || rating > 5 {
		http.Error(w, "Оценка должна быть от 1 до 5", http.StatusBadRequest)
		return
	}

	// Проверяем, что заявка завершена и принадлежит ученику
	var status string
	err = h.DB.QueryRow(r.Context(),
		"SELECT status FROM orders WHERE id = $1 AND student_id = $2",
		orderID, claims.UserID,
	).Scan(&status)

	if err != nil {
		http.Error(w, "Заявка не найдена", http.StatusNotFound)
		return
	}

	if status != "completed" {
		http.Error(w, "Отзыв можно оставить только для завершённой заявки", http.StatusBadRequest)
		return
	}

	// Проверяем, что отзыва ещё нет
	var exists bool
	h.DB.QueryRow(r.Context(),
		"SELECT EXISTS(SELECT 1 FROM reviews WHERE order_id = $1)", orderID,
	).Scan(&exists)

	if exists {
		http.Error(w, "Вы уже оставили отзыв", http.StatusConflict)
		return
	}

	// Сохраняем отзыв
	_, err = h.DB.Exec(r.Context(),
		"INSERT INTO reviews (order_id, student_id, tutor_profile_id, rating, comment) VALUES ($1, $2, $3, $4, $5)",
		orderID, claims.UserID, tutorProfileID, rating, comment,
	)
	if err != nil {
		http.Error(w, "Ошибка сохранения отзыва", http.StatusInternalServerError)
		return
	}

	// Обновляем рейтинг репетитора
	h.DB.Exec(r.Context(),
		`UPDATE tutor_profiles 
		 SET rating = (SELECT COALESCE(AVG(rating), 0) FROM reviews WHERE tutor_profile_id = $1),
		     reviews_count = (SELECT COUNT(*) FROM reviews WHERE tutor_profile_id = $1)
		 WHERE id = $1`,
		tutorProfileID,
	)

	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

// GetReviews — получить отзывы репетитора
func (h *ReviewsHandler) GetReviews(w http.ResponseWriter, r *http.Request) {
	tutorProfileID := chi.URLParam(r, "id")

	rows, err := h.DB.Query(r.Context(),
		`SELECT r.id, r.rating, r.comment, r.created_at, u.name
		 FROM reviews r
		 JOIN users u ON r.student_id = u.id
		 WHERE r.tutor_profile_id = $1
		 ORDER BY r.created_at DESC
		 LIMIT 20`,
		tutorProfileID,
	)
	if err != nil {
		http.Error(w, "Ошибка", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var reviews []models.Review
	for rows.Next() {
		var rev models.Review
		rows.Scan(&rev.ID, &rev.Rating, &rev.Comment, &rev.CreatedAt, &rev.StudentName)
		reviews = append(reviews, rev)
	}

	// Пока JSON, позже сделаем templ-шаблон
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if len(reviews) == 0 {
		w.Write([]byte("<p>Пока нет отзывов</p>"))
		return
	}

	for _, rev := range reviews {
		stars := strings.Repeat("★", rev.Rating) + strings.Repeat("☆", 5-rev.Rating)
		w.Write([]byte("<div style='border-bottom:1px solid #eee; padding:10px 0;'>"))
		w.Write([]byte("<p><strong>" + rev.StudentName + "</strong> " + stars + "</p>"))
		w.Write([]byte("<p>" + rev.Comment + "</p>"))
		w.Write([]byte("<p style='color:#999;font-size:12px;'>" + rev.CreatedAt.Format("02.01.2006") + "</p>"))
		w.Write([]byte("</div>"))
	}
}
