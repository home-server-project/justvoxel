package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type switchMatrixWebTestAPI struct {
	*fakeAPI
	role    string
	enabled bool
	calls   []string
}

func (f *switchMatrixWebTestAPI) Session(ctx context.Context, session string) (api.SessionInfo, error) {
	identity, err := f.fakeAPI.Session(ctx, session)
	identity.Role = f.role
	return identity, err
}

func (f *switchMatrixWebTestAPI) SwitchMatrixStatus(_ context.Context, _, id string) (api.SwitchMatrixStatus, error) {
	f.calls = append(f.calls, "status "+id)
	return api.SwitchMatrixStatus{Enabled: f.enabled}, nil
}

func (f *switchMatrixWebTestAPI) StartSwitchMatrix(context.Context, string) (api.SwitchMatrixStatus, error) {
	f.calls = append(f.calls, "start")
	return api.SwitchMatrixStatus{Enabled: true}, nil
}

func (f *switchMatrixWebTestAPI) StopSwitchMatrix(_ context.Context, _, id string) (api.SwitchMatrixStatus, error) {
	f.calls = append(f.calls, "stop "+id)
	return api.SwitchMatrixStatus{Enabled: true}, nil
}

func (f *switchMatrixWebTestAPI) RecoverSwitchMatrix(_ context.Context, _, id string) (api.SwitchMatrixStatus, error) {
	f.calls = append(f.calls, "recover "+id)
	return api.SwitchMatrixStatus{Enabled: true}, nil
}

func TestSwitchMatrixWebUIProxyGate(t *testing.T) {
	client := &switchMatrixWebTestAPI{fakeAPI: &fakeAPI{}, role: "administrator"}
	app, err := New(client, Config{ManagementAPI: "v1"})
	if err != nil {
		t.Fatal(err)
	}
	handler := app.Handler()
	const base = "http://example/api/migration/switch-matrix"
	const csrf = "csrf=csrf-token"
	const id = "a32d4725-8951-44cb-8b41-ace226d184a5"
	request := func(method, suffix, body string, want int) {
		t.Helper()
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, authenticatedAdminRequest(method, base+suffix, body))
		if w.Code != want {
			t.Fatalf("%s %s got %d, want %d: %s", method, suffix, w.Code, want, w.Body)
		}
	}
	client.role = "operator"
	request(http.MethodGet, "", "", 403)
	request(http.MethodPost, "/start", csrf, 403)
	client.role = "administrator"
	request(http.MethodPost, "/start", "csrf=wrong", 403)
	request(http.MethodPost, "/start?server=jv-source1", csrf, 400)
	request(http.MethodPost, "/start", csrf+"&operation_id="+id, 400)
	request(http.MethodPost, "/stop", csrf+"&operation_id=invalid", 400)
	request(http.MethodGet, "?server=jv-source1", "", 400)
	if len(client.calls) != 0 {
		t.Fatalf("rejected request reached the agent: %v", client.calls)
	}
	for _, action := range []string{"/start", "/stop", "/recover"} {
		body := csrf
		if action != "/start" {
			body += "&operation_id=" + id
		}
		request(http.MethodPost, action, body, 404)
	}
	for _, call := range client.calls {
		if call != "status " {
			t.Fatalf("production proxy reached an execution method: %v", client.calls)
		}
	}
	// The dashboard represents an unselected server with an empty query value.
	workspace := authenticatedAdminRequest(http.MethodGet, "http://example/workspace/migration/switch?server=", "")
	workspace.Header.Set(migrationWorkspaceFragmentHeader, "1")
	productPage := httptest.NewRecorder()
	handler.ServeHTTP(productPage, workspace)
	if productPage.Code != http.StatusBadRequest {
		t.Fatalf("production changed ordinary server selection: %d", productPage.Code)
	}
	client.enabled = true
	developmentPage := httptest.NewRecorder()
	handler.ServeHTTP(developmentPage, workspace)
	if developmentPage.Code != http.StatusOK || !strings.Contains(developmentPage.Body.String(), `data-server-id=""`) {
		t.Fatalf("development Switch requires a selected ordinary server: %d", developmentPage.Code)
	}
	request(http.MethodGet, "?operation_id="+id, "", 200)
	request(http.MethodPost, "/start", csrf, 202)
	request(http.MethodPost, "/stop", csrf+"&operation_id="+id, 202)
	request(http.MethodPost, "/recover", csrf+"&operation_id="+id, 200)
}
