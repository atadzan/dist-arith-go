package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/atadzan/dist-arith-go/internal/worker/grpc/calc"
)

func Worker(workerID int, grpcClient pb.CalcWorkerServiceClient) {
	log.Printf("Worker %d run.", workerID)
	ctx := context.Background()
	workerId := fmt.Sprintf("worker-%d", workerID)

	for {
		log.Printf("Worker %d: Request for task...", workerID)
		var (
			task *pb.Task
			err  error
		)
		retryAfter := 1 * time.Second

		getTaskReq := &pb.GetTaskRequest{WorkerId: workerId}
		getTaskResp, err := grpcClient.GetTask(ctx, getTaskReq)
		if err != nil {
			log.Printf("Worker %d: can't get task: %v. Retry after %v...", workerID, err, retryAfter)
			time.Sleep(retryAfter)
			continue
		}

		switch taskInfo := getTaskResp.TaskInfo.(type) {
		case *pb.GetTaskResponse_Task:
			task = taskInfo.Task
			log.Printf("Worker %d: Received task %d: %f %s %f (time: %dms)",
				workerID, task.Id, task.Arg1, task.Operation, task.Arg2, task.OperationTimeMs)
		case *pb.GetTaskResponse_NoTask:
			if taskInfo.NoTask != nil && taskInfo.NoTask.RetryAfterSeconds > 0 {
				retryAfter = time.Duration(taskInfo.NoTask.RetryAfterSeconds) * time.Second
			}
			log.Printf("Worker %d: No available tasks. Retry after %v...", workerID, retryAfter)
			time.Sleep(retryAfter)
			continue
		default:
			log.Printf("Worker %d: Received unknown response. Retry after %v...", workerID, retryAfter)
			time.Sleep(retryAfter)
			continue
		}

		startTime := time.Now()
		result, computeErr := compute(task.Arg1, task.Arg2, task.Operation)
		computationDuration := time.Since(startTime)

		if task.OperationTimeMs > 0 {
			requiredDuration := time.Duration(task.OperationTimeMs) * time.Millisecond
			if computationDuration < requiredDuration {
				time.Sleep(requiredDuration - computationDuration)
			}
		}

		submitReq := &pb.SubmitResultRequest{
			TaskId:   task.Id,
			WorkerId: workerId,
		}
		if computeErr != nil {
			log.Printf("Worker %d: can't calculate task %d: %v", workerID, task.Id, computeErr)
			submitReq.ResultStatus = &pb.SubmitResultRequest_Error{
				Error: &pb.TaskError{Message: computeErr.Error()},
			}
		} else {
			log.Printf("Worker %d: Calculated task %d. Result: %f", workerID, task.Id, result)
			submitReq.ResultStatus = &pb.SubmitResultRequest_Result{Result: result}
		}

		_, err = grpcClient.SubmitResult(ctx, submitReq)
		if err != nil {
			log.Printf("Worker %d: occured error taskId:%d. Err: %v.", workerID, task.Id, err)
			time.Sleep(retryAfter)
		} else {
			log.Printf("Worker %d: task result %d sent.", workerID, task.Id)
		}

	}
}

func compute(arg1, arg2 float64, op string) (float64, error) {
	switch op {
	case "+":
		return arg1 + arg2, nil
	case "-":
		return arg1 - arg2, nil
	case "*":
		return arg1 * arg2, nil
	case "/":
		if arg2 == 0 {
			return 0, fmt.Errorf("division to zero")
		}
		return arg1 / arg2, nil
	default:
		return 0, fmt.Errorf("unkown operation: %s", op)
	}
}
