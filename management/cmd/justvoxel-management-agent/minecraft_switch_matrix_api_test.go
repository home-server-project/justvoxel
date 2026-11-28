package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func matrixAPIRequest(t *testing.T, mux *http.ServeMux, method, suffix, body string, want int) switchMatrixAPIResponse {
	t.Helper()
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, authorizedRequest(method, switchMatrixAPIPath+suffix, body))
	if w.Code != want {
		t.Fatalf("%s %s: got %d, want %d: %s", method, suffix, w.Code, want, w.Body)
	}
	var status switchMatrixAPIResponse
	if want == http.StatusOK || want == http.StatusAccepted {
		if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
			t.Fatal(err)
		}
		for _, private := range []string{"matrix_fixtures", "instance_id", "source_release", "target_release", "secret-example", "config_path", "plan_fingerprint", "rcon_password"} {
			if strings.Contains(w.Body.String(), private) {
				t.Fatalf("private evidence crossed the API: %s", private)
			}
		}
	}
	return status
}

func TestSwitchMatrixDevelopmentGate(t *testing.T) {
	s := adminServerForTest()
	f := newInstanceActivationFixture(t)
	s.operations = matrixTestStore(t, f.backend)
	mux := http.NewServeMux()
	registerSwitchMatrixRoutes(mux, s)
	if got := matrixAPIRequest(t, mux, "GET", "", "", 200); got.Enabled != switchMatrixDevelopment || got.Operation != nil {
		t.Fatalf("wrong capability or initial status: %+v", got)
	}
	for _, query := range []string{"?operation_id=invalid", "?operation_id=", "?server=jv-source1", "?operation_id=%zz"} {
		matrixAPIRequest(t, mux, "GET", query, "", 400)
	}
	unauthenticated := httptest.NewRecorder()
	mux.ServeHTTP(unauthenticated, httptest.NewRequest("GET", switchMatrixAPIPath, nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("status accepted without a session: %d", unauthenticated.Code)
	}

	for _, role := range []principalRole{roleOperator, roleViewer} {
		user := roleServerForTest(role)
		roleMux := http.NewServeMux()
		registerSwitchMatrixRoutes(roleMux, user)
		matrixAPIRequest(t, roleMux, "GET", "", "", 403)
		for _, action := range []string{"/start", "/stop", "/recover"} {
			want := http.StatusNotFound
			if switchMatrixDevelopment {
				want = http.StatusForbidden
			}
			matrixAPIRequest(t, roleMux, "POST", action, `{}`, want)
		}
	}
	if switchMatrixDevelopment {
		matrixAPIRequest(t, mux, "POST", "/start", `{"server":"jv-source1"}`, 400)
		for _, action := range []string{"/stop", "/recover"} {
			matrixAPIRequest(t, mux, "POST", action, `{"operation_id":"invalid"}`, 400)
		}
	} else {
		for _, action := range []string{"/start", "/stop", "/recover"} {
			matrixAPIRequest(t, mux, "POST", action, `{}`, 404)
		}
		if current, err := s.operations.currentMinecraftMaintenance(); err != nil || current != nil {
			t.Fatalf("production request admitted matrix maintenance: %+v %v", current, err)
		}
	}

	// Reopen a persisted development reservation in either build. Production
	// keeps the same journal and lock, with no recovery or restart route.
	j, err := s.operations.beginSwitchMatrix(switchMatrixRunnerFingerprint())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.operations.saveSwitchMatrixRun(switchMatrixRun{Schema: 1, ID: j.OperationID, State: "preparing"}); err != nil {
		t.Fatal(err)
	}
	if err := s.operations.close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := openOperationStore(s.operations.baseDir)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	s.operations = reopened
	got := matrixAPIRequest(t, mux, "GET", "", "", 200).Operation
	if got == nil || got.OperationID != j.OperationID || got.State != "interrupted" ||
		!got.ReservationActive || !got.RecoveryRequired {
		t.Fatalf("restart lost its maintenance reservation: %+v", got)
	}
	if !switchMatrixDevelopment {
		matrixAPIRequest(t, mux, "POST", "/start", `{"case_ids":["fabric.leaf.clone.26_1_plus.java.running"]}`, 404)
		matrixAPIRequest(t, mux, "POST", "/recover", `{"operation_id":"`+j.OperationID+`"}`, 404)
	}
}

func TestSwitchMatrixImagePackaging(t *testing.T) {
	root := "../../.."
	// The validation container builds from /work and mounts image sources at /repo.
	if _, err := os.Stat(filepath.Join(root, "Containerfile")); os.IsNotExist(err) {
		root = "/repo"
	}
	read := func(path string) string {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	product := read("Containerfile")
	if strings.Contains(product, "management-agent-development") || strings.Contains(product, "COPY build_artifacts /") ||
		!strings.Contains(product, "COPY build_artifacts/management/management-agent /build_artifacts/management/management-agent\n") ||
		!strings.Contains(product, "build-common.sh false") {
		t.Fatal("VM/HWS assembly must install and verify only the production artifact")
	}
	if !strings.Contains(read("Containerfile.dev"), "management-agent-development /build_artifacts/management/management-agent") ||
		!strings.Contains(read("Containerfile.dev"), "build-common.sh true") ||
		!strings.Contains(read("build_files/build-management-agent.sh"), "-tags justvoxel_dev -o /out/management-agent-development") ||
		!strings.Contains(read("jv-build.sh"), "build-management-agent.sh development") {
		t.Fatal("development artifacts must be built and verified separately")
	}
	if !strings.Contains(read("build_files/build-common.sh"), `grep -Fxq "Switch matrix=${expected_switch_matrix}"`) {
		t.Fatal("image assembly must verify the compiled agent capability")
	}
	for _, name := range []string{"runner", "runtime", "development", "selection"} {
		if !strings.HasPrefix(read("management/cmd/justvoxel-management-agent/minecraft_switch_matrix_"+name+".go"), "//go:build justvoxel_dev\n") {
			t.Fatalf("executable matrix %s is in the production build", name)
		}
	}
	entries, err := filepath.Glob(filepath.Join(root, "mjust/libexec/*matrix*"))
	if err != nil || len(entries) != 1 || filepath.Base(entries[0]) != "switch-matrix-reservation.sh" {
		t.Fatalf("common helper directory includes a possible matrix runner: %v %v", entries, err)
	}
}
