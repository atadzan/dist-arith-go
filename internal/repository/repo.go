package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/atadzan/dist-arith-go/internal/constants"
	"github.com/atadzan/dist-arith-go/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

type Repository interface {
	CreateTables() error
	CreateUser(login, passwordHash string) (int64, error)
	GetUserByLogin(login string) (*models.User, error)
	CreateExpression(userID int64, expression string) (int64, error)
	GetExpressionByID(id, userID int64) (*models.Expression, error)
	GetExpressionsByUserID(userID int64) ([]models.Expression, error)
	UpdateExpressionStatusResult(id int64, status string, result sql.NullFloat64, stepsJSON sql.NullString) error
	CreateTask(expressionID int64, operation string, arg1, arg2 float64) (int64, error)
	GetAndLeasePendingTask() (*models.Task, error)
	CompleteTask(taskID int64, result float64) error
	FailTask(taskID int64) error
	GetTaskByID(taskID int64) (*models.Task, error)
	HasPendingTasks(expressionID int64) (bool, error)
	GetExpressionByIDInternal(id int64) (*models.Expression, error)
	GetAllTasksForExpression(expressionID int64) ([]models.Task, error)
}

type repo struct {
	db *sql.DB
	mx *sync.RWMutex
}

func New(db *sql.DB) (Repository, error) {
	return &repo{db: db, mx: new(sync.RWMutex)}, nil
}

func (r *repo) CreateTables() error {
	migrationTables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			login TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS expressions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			expression TEXT NOT NULL,
			status TEXT NOT NULL,
			result REAL,
			steps TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			expression_id INTEGER NOT NULL,
			operation TEXT NOT NULL,
			arg1 REAL NOT NULL,
			arg2 REAL NOT NULL,
			result REAL,
			status TEXT NOT NULL,
			retries INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(expression_id) REFERENCES expressions(id)
		)`,
	}
	for _, tableCreateQuery := range migrationTables {
		if _, err := r.db.Exec(tableCreateQuery); err != nil {
			return fmt.Errorf("occured error while applying db migration. Err: %v", err)
		}
	}
	return nil
}

func (r *repo) CreateUser(login, passwordHash string) (int64, error) {
	r.mx.Lock()
	defer r.mx.Unlock()

	query := `INSERT INTO users (login, password_hash) VALUES (?, ?)`
	res, err := r.db.Exec(query, login, passwordHash)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: users.login") {
			return 0, fmt.Errorf("user with this login '%s' exists", login)
		}
		return 0, fmt.Errorf("can't create user. Err: %v", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("occured error while getting userId. Err: %v", err)
	}

	return id, nil
}

func (r *repo) GetUserByLogin(login string) (*models.User, error) {
	r.mx.RLock()
	defer r.mx.RUnlock()

	query := `SELECT id, login, password_hash, created_at FROM users WHERE login = ?`
	row := r.db.QueryRow(query, login)

	user := new(models.User)
	err := row.Scan(&user.ID, &user.Login, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("can't find user. Login: '%s'. Err: %v", login, err)
	}

	return user, nil
}

func (r *repo) CreateExpression(userID int64, expression string) (int64, error) {
	r.mx.Lock()
	defer r.mx.Unlock()

	query := `INSERT INTO expressions (user_id, expression, status) VALUES (?, ?, ?)`
	res, err := r.db.Exec(query, userID, expression, constants.StatusPending)
	if err != nil {
		return 0, fmt.Errorf("can't create expression. Err: %v", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("can't get userId. Err: %v", err)
	}
	return id, nil
}

func (r *repo) GetExpressionByID(id, userID int64) (*models.Expression, error) {
	r.mx.RLock()
	defer r.mx.RUnlock()

	query := `SELECT id, user_id, expression, status, result, steps, created_at, updated_at
	         FROM expressions WHERE id = ? AND user_id = ?`
	row := r.db.QueryRow(query, id, userID)

	expr := new(models.Expression)
	err := row.Scan(
		&expr.ID, &expr.UserID, &expr.Expression, &expr.Status,
		&expr.Result, &expr.Steps, &expr.CreatedAt, &expr.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("can't get expression. Id: %d, Err: %v", id, err)
	}
	return expr, nil
}

func (r *repo) GetExpressionsByUserID(userID int64) ([]models.Expression, error) {
	r.mx.RLock()
	defer r.mx.RUnlock()

	query := `SELECT id, user_id, expression, status, result, steps, created_at, updated_at
	         FROM expressions WHERE user_id = ? ORDER BY created_at DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения списка выражений для пользователя ID %d: %w", userID, err)
	}
	defer rows.Close()

	expressions := make([]models.Expression, 0)
	for rows.Next() {
		expr := models.Expression{}
		if err = rows.Scan(
			&expr.ID, &expr.UserID, &expr.Expression, &expr.Status,
			&expr.Result, &expr.Steps, &expr.CreatedAt, &expr.UpdatedAt,
		); err != nil {
			log.Printf("error while scanning expressions. Err: %v", err)
			continue
		}
		expressions = append(expressions, expr)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("occurred error while iterarting expressions: %w", err)
	}

	return expressions, nil
}

func (r *repo) UpdateExpressionStatusResult(id int64, status string, result sql.NullFloat64, stepsJSON sql.NullString) error {
	r.mx.Lock()
	defer r.mx.Unlock()

	query := `UPDATE expressions SET status = ?, result = ?, steps = ?, updated_at = CURRENT_TIMESTAMP
	         WHERE id = ?`
	_, err := r.db.Exec(query, status, result, stepsJSON, id)
	if err != nil {
		return fmt.Errorf("can't update expression. Id: %d. Err: %v", id, err)
	}

	return nil
}

func (r *repo) CreateTask(expressionID int64, operation string, arg1, arg2 float64) (int64, error) {
	r.mx.Lock()
	defer r.mx.Unlock()

	query := `INSERT INTO tasks (expression_id, operation, arg1, arg2, status) VALUES (?, ?, ?, ?, ?)`
	res, err := r.db.Exec(query, expressionID, operation, arg1, arg2, constants.StatusPending)
	if err != nil {
		return 0, fmt.Errorf("can't create task. Id:%d. Err:%v", expressionID, err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("can't get task ID: %v", err)
	}

	return id, nil
}

func (r *repo) GetAndLeasePendingTask() (*models.Task, error) {
	r.mx.Lock()
	defer r.mx.Unlock()

	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("can't run transaction. Err: %v", err)
	}
	defer func() {
		if rec := recover(); rec != nil {
			_ = tx.Rollback()
			panic(r)
		} else if err != nil {
			_ = tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				log.Printf("can't commit changes. Err: %v", err)
			}
		}
	}()

	querySelect := `SELECT id, expression_id, operation, arg1, arg2, status, retries, created_at, updated_at
	                FROM tasks WHERE status = ? ORDER BY created_at ASC LIMIT 1`
	row := tx.QueryRow(querySelect, constants.StatusPending)

	task := new(models.Task)
	if err = row.Scan(
		&task.ID, &task.ExpressionID, &task.Operation, &task.Arg1, &task.Arg2,
		&task.Status, &task.Retries, &task.CreatedAt, &task.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("can't fetch task.Err: %v", err)
	}

	queryUpdate := `UPDATE tasks SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err = tx.Exec(queryUpdate, constants.StatusInProgress, task.ID)
	if err != nil {
		return nil, fmt.Errorf("can't update task status.TaskId: %d. Err: %v", task.ID, err)
	}

	task.Status = constants.StatusInProgress
	return task, nil
}

func (r *repo) CompleteTask(taskID int64, result float64) error {
	r.mx.Lock()
	defer r.mx.Unlock()

	query := `UPDATE tasks SET status = ?, result = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status = ?`
	res, err := r.db.Exec(query, constants.StatusDone, result, taskID, constants.StatusInProgress)
	if err != nil {
		return fmt.Errorf("can't finish task. TaskId: %d. Err: %v", taskID, err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		log.Printf("can't update task status. TaskId: %d", taskID)
	}

	return nil
}

func (r *repo) FailTask(taskID int64) error {
	r.mx.Lock()
	defer r.mx.Unlock()

	query := `UPDATE tasks SET status = ?, retries = retries + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND status = ?`
	res, err := r.db.Exec(query, constants.StatusPending, taskID, constants.StatusInProgress)
	if err != nil {
		return fmt.Errorf("occured error while update process. TaskId: %d. Err: %v", taskID, err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		log.Printf("operation failed. TaskId: %d", taskID)
	}
	return nil
}

func (r *repo) GetTaskByID(taskID int64) (*models.Task, error) {
	r.mx.RLock()
	defer r.mx.RUnlock()

	query := `SELECT id, expression_id, operation, arg1, arg2, result, status, retries, created_at, updated_at
	         FROM tasks WHERE id = ?`
	row := r.db.QueryRow(query, taskID)

	task := new(models.Task)
	err := row.Scan(
		&task.ID, &task.ExpressionID, &task.Operation, &task.Arg1, &task.Arg2,
		&task.Result, &task.Status, &task.Retries, &task.CreatedAt, &task.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("can't get task by id. TaskId: %d. Err: %v", taskID, err)
	}
	return task, nil
}

func (r *repo) HasPendingTasks(expressionID int64) (bool, error) {
	r.mx.RLock()
	defer r.mx.RUnlock()

	query := `SELECT 1 FROM tasks WHERE expression_id = ? AND status IN (?, ?) LIMIT 1`
	var exists int
	if err := r.db.QueryRow(query, expressionID, constants.StatusPending, constants.StatusInProgress).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("occured error. ExpressionId: %d. Err: %v", expressionID, err)
	}
	return true, nil
}

func (r *repo) GetExpressionByIDInternal(id int64) (*models.Expression, error) {
	r.mx.RLock()
	defer r.mx.RUnlock()

	query := `SELECT id, user_id, expression, status, result, steps, created_at, updated_at
	         FROM expressions WHERE id = ?`
	row := r.db.QueryRow(query, id)

	expr := new(models.Expression)
	err := row.Scan(
		&expr.ID, &expr.UserID, &expr.Expression, &expr.Status,
		&expr.Result, &expr.Steps, &expr.CreatedAt, &expr.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("occured error. ExpressionId: %d. Err: %v", id, err)
	}
	return expr, nil
}

func (r *repo) GetAllTasksForExpression(expressionID int64) ([]models.Task, error) {
	r.mx.RLock()
	defer r.mx.RUnlock()

	query := `SELECT id, expression_id, operation, arg1, arg2, result, status, retries, created_at, updated_at
		FROM tasks WHERE expression_id = ?`
	rows, err := r.db.Query(query, expressionID)
	if err != nil {
		return nil, fmt.Errorf("occured error. ExpressionId: %d. Err: %v", expressionID, err)
	}
	defer rows.Close()

	tasks := make([]models.Task, 0)
	for rows.Next() {
		var task models.Task
		if err := rows.Scan(
			&task.ID, &task.ExpressionID, &task.Operation,
			&task.Arg1, &task.Arg2, &task.Result,
			&task.Status, &task.Retries, &task.CreatedAt, &task.UpdatedAt,
		); err != nil {
			log.Printf("can't scan err: %v", err)
			continue
		}
		tasks = append(tasks, task)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("occured error: %v", err)
	}

	return tasks, nil
}
