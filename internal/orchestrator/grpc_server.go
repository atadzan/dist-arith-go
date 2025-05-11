package orchestrator

import (
	"context"
	"log"

	"github.com/atadzan/dist-arith-go/internal/repository"

	pb "github.com/atadzan/dist-arith-go/internal/worker/grpc/calc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type grpcServer struct {
	pb.UnimplementedCalcWorkerServiceServer
	repo      repository.Repository
	opTimes   *OperationTimes
	scheduler *Scheduler
}

func NewCalculatorGRPCServer(repo repository.Repository, opTimes *OperationTimes, scheduler *Scheduler) *grpcServer {
	return &grpcServer{
		repo:      repo,
		opTimes:   opTimes,
		scheduler: scheduler,
	}
}

func (s *grpcServer) GetTask(ctx context.Context, req *pb.GetTaskRequest) (*pb.GetTaskResponse, error) {
	log.Printf("gRPC: Get task from worker: %s", req.GetWorkerId())

	task, err := s.repo.GetAndLeasePendingTask()
	if err != nil {
		log.Printf("gRPC: can't get tasks from DB: %v", err)
		return nil, status.Errorf(codes.Internal, "task fetch error: %v", err)
	}

	if task == nil {
		log.Println("gRPC: no available tasks for worker")
		return &pb.GetTaskResponse{
			TaskInfo: &pb.GetTaskResponse_NoTask{
				NoTask: &pb.NoTaskAvailable{
					RetryAfterSeconds: 5,
				},
			},
		}, nil
	}

	log.Printf("gRPC: Sending task %d to worker %s", task.ID, req.GetWorkerId())
	return &pb.GetTaskResponse{
		TaskInfo: &pb.GetTaskResponse_Task{
			Task: &pb.Task{
				Id:              task.ID,
				Arg1:            task.Arg1,
				Arg2:            task.Arg2,
				Operation:       task.Operation,
				OperationTimeMs: s.getOperationTimeMs(task.Operation),
			},
		},
	}, nil
}

func (s *grpcServer) SubmitResult(ctx context.Context, req *pb.SubmitResultRequest) (*pb.SubmitResultResponse, error) {
	log.Printf("gRPC: Received SubmitResult for task %d from worker: %s", req.TaskId, req.GetWorkerId())
	var taskErr error

	switch result := req.ResultStatus.(type) {
	case *pb.SubmitResultRequest_Result:
		taskErr = s.repo.CompleteTask(req.TaskId, result.Result)
		if taskErr == nil {
			log.Printf("gRPC: Task id %d completed in database", req.TaskId)
		} else {
			log.Printf("gRPC: occured error. TaskId %d, err: %v", req.TaskId, taskErr)
		}
	case *pb.SubmitResultRequest_Error:
		log.Printf("gRPC: TaskId %d finished with err: %s", req.TaskId, result.Error.Message)
		taskErr = s.repo.FailTask(req.TaskId)
		if taskErr != nil {
			log.Printf("gRPC: occured error taskId %d, err: %v", req.TaskId, taskErr)
		}
	default:
		log.Printf("gRPC: invalid task status %d", req.TaskId)
		return nil, status.Error(codes.InvalidArgument, "invalid task status")
	}

	if taskErr != nil {
		return nil, status.Errorf(codes.Internal, "occurred error: %v", taskErr)
	}

	go s.scheduler.ProcessTaskCompletion(req.TaskId)

	return &pb.SubmitResultResponse{Acknowledged: true}, nil
}

func (s *grpcServer) getOperationTimeMs(op string) int32 {
	var t int
	switch op {
	case "+":
		t = s.opTimes.Addition
	case "-":
		t = s.opTimes.Subtraction
	case "*":
		t = s.opTimes.Multiplication
	case "/":
		t = s.opTimes.Division
	default:
		t = 1000
	}
	return int32(t)
}
