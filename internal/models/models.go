package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	Name         string    `json:"name"`
	Phone        *string   `json:"phone,omitempty"`      // ← *string вместо string
	AvatarURL    *string   `json:"avatar_url,omitempty"` // ← тоже на всякий случай
	CreatedAt    time.Time `json:"created_at"`
}

type TutorProfile struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	Headline        string    `json:"headline"`
	Description     string    `json:"description"`
	ExperienceYears int       `json:"experience_years"`
	Education       string    `json:"education"`
	PricePerHour    int       `json:"price_per_hour"`
	VideoLink       string    `json:"video_link,omitempty"`
	City            string    `json:"city"`
	IsActive        bool      `json:"is_active"`
	Subjects        []Subject `json:"subjects"`
	CreatedAt       time.Time `json:"created_at"`
	// Для отображения в каталоге (join)
	TutorName    string  `json:"tutor_name,omitempty"`
	TutorAvatar  string  `json:"tutor_avatar,omitempty"`
	Rating       float64 `json:"rating"`
	ReviewsCount int     `json:"reviews_count"`
	IsVerified   bool    `json:"is_verified"`
}

type Subject struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Order struct {
	ID             string    `json:"id"`
	StudentID      string    `json:"student_id"`
	TutorProfileID string    `json:"tutor_profile_id"`
	SubjectID      int       `json:"subject_id"`
	StudentName    string    `json:"student_name"`
	StudentPhone   string    `json:"student_phone"`
	Goal           string    `json:"goal"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type Message struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	SenderID  string    `json:"sender_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

type OrderView struct {
	ID             string
	StudentID      string
	TutorProfileID string
	StudentName    string
	StudentPhone   string
	TutorName      string
	SubjectName    string
	Goal           string
	Status         string
	CreatedAt      time.Time
}

type Review struct {
	ID             string    `json:"id"`
	OrderID        string    `json:"order_id"`
	StudentID      string    `json:"student_id"`
	TutorProfileID string    `json:"tutor_profile_id"`
	Rating         int       `json:"rating"`
	Comment        string    `json:"comment"`
	CreatedAt      time.Time `json:"created_at"`
	StudentName    string    `json:"student_name,omitempty"`
}
