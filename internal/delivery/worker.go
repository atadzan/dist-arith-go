package delivery

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type Worker struct {
	orchestratorAddress string
}

func NewWorkerHandler(orchestratorAddress string) Worker {
	return Worker{
		orchestratorAddress: orchestratorAddress,
	}
}

func (w *Worker) Run() {
	ticker := time.NewTicker(time.Minute)
	for _ = range ticker.C {
		task, err := w.GetTask()
		if err != nil {
			log.Printf("can't get task. Err: %v", err)
			continue
		}
		// send result to orchestrator
		result, err := w.Calculate(task)
		if err != nil {
			log.Printf("can't calculate expression. Err :%v", err)
			continue
		}
		if err = w.SendResult(task.Id, result); err != nil {
			log.Printf("can't send task result. Err: %v", err)
			continue
		}
	}
}

func (w *Worker) GetTask() (task Task, err error) {
	resp, err := sendGETRequest(fmt.Sprintf("%s/internal/task", w.orchestratorAddress))
	if err != nil {
		return
	}
	if err = json.Unmarshal(resp, &task); err != nil {
		return
	}
	return
}

func (w *Worker) Calculate(task Task) (float64, error) {
	return calculate(task.Expression)
}

func (w *Worker) SendResult(id string, result float64) error {
	reqBody := SendResult{Id: id, Result: result}
	rawBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("can't marshal send request body. Err: %v", err)
		return err
	}
	_, err = sendPOSTRequest(fmt.Sprintf("%s/internal/task", w.orchestratorAddress), rawBody)
	if err != nil {
		log.Printf("can't send task result. Err: %v", err)
		return err
	}
	return nil
}
