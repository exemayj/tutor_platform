package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tutor_platform/internal/models"
	"tutor_platform/web/templates/layouts"
	"tutor_platform/web/templates/pages"
)

type CatalogHandler struct {
	DB *pgxpool.Pool
}

func NewCatalogHandler(db *pgxpool.Pool) *CatalogHandler {
	return &CatalogHandler{DB: db}
}

// HomePage — главная страница с последними репетиторами
func (h *CatalogHandler) HomePage(w http.ResponseWriter, r *http.Request) {
	tutors, _ := h.getTutors(r, "", "", "", 0, 0, 10, 0, "")
	subjects, _ := h.getSubjects(r)
	userInfo := layouts.GetUserInfo(r.Context())
	pages.HomePage(subjects, tutors, userInfo).Render(r.Context(), w)
}

// CatalogPage — каталог с фильтрами
func (h *CatalogHandler) CatalogPage(w http.ResponseWriter, r *http.Request) {
	subjectSlug := r.URL.Query().Get("subject")
	city := r.URL.Query().Get("city")
	name := r.URL.Query().Get("name")
	priceMinStr := r.URL.Query().Get("price_min")
	priceMaxStr := r.URL.Query().Get("price_max")
	sortBy := r.URL.Query().Get("sort")

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit := 10
	offset := (page - 1) * limit

	priceMin, _ := strconv.Atoi(priceMinStr)
	priceMax, _ := strconv.Atoi(priceMaxStr)

	tutors, _ := h.getTutors(r, subjectSlug, city, name, priceMin, priceMax, limit, offset, sortBy)
	subjects, _ := h.getSubjects(r)
	userInfo := layouts.GetUserInfo(r.Context())

	pages.CatalogPage(subjects, tutors, subjectSlug, city, name, priceMinStr, priceMaxStr, sortBy, page, userInfo).Render(r.Context(), w)
}

func (h *CatalogHandler) TutorPage(w http.ResponseWriter, r *http.Request) {
	tutorID := chi.URLParam(r, "id")
	sentOk := r.URL.Query().Get("sent") == "ok"

	tutor, err := h.getTutorByID(r, tutorID)
	if err != nil {
		http.Error(w, "Репетитор не найден", http.StatusNotFound)
		return
	}

	userInfo := layouts.GetUserInfo(r.Context())
	pages.TutorPage(*tutor, sentOk, userInfo).Render(r.Context(), w)
}

// getTutors — поиск репетиторов с фильтрами
func (h *CatalogHandler) getTutors(r *http.Request, subjectSlug, city, name string, priceMin, priceMax, limit, offset int, sortBy string) ([]models.TutorProfile, error) {
	query := `
		SELECT tp.id, tp.user_id, tp.headline, tp.description, 
		       tp.experience_years, tp.education, tp.price_per_hour, tp.city,
		       u.name, u.avatar_url, tp.rating, tp.reviews_count, tp.is_verified, MAX(tp.created_at) as created_at
		FROM tutor_profiles tp
		JOIN users u ON tp.user_id = u.id
		LEFT JOIN tutor_subjects ts ON tp.id = ts.tutor_profile_id
		LEFT JOIN subjects s ON ts.subject_id = s.id
		WHERE tp.is_active = true
	`

	args := []interface{}{}
	argIdx := 1

	if name != "" {
		query += " AND LOWER(u.name) LIKE LOWER($" + strconv.Itoa(argIdx) + ")"
		args = append(args, "%"+name+"%")
		argIdx++
	}

	if subjectSlug != "" {
		query += " AND s.slug = $" + strconv.Itoa(argIdx)
		args = append(args, subjectSlug)
		argIdx++
	}

	if city != "" {
		query += " AND LOWER(tp.city) = LOWER($" + strconv.Itoa(argIdx) + ")"
		args = append(args, city)
		argIdx++
	}

	if priceMin > 0 {
		query += " AND tp.price_per_hour >= $" + strconv.Itoa(argIdx)
		args = append(args, priceMin)
		argIdx++
	}

	if priceMax > 0 {
		query += " AND tp.price_per_hour <= $" + strconv.Itoa(argIdx)
		args = append(args, priceMax)
		argIdx++
	}

	query += " GROUP BY tp.id, u.name, u.avatar_url"

	switch sortBy {
	case "rating":
		query += " ORDER BY tp.rating DESC, tp.reviews_count DESC"
	case "price_asc":
		query += " ORDER BY tp.price_per_hour ASC"
	case "price_desc":
		query += " ORDER BY tp.price_per_hour DESC"
	default:
		query += " ORDER BY created_at DESC"
	}

	query += " LIMIT $" + strconv.Itoa(argIdx)
	args = append(args, limit)
	argIdx++

	query += " OFFSET $" + strconv.Itoa(argIdx)
	args = append(args, offset)

	rows, err := h.DB.Query(r.Context(), query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tutors []models.TutorProfile
	for rows.Next() {
		var t models.TutorProfile
		var avatarURL *string
		var createdAt interface{}
		err := rows.Scan(&t.ID, &t.UserID, &t.Headline, &t.Description,
			&t.ExperienceYears, &t.Education, &t.PricePerHour, &t.City,
			&t.TutorName, &avatarURL, &t.Rating, &t.ReviewsCount, &t.IsVerified, &createdAt)
		if err != nil {
			return nil, err
		}
		log.Printf("Репетитор: %s, Rating: %f, Reviews: %d", t.Headline, t.Rating, t.ReviewsCount)
		if avatarURL != nil {
			t.TutorAvatar = *avatarURL
		}

		t.Subjects, _ = h.getTutorSubjects(r, t.ID)

		tutors = append(tutors, t)
	}

	return tutors, nil
}

// getTutorByID — один репетитор по ID
func (h *CatalogHandler) getTutorByID(r *http.Request, id string) (*models.TutorProfile, error) {
	var t models.TutorProfile
	var avatarURL *string
	err := h.DB.QueryRow(r.Context(),
		`SELECT tp.id, tp.user_id, tp.headline, tp.description, 
		        tp.experience_years, tp.education, tp.price_per_hour, tp.city,
		        u.name, u.avatar_url, tp.is_verified
		 FROM tutor_profiles tp
		 JOIN users u ON tp.user_id = u.id
		 WHERE tp.id = $1 AND tp.is_active = true`,
		id,
	).Scan(&t.ID, &t.UserID, &t.Headline, &t.Description,
		&t.ExperienceYears, &t.Education, &t.PricePerHour, &t.City,
		&t.TutorName, &avatarURL, &t.IsVerified)

	if err != nil {
		return nil, err
	}

	if avatarURL != nil {
		t.TutorAvatar = *avatarURL
	}

	t.Subjects, _ = h.getTutorSubjects(r, t.ID)

	return &t, nil
}

// getTutorSubjects — предметы репетитора
func (h *CatalogHandler) getTutorSubjects(r *http.Request, profileID string) ([]models.Subject, error) {
	rows, err := h.DB.Query(r.Context(),
		`SELECT s.id, s.name, s.slug 
		 FROM subjects s 
		 JOIN tutor_subjects ts ON s.id = ts.subject_id 
		 WHERE ts.tutor_profile_id = $1`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjects []models.Subject
	for rows.Next() {
		var s models.Subject
		rows.Scan(&s.ID, &s.Name, &s.Slug)
		subjects = append(subjects, s)
	}
	return subjects, nil
}

// getSubjects — все предметы
func (h *CatalogHandler) getSubjects(r *http.Request) ([]models.Subject, error) {
	rows, err := h.DB.Query(r.Context(), "SELECT id, name, slug FROM subjects ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjects []models.Subject
	for rows.Next() {
		var s models.Subject
		rows.Scan(&s.ID, &s.Name, &s.Slug)
		subjects = append(subjects, s)
	}
	return subjects, nil
}

func (h *CatalogHandler) PrivacyPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="ru">
<head><meta charset="utf-8"><title>Политика конфиденциальности</title></head>
<body style="max-width:700px;margin:40px auto;font-family:Arial;">
<h1>Политика конфиденциальности</h1>
<p>Мы собираем следующие данные: имя, email, телефон (при отправке заявки).</p>
<p>Данные используются только для связи между учениками и репетиторами.</p>
<p>Данные хранятся на сервере в России и не передаются третьим лицам.</p>
<p>Вы можете запросить удаление данных, написав на email: support@ваш-домен.ru</p>
</body>
</html>`))
}

func (h *CatalogHandler) TermsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="ru">
<head><meta charset="utf-8"><title>Условия использования</title></head>
<body style="max-width:700px;margin:40px auto;font-family:Arial;">
<h1>Условия использования</h1>
<p>Сервис предоставляет площадку для поиска репетиторов.</p>
<p>Пользователи обязуются указывать достоверную информацию.</p>
<p>Администрация вправе заблокировать пользователя за нарушение правил.</p>
<p>Запрещены оскорбления, спам и мошенничество.</p>
</body>
</html>`))
}
