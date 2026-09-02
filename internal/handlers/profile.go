package handlers

import (
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"tutor_platform/internal/middleware"
	"tutor_platform/internal/models"
	"tutor_platform/web/templates/layouts"
	"tutor_platform/web/templates/pages"
)

type ProfileHandler struct {
	DB *pgxpool.Pool
}

func NewProfileHandler(db *pgxpool.Pool) *ProfileHandler {
	return &ProfileHandler{DB: db}
}

// ProfilePage показывает профиль: ученику — его данные, репетитору — анкету
func (h *ProfileHandler) ProfilePage(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "Не авторизован", http.StatusUnauthorized)
		return
	}

	var user models.User
	err := h.DB.QueryRow(r.Context(),
		"SELECT id, email, role, name, phone, avatar_url FROM users WHERE id = $1",
		claims.UserID,
	).Scan(&user.ID, &user.Email, &user.Role, &user.Name, &user.Phone, &user.AvatarURL)

	if err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	var profile *models.TutorProfile
	var subjects []models.Subject

	if user.Role == "tutor" {
		profile, _ = h.getTutorProfile(r, claims.UserID)
		subjects, _ = h.getSubjects(r)
	}

	userInfo := layouts.GetUserInfo(r.Context())
	pages.ProfilePage(user, profile, subjects, userInfo).Render(r.Context(), w)
}

// UpdateProfile обновляет профиль: для репетитора — создаёт/обновляет анкету
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserFromContext(r.Context())
	if claims == nil || claims.Role != "tutor" {
		http.Error(w, "Только репетитор может обновлять анкету", http.StatusForbidden)
		return
	}

	// Парсим multipart form (для файлов)
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		http.Error(w, "Ошибка обработки формы", http.StatusBadRequest)
		return
	}

	headline := strings.TrimSpace(r.FormValue("headline"))
	description := strings.TrimSpace(r.FormValue("description"))
	education := strings.TrimSpace(r.FormValue("education"))
	city := strings.TrimSpace(r.FormValue("city"))
	priceStr := r.FormValue("price_per_hour")
	subjectIDs := r.Form["subject_ids"]

	if headline == "" || description == "" || city == "" {
		http.Error(w, "Заголовок, описание и город обязательны", http.StatusBadRequest)
		return
	}

	price := 0
	if priceStr != "" {
		for _, c := range priceStr {
			if c >= '0' && c <= '9' {
				price = price*10 + int(c-'0')
			}
		}
	}

	// Обработка файла
	avatarURL := ""
	file, header, err := r.FormFile("avatar")
	if err == nil {
		defer file.Close()

		// Генерируем уникальное имя файла
		ext := ""
		if idx := strings.LastIndex(header.Filename, "."); idx != -1 {
			ext = header.Filename[idx:]
		}
		filename := claims.UserID + ext

		// Сохраняем файл
		dst, err := os.Create("web/static/uploads/" + filename)
		if err == nil {
			defer dst.Close()
			io.Copy(dst, file)
			avatarURL = "/static/uploads/" + filename
		}
	}

	// Создаём или обновляем анкету
	if avatarURL != "" {
		_, err = h.DB.Exec(r.Context(),
			`INSERT INTO tutor_profiles (user_id, headline, description, education, city, price_per_hour)
     		 VALUES ($1, $2, $3, $4, $5, $6)
     		 ON CONFLICT (user_id) 
     		 DO UPDATE SET headline = $2, description = $3, education = $4, city = $5, price_per_hour = $6`,
			claims.UserID, headline, description, education, city, price,
		)
	} else {
		_, err = h.DB.Exec(r.Context(),
			`INSERT INTO tutor_profiles (user_id, headline, description, education, city, price_per_hour)
     		 VALUES ($1, $2, $3, $4, $5, $6)
     		 ON CONFLICT (user_id) 
     		 DO UPDATE SET headline = $2, description = $3, education = $4, city = $5, price_per_hour = $6`,
			claims.UserID, headline, description, education, city, price,
		)
	}

	// Обновляем аватарку в users
	if avatarURL != "" {
		h.DB.Exec(r.Context(), "UPDATE users SET avatar_url = $1 WHERE id = $2", avatarURL, claims.UserID)
	}

	if err != nil {
		http.Error(w, "Ошибка сохранения анкеты: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Обновляем предметы
	h.DB.Exec(r.Context(), "DELETE FROM tutor_subjects WHERE tutor_profile_id = (SELECT id FROM tutor_profiles WHERE user_id = $1)", claims.UserID)

	for _, subjID := range subjectIDs {
		h.DB.Exec(r.Context(),
			"INSERT INTO tutor_subjects (tutor_profile_id, subject_id) VALUES ((SELECT id FROM tutor_profiles WHERE user_id = $1), $2)",
			claims.UserID, subjID,
		)
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

// getTutorProfile возвращает анкету репетитора с предметами
func (h *ProfileHandler) getTutorProfile(r *http.Request, userID string) (*models.TutorProfile, error) {
	var p models.TutorProfile
	err := h.DB.QueryRow(r.Context(),
		`SELECT id, user_id, COALESCE(headline,''), COALESCE(description,''), 
		        experience_years, COALESCE(education,''), price_per_hour, 
		        COALESCE(video_link,''), COALESCE(city,''), is_active
		 FROM tutor_profiles WHERE user_id = $1`,
		userID,
	).Scan(&p.ID, &p.UserID, &p.Headline, &p.Description,
		&p.ExperienceYears, &p.Education, &p.PricePerHour,
		&p.VideoLink, &p.City, &p.IsActive)

	if err != nil {
		return nil, err
	}

	// Подгружаем предметы
	rows, err := h.DB.Query(r.Context(),
		`SELECT s.id, s.name, s.slug 
		 FROM subjects s 
		 JOIN tutor_subjects ts ON s.id = ts.subject_id 
		 WHERE ts.tutor_profile_id = $1`, p.ID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var subj models.Subject
			rows.Scan(&subj.ID, &subj.Name, &subj.Slug)
			p.Subjects = append(p.Subjects, subj)
		}
	}

	return &p, nil
}

// getSubjects возвращает все предметы
func (h *ProfileHandler) getSubjects(r *http.Request) ([]models.Subject, error) {
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
