package orchestrator

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/atadzan/dist-arith-go/internal/constants"
	"github.com/atadzan/dist-arith-go/internal/models"
	"github.com/atadzan/dist-arith-go/internal/repository"
)

type OperationTimes struct {
	Addition       int
	Subtraction    int
	Multiplication int
	Division       int
}

type Scheduler struct {
	repo    repository.Repository
	opTimes *OperationTimes
}

func NewScheduler(db repository.Repository) *Scheduler {
	return &Scheduler{
		repo:    db,
		opTimes: initOperationTimes(),
	}
}

func (s *Scheduler) ScheduleTasks(expressionID int64, expression string) error {
	parser := NewParser(expression)
	ast, err := parser.Parse()
	if err != nil {
		errMsg := fmt.Sprintf("parse error: %v", err)
		s.repo.UpdateExpressionStatusResult(expressionID, constants.StatusError, sql.NullFloat64{}, sql.NullString{String: errMsg, Valid: true})
		return fmt.Errorf("parse error, expression ID %d: %w", expressionID, err)
	}

	err = s.planTasksRecursive(ast, expressionID)
	if err != nil {
		errMsg := fmt.Sprintf("occured error: %v", err)
		s.repo.UpdateExpressionStatusResult(expressionID, constants.StatusError, sql.NullFloat64{}, sql.NullString{String: errMsg, Valid: true})
		return fmt.Errorf("occured error, expression ID %d: %w", expressionID, err)
	}

	if ast.Value == nil {
		err = s.repo.UpdateExpressionStatusResult(expressionID, constants.StatusInProgress, sql.NullFloat64{}, sql.NullString{})
		if err != nil {
			log.Printf("occured error, expression ID %d: %v", expressionID, err)
		}
	} else {
		log.Printf("Expression ID %d. Value (%f)", expressionID, *ast.Value)
		stepsJSON, _ := json.Marshal([]string{fmt.Sprintf("Result: %f", *ast.Value)})
		err = s.repo.UpdateExpressionStatusResult(expressionID,
			constants.StatusDone,
			sql.NullFloat64{Float64: *ast.Value, Valid: true},
			sql.NullString{String: string(stepsJSON), Valid: true},
		)
		if err != nil {
			log.Printf("can't update status to done для числового выражения ID %d: %v", expressionID, err)
		}
	}

	return nil
}

func (s *Scheduler) planTasksRecursive(node *Node, expressionID int64) error {
	if node == nil || node.Value != nil { // Базовый случай: лист (число) или пустой узел
		return nil
	}

	if err := s.planTasksRecursive(node.Left, expressionID); err != nil {
		return err
	}
	if err := s.planTasksRecursive(node.Right, expressionID); err != nil {
		return err
	}

	leftReady := node.Left != nil && node.Left.Value != nil
	rightReady := node.Right != nil && node.Right.Value != nil

	if leftReady && rightReady {
		_, err := s.repo.CreateTask(
			expressionID,
			node.Op,
			*node.Left.Value,
			*node.Right.Value,
		)
		if err != nil {
			return fmt.Errorf("occured err '%s' expression ID:%d, err: %v", node.Op, expressionID, err)
		}
	}

	return nil
}

func (s *Scheduler) GetOperationTimes() *OperationTimes {
	return s.opTimes
}

func fillASTValues(node *Node, doneTasks []models.Task) {
	if node == nil {
		return
	}
	if node.Value != nil {
		// Значение уже известно (либо лист, либо ранее заполненный узел)
		return
	}
	fillASTValues(node.Left, doneTasks)
	fillASTValues(node.Right, doneTasks)
	if node.Op != "" && node.Left != nil && node.Left.Value != nil && node.Right != nil && node.Right.Value != nil {
		for _, t := range doneTasks {
			if t.Operation == node.Op && t.Arg1 == *node.Left.Value && t.Arg2 == *node.Right.Value {
				val := t.Result.Float64
				node.Value = &val
				break
			}
		}
	}
}

func (s *Scheduler) ProcessTaskCompletion(taskID int64) {
	log.Printf("Scheduler: Processing task ID %d", taskID)

	task, err := s.repo.GetTaskByID(taskID)
	if err != nil {
		log.Printf("Scheduler: error can't get task %d from db: %v", taskID, err)
		return
	}
	if task == nil {
		log.Printf("Scheduler: Task ID %d not found", taskID)
		return
	}

	expr, err := s.repo.GetExpressionByIDInternal(task.ExpressionID)
	if err != nil {
		log.Printf("Scheduler: fetch error of expression %d. Err: %v", task.ExpressionID, err)
		return
	}
	if expr == nil {
		log.Printf("Scheduler: Expression ID %d for task ID %d not found", task.ExpressionID, taskID)
		return
	}

	parser := NewParser(expr.Expression)
	ast, err := parser.Parse()
	if err != nil {
		errMsg := fmt.Sprintf("parsing error. TaskId: %d, err: %v", taskID, err)
		log.Printf("Scheduler: %s", errMsg)
		s.repo.UpdateExpressionStatusResult(expr.ID,
			constants.StatusError,
			sql.NullFloat64{},
			sql.NullString{String: errMsg, Valid: true},
		)
		return
	}

	allTasks, err := s.repo.GetAllTasksForExpression(expr.ID)
	if err != nil {
		log.Printf("Scheduler: error fetching expressions. ID:%d, err:%v", expr.ID, err)
	}
	doneTasks := make([]models.Task, 0)
	for _, t := range allTasks {
		if t.Status == constants.StatusDone {
			doneTasks = append(doneTasks, t)
		}
	}

	fillASTValues(ast, doneTasks)

	err = s.planTasksRecursive(ast, expr.ID)
	if err != nil {
		log.Printf("Scheduler: Occured error expression ID %d: %v", expr.ID, err)
		return
	}

	if ast.Value != nil {
		result := *ast.Value
		s.repo.UpdateExpressionStatusResult(expr.ID,
			constants.StatusDone,
			sql.NullFloat64{Float64: result, Valid: true},
			sql.NullString{},
		)
		log.Printf("Scheduler: Expression ID %d result %f.", expr.ID, result)
	} else {
		s.repo.UpdateExpressionStatusResult(expr.ID,
			constants.StatusInProgress,
			sql.NullFloat64{},
			sql.NullString{},
		)
	}
}

func initOperationTimes() *OperationTimes {
	return &OperationTimes{
		Addition:       readTimeEnv("TIME_ADDITION_MS", 1000),
		Subtraction:    readTimeEnv("TIME_SUBTRACTION_MS", 1000),
		Multiplication: readTimeEnv("TIME_MULTIPLICATION_MS", 1000),
		Division:       readTimeEnv("TIME_DIVISION_MS", 1000),
	}
}

func readTimeEnv(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if t, err := strconv.Atoi(v); err == nil && t >= 0 {
			return t
		} else {
			log.Println(err)
		}
	}
	return defaultValue
}
