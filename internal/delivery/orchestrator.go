package delivery

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type OrchestratorHandler struct {
	Handler *http.ServeMux
	storage map[string]Task
	mx      *sync.Mutex
}

func NewOrchestratorHandler() OrchestratorHandler {
	return OrchestratorHandler{
		Handler: http.NewServeMux(),
		storage: make(map[string]Task),
		mx:      new(sync.Mutex),
	}
}

func (o *OrchestratorHandler) InitRoutes() {
	o.Handler.HandleFunc("POST /api/v1/calculate", o.Calculate)
	o.Handler.HandleFunc("GET /api/v1/expressions", o.ListExpressions)
	o.Handler.HandleFunc("GET /api/v1/expressions/{id}", o.GetExpressionById)
	o.Handler.HandleFunc("GET /internal/task", o.ReceiveTask)
	o.Handler.HandleFunc("POST /internal/task", o.SendTaskResult)
}

func (o *OrchestratorHandler) save(expression string) (uniqueId string) {
	id := uuid.New()
	o.mx.Lock()
	o.storage[id.String()] = Task{
		Id:         id.String(),
		Status:     "pending",
		Result:     0,
		Expression: expression,
	}
	o.mx.Unlock()
	return id.String()
}

func (o *OrchestratorHandler) Calculate(w http.ResponseWriter, r *http.Request) {
	var input inputExpression
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		newErrorResp(w, http.StatusUnprocessableEntity, errInvalidInputParams)
		return
	}
	defer r.Body.Close()
	if err = json.Unmarshal(reqBody, &input); err != nil {
		newErrorResp(w, http.StatusUnprocessableEntity, errInvalidInputParams)
		return
	}
	if len(input.Expression) == 0 {
		newErrorResp(w, http.StatusUnprocessableEntity, errInvalidInputParams)
		return
	}
	uniqueId := o.save(input.Expression)
	w.Write([]byte(fmt.Sprintf("{\"id\": \"%s\"}", uniqueId)))
	w.WriteHeader(http.StatusCreated)
}

func (o *OrchestratorHandler) ListExpressions(w http.ResponseWriter, r *http.Request) {
	o.mx.Lock()
	var result ListExpressions
	for _, task := range o.storage {
		result.Expressions = append(result.Expressions, ListExpression{
			Id:     task.Id,
			Status: task.Status,
			Result: task.Result,
		})
	}
	o.mx.Unlock()
	if len(result.Expressions) == 0 {
		result.Expressions = []ListExpression{}
	}
	resp, err := json.Marshal(result)
	if err != nil {
		newErrorResp(w, http.StatusInternalServerError, errInternalServerMsg)
		return
	}
	w.Write(resp)
}

func (o *OrchestratorHandler) GetExpressionById(w http.ResponseWriter, r *http.Request) {
	taskId := r.PathValue("id")
	if len(taskId) == 0 {
		newErrorResp(w, http.StatusNotFound, errExpressionNotFound)
		return
	}
	o.mx.Lock()

	var result ListExpressionById
	task, ok := o.storage[taskId]
	o.mx.Unlock()
	if !ok {
		newErrorResp(w, http.StatusNotFound, errExpressionNotFound)
		return
	}
	result = ListExpressionById{Expression: ListExpression{
		Id:     task.Id,
		Status: task.Status,
		Result: task.Result,
	}}

	resp, err := json.Marshal(result)
	if err != nil {
		newErrorResp(w, http.StatusInternalServerError, errInternalServerMsg)
		return
	}
	w.Write(resp)
}

func (o *OrchestratorHandler) ReceiveTask(w http.ResponseWriter, r *http.Request) {
	var resp ReceiveExpression
	o.mx.Lock()
	for _, task := range o.storage {
		if task.Status == "pending" {
			resp.Task.Id = task.Id
			resp.Task.Expression = task.Expression
			resp.Task.OperationTime = 2 * time.Second
			break
		}
	}
	o.mx.Unlock()
	rawResp, err := json.Marshal(resp)
	if err != nil {
		newErrorResp(w, http.StatusInternalServerError, errInternalServerMsg)
		return
	}
	w.Write(rawResp)
}

func (o *OrchestratorHandler) SendTaskResult(w http.ResponseWriter, r *http.Request) {
	var input SendResult
	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		newErrorResp(w, http.StatusUnprocessableEntity, errInvalidInputParams)
		return
	}
	defer r.Body.Close()
	if err = json.Unmarshal(reqBody, &input); err != nil {
		newErrorResp(w, http.StatusUnprocessableEntity, errInvalidInputParams)
		return
	}
	o.mx.Lock()
	task, ok := o.storage[input.Id]
	o.mx.Unlock()
	if !ok {
		newErrorResp(w, http.StatusNotFound, errExpressionNotFound)
		return
	}
	task.Status = "finished"
	task.Result = input.Result
	o.mx.Lock()
	o.storage[input.Id] = task
	o.mx.Unlock()
}
