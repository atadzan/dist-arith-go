package orchestrator

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/atadzan/dist-arith-go/internal/repository"
	db "github.com/atadzan/dist-arith-go/pkg/database"

	pb "github.com/atadzan/dist-arith-go/internal/worker/grpc/calc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func dialer() (*grpc.ClientConn, func(), error) {
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	dbConn, err := db.NewDBConn(":memory:")
	if err != nil {
		return nil, nil, err
	}
	repo, err := repository.New(dbConn)
	if err != nil {
		if strings.Contains(err.Error(), "requires cgo") {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	if _, err := repo.GetAndLeasePendingTask(); err != nil {
		if strings.Contains(err.Error(), "requires cgo") {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	scheduler := NewScheduler(repo)
	pb.RegisterCalcWorkerServiceServer(srv, NewCalculatorGRPCServer(repo, scheduler.GetOperationTimes(), scheduler))
	go srv.Serve(lis)

	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithInsecure())
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { conn.Close(); srv.Stop() }
	return conn, cleanup, nil
}

func TestGetTask_NoTask(t *testing.T) {
	conn, cleanup, err := dialer()
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Skip("skip gRPC tests: cgo disabled or in-memory DB not available")
	}
	defer cleanup()

	client := pb.NewCalcWorkerServiceClient(conn)
	resp, err := client.GetTask(context.Background(), &pb.GetTaskRequest{WorkerId: "test"})
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if _, ok := resp.TaskInfo.(*pb.GetTaskResponse_NoTask); !ok {
		t.Errorf("expected NoTaskAvailable, got %T", resp.TaskInfo)
	}
}
