package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"tutor_platform/internal/config"
	"tutor_platform/internal/database"
	"tutor_platform/internal/handlers"
	mw "tutor_platform/internal/middleware"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Ошибка БД: %v", err)
	}
	defer db.Close()

	// Инициализируем хендлеры
	authHandler := handlers.NewAuthHandler(db, cfg.JWTSecret)
	profileHandler := handlers.NewProfileHandler(db)
	catalogHandler := handlers.NewCatalogHandler(db)
	ordersHandler := handlers.NewOrdersHandler(db)
	chatHandler := handlers.NewChatHandler(db)
	adminHandler := handlers.NewAdminHandler(db)
	reviewsHandler := handlers.NewReviewsHandler(db)

	// Rate limiter: 5 запросов в секунду, максимум 10
	authLimiter := mw.NewRateLimiter(5, 10)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Статика
	r.Handle("/static/*", http.StripPrefix("/static/",
		http.FileServer(http.Dir("web/static"))))

	// Публичные страницы (с опциональной авторизацией)
	r.Group(func(r chi.Router) {
		r.Use(mw.OptionalAuth(cfg.JWTSecret))
		r.Use(authLimiter.Middleware)

		r.Get("/", catalogHandler.HomePage)
		r.Get("/catalog", catalogHandler.CatalogPage)
		r.Get("/tutor/{id}", catalogHandler.TutorPage)
		r.Get("/login", authHandler.LoginPage)
		r.Post("/login", authHandler.Login)
		r.Get("/register", authHandler.RegisterPage)
		r.Post("/register", authHandler.Register)
		r.Get("/logout", authHandler.Logout)
		r.Get("/privacy", catalogHandler.PrivacyPage)
		r.Get("/terms", catalogHandler.TermsPage)
	})

	// Для авторизованных
	r.Group(func(r chi.Router) {
		r.Use(mw.AuthMiddleware(cfg.JWTSecret))

		r.Get("/profile", profileHandler.ProfilePage)
		r.Post("/profile", profileHandler.UpdateProfile)
		r.Get("/orders", ordersHandler.StudentOrdersPage)
		r.Get("/orders/received", ordersHandler.TutorOrdersPage)
		r.Post("/orders", ordersHandler.CreateOrder)
		r.Get("/orders/{id}/accept", ordersHandler.AcceptOrder)
		r.Get("/orders/{id}/decline", ordersHandler.DeclineOrder)
		r.Get("/orders/{id}/complete", ordersHandler.CompleteOrder)
		r.Get("/chat/{orderID}", chatHandler.ChatPage)
		r.Get("/ws/chat/{orderID}", chatHandler.ChatWebSocket)
		r.Post("/reviews", reviewsHandler.CreateReview)
		r.Get("/reviews/{id}", reviewsHandler.GetReviews)
	})

	// Админка
	r.Group(func(r chi.Router) {
		r.Use(mw.AuthMiddleware(cfg.JWTSecret))
		r.Use(mw.AdminOnly)

		r.Get("/admin", adminHandler.Dashboard)
		r.Get("/admin/tutors/{id}/block", adminHandler.BlockTutor)
		r.Get("/admin/tutors/{id}/unblock", adminHandler.UnblockTutor)
		r.Get("/admin/tutors/{id}/verify", adminHandler.VerifyTutor)
		r.Get("/admin/tutors/{id}/unverify", adminHandler.UnverifyTutor)
	})

	log.Printf("Сервер на :%s", cfg.ServerPort)
	http.ListenAndServe(":"+cfg.ServerPort, r)
}
