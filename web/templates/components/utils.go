package components

import (
	"strconv"
	"tutor_platform/internal/models"
)

func SubjectNames(subjects []models.Subject) string {
	var names []string
	for _, s := range subjects {
		names = append(names, s.Name)
	}
	result := ""
	for i, n := range names {
		if i > 0 {
			result += ", "
		}
		result += n
	}
	return result
}

func RoleLabel(role string) string {
	if role == "tutor" {
		return "Репетитор"
	}
	if role == "student" {
		return "Ученик"
	}
	return role
}

func HasSubject(subjects []models.Subject, id int) bool {
	for _, s := range subjects {
		if s.ID == id {
			return true
		}
	}
	return false
}

func IntToStr(i int) string {
	return strconv.Itoa(i)
}

func StatusLabel(status string) string {
	switch status {
	case "new":
		return "Новая"
	case "accepted":
		return "Принята"
	case "declined":
		return "Отклонена"
	case "completed":
		return "Завершена"
	default:
		return status
	}
}
