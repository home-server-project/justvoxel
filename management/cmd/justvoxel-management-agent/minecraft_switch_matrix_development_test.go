//go:build justvoxel_dev

package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestSwitchMatrixDevelopmentAPI(t *testing.T) {
	t.Run("persistent launch and reconnect", func(t *testing.T) {
		s := adminServerForTest()
		f := newInstanceActivationFixture(t)
		o := matrixTestStore(t, f.backend)
		s.operations = o
		mux := http.NewServeMux()
		registerSwitchMatrixRoutes(mux, s)
		release := make(chan struct{})
		discovery := make(chan struct{}, 1)
		oldCommand := f.backend.command
		f.backend.command = func(ctx context.Context, command string, args ...string) ([]byte, error) {
			if command == switchMatrixVersionHelper && len(args) == 1 && args[0] == "matrix-catalog" {
				discovery <- struct{}{}
				select {
				case <-release:
					return []byte("[]"), nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			}
			return nil, fmt.Errorf("unexpected test command %s", command)
		}
		t.Cleanup(func() {
			close(release)
			o.switchMatrixWorkerMu.Lock()
			defer o.switchMatrixWorkerMu.Unlock()
			f.backend.command = oldCommand
		})
		started := matrixAPIRequest(t, mux, "POST", "/start", "{}", 202).Operation
		if started == nil || !validOperationID(started.OperationID) {
			t.Fatalf("launch lacks durable identity: %+v", started)
		}
		select {
		case <-discovery:
		case <-time.After(5 * time.Second):
			t.Fatal("admitted worker did not start discovery")
		}
		matrixAPIRequest(t, mux, "POST", "/start", "{}", 409)
		for _, suffix := range []string{"", "?operation_id=" + started.OperationID} {
			got := matrixAPIRequest(t, mux, "GET", suffix, "", 200).Operation
			if got == nil || got.OperationID != started.OperationID || got.State != "preparing" ||
				got.Progress == nil || got.Progress.ManifestReady || !got.ReservationActive {
				t.Fatalf("status did not reconnect to discovery: %+v", got)
			}
		}
	})

	t.Run("stop waits for admitted case", func(t *testing.T) {
		s := adminServerForTest()
		worker, j, run := newMatrixRunnerTest(t, 3)
		s.operations = worker.operations
		// Retain private release information in the downloadable manifest only.
		runtime := *run.Cases[0].SourceRelease.Runtime
		runtime.Digest = "secret-example"
		run.Cases[0].SourceRelease.Runtime = &runtime
		if err := s.operations.saveSwitchMatrixRun(run); err != nil {
			t.Fatal(err)
		}
		mux := http.NewServeMux()
		registerSwitchMatrixRoutes(mux, s)
		cases := 0
		err := s.operations.runSwitchMatrixCases(j.OperationID, func(c switchMatrixCase) (switchMatrixCaseResult, error) {
			cases++
			got := matrixAPIRequest(t, mux, "POST", "/stop", `{"operation_id":"`+j.OperationID+`"}`, 202).Operation
			if got == nil || got.Progress == nil || !got.Progress.CancelRequested || got.State != "running" ||
				got.Progress.Source != c.Source || got.Progress.Action != c.Action || !got.ReservationActive {
				t.Fatalf("stop did not retain the admitted case: %+v", got)
			}
			return matrixRunnerPassed(c), nil
		})
		got := matrixAPIRequest(t, mux, "GET", "", "", 200).Operation
		if err != nil || cases != 1 || got == nil || got.Progress == nil || got.ReservationActive || got.State != "cancelled" ||
			got.Progress.Passed != 1 || got.Progress.Unfinished != 2 {
			t.Fatalf("invalid cancellation result: %d %+v %v", cases, got, err)
		}
	})

	t.Run("explicit recovery leaves interrupted work unfinished", func(t *testing.T) {
		s := adminServerForTest()
		worker, j, run := newMatrixRunnerTest(t, 2)
		o := worker.operations
		run.Results[0] = switchMatrixCaseResult{Outcome: matrixExecuting, StartedAt: time.Now()}
		if err := o.saveSwitchMatrixRun(run); err != nil {
			t.Fatal(err)
		}
		if err := o.close(); err != nil {
			t.Fatal(err)
		}
		reopened, err := openOperationStore(o.baseDir)
		if err != nil {
			t.Fatal(err)
		}
		defer reopened.close()
		s.operations = reopened
		mux := http.NewServeMux()
		registerSwitchMatrixRoutes(mux, s)
		matrixAPIRequest(t, mux, "POST", "/start", "{}", 409)
		got := matrixAPIRequest(t, mux, "POST", "/recover", `{"operation_id":"`+j.OperationID+`"}`, 200).Operation
		if got == nil || got.Progress == nil || got.State != "recovered" || got.ReservationActive || got.RecoveryRequired ||
			got.Progress.Passed != 0 || got.Progress.Unfinished != 2 {
			t.Fatalf("recovery lost or reran interrupted work: %+v", got)
		}
	})
}
