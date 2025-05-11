package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID           int64     `json:"id"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Expression struct {
	ID         int64           `json:"id"`
	UserID     int64           `json:"user_id"`
	Expression string          `json:"expression"`
	Status     string          `json:"status"`
	Result     sql.NullFloat64 `json:"result,omitempty"`
	Steps      sql.NullString  `json:"steps,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type Task struct {
	ID           int64           `json:"id"`
	ExpressionID int64           `json:"expression_id"`
	Operation    string          `json:"operation"`
	Arg1         float64         `json:"arg1"`
	Arg2         float64         `json:"arg2"`
	Result       sql.NullFloat64 `json:"result,omitempty"`
	Status       string          `json:"status"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Retries      int             `json:"retries"`
}
