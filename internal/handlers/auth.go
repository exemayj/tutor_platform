package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"tutor_platform/internal/middleware"
	"tutor_platform/internal/models"
	"tutor_platform/web/templates/layouts"
	"tutor_platform/web/templates/pages"
)

type AuthHandler struct {
	DB        *pgxpool.Pool
	JWTSecret string
}

func NewAuthHandler(db *pgxpool.Pool, jwtSecret string) *AuthHandler {
	return &AuthHandler{DB: db, JWTSecret: jwtSecret}
}

func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	userInfo := layouts.GetUserInfo(r.Context())
	pages.LoginPage("", userInfo).Render(r.Context(), w)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {

	err := r.ParseForm()
	if err != nil {
		log.Printf("ParseForm error: %v", err)
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	if email == "" || password == "" {
		http.Error(w, "Email и пароль обязательны", http.StatusBadRequest)
		return
	}

	var user models.User
	err = h.DB.QueryRow(r.Context(),
		"SELECT id, email, password_hash, role, name, phone, avatar_url FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.Name, &user.Phone, &user.AvatarURL)

	if err != nil {
		http.Error(w, "Неверный email или пароль", http.StatusUnauthorized)
		return
	}
	if err != nil {
		log.Printf("Ошибка БД: %v", err)
		http.Error(w, "Ошибка сервера: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		http.Error(w, "Неверный email или пароль", http.StatusUnauthorized)
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.Email, user.Role, h.JWTSecret)
	if err != nil {
		log.Printf("Ошибка токена: %v", err)
		http.Error(w, "Ошибка создания токена", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	userInfo := layouts.GetUserInfo(r.Context())
	pages.RegisterPage("", userInfo).Render(r.Context(), w)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	name := strings.TrimSpace(r.FormValue("name"))
	role := r.FormValue("role")
	agree := r.FormValue("agree")

	if email == "" || password == "" || name == "" || role == "" {
		http.Error(w, "Все поля обязательны", http.StatusBadRequest)
		return
	}

	if agree != "on" {
		http.Error(w, "Необходимо согласие на обработку данных", http.StatusBadRequest)
		return
	}

	if role != "tutor" && role != "student" {
		http.Error(w, "Роль должна быть tutor или student", http.StatusBadRequest)
		return
	}

	if len(password) < 6 {
		http.Error(w, "Пароль должен быть не менее 6 символов", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
		return
	}

	var exists bool
	h.DB.QueryRow(r.Context(),
		"SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", email,
	).Scan(&exists)

	if exists {
		http.Error(w, "Пользователь с таким email уже существует", http.StatusConflict)
		return
	}

	var userID string
	err = h.DB.QueryRow(r.Context(),
		`INSERT INTO users (email, password_hash, role, name) 
		 VALUES ($1, $2, $3, $4) 
		 RETURNING id`,
		email, string(hash), role, name,
	).Scan(&userID)

	if err != nil {
		log.Printf("Ошибка вставки: %v", err)
		http.Error(w, "Ошибка создания пользователя", http.StatusInternalServerError)
		return
	}

	token, err := middleware.GenerateToken(userID, email, role, h.JWTSecret)
	if err != nil {
		http.Error(w, "Ошибка создания токена", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
