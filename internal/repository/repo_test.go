package repository

import (
	"strings"
	"testing"

	"github.com/atadzan/dist-arith-go/internal/constants"
	"github.com/atadzan/dist-arith-go/pkg/database"
)

func TestUserAndExpressionCRUD(t *testing.T) {
	testingDb, err := database.GetTestingDBConn()
	if err != nil {
		t.Fatalf("can't establish db connection")
	}
	defer testingDb.Close()
	repo, err := New(testingDb)
	if err != nil {
		if strings.Contains(err.Error(), "requires cgo") {
			t.Skipf("skip DB tests: %v", err)
		}
		t.Fatalf("New error: %v", err)
	}
	if err = repo.CreateTables(); err != nil {
		if strings.Contains(err.Error(), "requires cgo") {
			t.Skipf("skip DB tests: %v", err)
		}
		t.Fatalf("InitDB error: %v", err)
	}

	uid, err := repo.CreateUser("testuser", "hashpass")
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	user, err := repo.GetUserByLogin("testuser")
	if err != nil {
		t.Fatalf("GetUserByLogin error: %v", err)
	}
	if user == nil || user.ID != uid {
		t.Fatalf("GetUserByLogin returned wrong user: %+v", user)
	}

	exprID, err := repo.CreateExpression(uid, "1+1")
	if err != nil {
		t.Fatalf("CreateExpression error: %v", err)
	}
	expr, err := repo.GetExpressionByID(exprID, uid)
	if err != nil {
		t.Fatalf("GetExpressionByID error: %v", err)
	}
	if expr == nil || expr.Expression != "1+1" || expr.Status != constants.StatusPending {
		t.Fatalf("GetExpressionByID returned wrong expression: %+v", expr)
	}

	list, err := repo.GetExpressionsByUserID(uid)
	if err != nil {
		t.Fatalf("GetExpressionsByUserID error: %v", err)
	}
	if len(list) != 1 || list[0].ID != exprID {
		t.Fatalf("GetExpressionsByUserID returned wrong list: %+v", list)
	}
}

func TestTaskLifecycle(t *testing.T) {
	testingDb, err := database.GetTestingDBConn()
	if err != nil {
		t.Fatalf("can't establish db connection")
	}
	defer testingDb.Close()
	repo, err := New(testingDb)
	if err != nil {
		if strings.Contains(err.Error(), "requires cgo") {
			t.Skipf("skip DB tests: %v", err)
		}
		t.Fatalf("New error: %v", err)
	}
	if err = repo.CreateTables(); err != nil {
		if strings.Contains(err.Error(), "requires cgo") {
			t.Skipf("skip DB tests: %v", err)
		}
		t.Fatalf("InitDB error: %v", err)
	}

	uid, err := repo.CreateUser("u2", "h2")
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	exprID, err := repo.CreateExpression(uid, "2*3")
	if err != nil {
		t.Fatalf("CreateExpression error: %v", err)
	}

	tid, err := repo.CreateTask(exprID, "*", 2, 3)
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	task, err := repo.GetAndLeasePendingTask()
	if err != nil {
		t.Fatalf("GetAndLeasePendingTask error: %v", err)
	}
	if task == nil || task.ID != tid || task.Status != constants.StatusInProgress {
		t.Fatalf("GetAndLeasePendingTask returned wrong: %+v", task)
	}

	if err = repo.CompleteTask(tid, 6); err != nil {
		t.Fatalf("CompleteTask error: %v", err)
	}
	t2, err := repo.GetTaskByID(tid)
	if err != nil {
		t.Fatalf("GetTaskByID error: %v", err)
	}
	if t2.Status != constants.StatusDone || !t2.Result.Valid || t2.Result.Float64 != 6 {
		t.Fatalf("GetTaskByID after complete wrong: %+v", t2)
	}

	tid2, err := repo.CreateTask(exprID, "+", 1, 1)
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	task2, _ := repo.GetAndLeasePendingTask()
	if task2 == nil {
		t.Fatalf("Expected task2 leased, got nil")
	}
	if err := repo.FailTask(tid2); err != nil {
		t.Fatalf("FailTask error: %v", err)
	}
	t3, _ := repo.GetTaskByID(tid2)
	if t3.Status != constants.StatusPending || t3.Retries != 1 {
		t.Fatalf("FailTask not applied: %+v", t3)
	}

	has, err := repo.HasPendingTasks(exprID)
	if err != nil {
		t.Fatalf("HasPendingTasks error: %v", err)
	}
	if !has {
		t.Fatalf("HasPendingTasks returned false, expected true")
	}

	has2, _ := repo.HasPendingTasks(9999)
	if has2 {
		t.Fatalf("HasPendingTasks for unknown expr should be false")
	}
}
