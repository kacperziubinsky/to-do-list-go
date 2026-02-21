package models

import (
	"fmt"
	"strings"
	"time"
)

type JSONTime time.Time

func (jt *JSONTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "" {
		*jt = JSONTime(time.Now())
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}
	*jt = JSONTime(t)
	return nil
}

func (jt JSONTime) MarshalJSON() ([]byte, error) {
	t := time.Time(jt)
	return []byte(fmt.Sprintf("\"%s\"", t.Format("2006-01-02"))), nil
}

type Task struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Date        JSONTime `json:"date"`
	UserID      int      `json:"user_id"`
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	UserID   int    `json:"user_id"`
}

type UpdateTaskRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Date        JSONTime `json:"date"`
}

type TaskQueryParams struct {
	Search  string // keyword in name or description
	SortBy  string // "date", "name", "status"
	Order   string // "asc", "desc"
	Status  string // filter by status
	DateFrom string // "2006-01-02"
	DateTo   string // "2006-01-02"
}