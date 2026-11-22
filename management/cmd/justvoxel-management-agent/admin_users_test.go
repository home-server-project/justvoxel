package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func adminServerWithStore(t *testing.T) (*server, *webUIStore) {
	t.Helper()
	store, _ := openTestWebUIStore(t)
	now := time.Now()
	return &server{
		store: store,
		sessions: map[string]session{
			"token": {
				Username:   systemAdminUsername,
				Role:       roleAdministrator,
				AuthSource: authSourceSystem,
				Created:    now,
				LastSeen:   now,
			},
		},
	}, store
}

func TestAdministratorCreatesAndListsWebUIUsers(t *testing.T) {
	s, store := adminServerWithStore(t)
	oldValidate := systemValidatePass
	systemValidatePass = func(_, _, _ string) error { return nil }
	defer func() { systemValidatePass = oldValidate }()

	rr := httptest.NewRecorder()
	s.adminCreateUser(rr, authorizedRequest(http.MethodPost, "/v1/admin/users", `{"username":"Ilya","role":"operator","password":"example-password"}`))
	if rr.Code != http.StatusCreated {
		t.Fatalf("create returned %d: %s", rr.Code, rr.Body.String())
	}
	var created adminUserView
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Username != "Ilya" || created.Role != webUserRoleOperator || created.RestartLimit != 2 || created.BackupLimit != 2 {
		t.Fatalf("unexpected created user: %#v", created)
	}
	if _, ok, err := store.checkWebUserPassword("Ilya", "example-password"); err != nil || !ok {
		t.Fatalf("created credential rejected: ok=%v err=%v", ok, err)
	}

	rr = httptest.NewRecorder()
	s.adminListUsers(rr, authorizedRequest(http.MethodGet, "/v1/admin/users", ""))
	if rr.Code != http.StatusOK {
		t.Fatalf("list returned %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"username":"voxel"`) || !strings.Contains(rr.Body.String(), `"username":"Ilya"`) {
		t.Fatalf("list missing primary admin or user: %s", rr.Body.String())
	}
}

func TestViewerCannotManageUsers(t *testing.T) {
	store, _ := openTestWebUIStore(t)
	s := roleServerForTest(roleViewer)
	s.store = store
	rr := httptest.NewRecorder()
	s.adminListUsers(rr, authorizedRequest(http.MethodGet, "/v1/admin/users", ""))
	if rr.Code != http.StatusForbidden {
		t.Fatalf("viewer list returned %d", rr.Code)
	}
}

func TestDisablingChangingRoleAndPasswordInvalidateWebUserSessions(t *testing.T) {
	s, store := adminServerWithStore(t)
	user, err := store.createWebUser("Alex", webUserRoleOperator, "first-password")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	s.sessions["alex"] = session{Username: user.Username, Role: roleOperator, AuthSource: authSourceWebUI, WebUserID: user.ID, Created: now, LastSeen: now}

	req := authorizedRequest(http.MethodPatch, "/v1/admin/users/1/enabled", `{"enabled":false}`)
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	s.adminSetUserEnabled(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("disable returned %d: %s", rr.Code, rr.Body.String())
	}
	if _, exists := s.sessions["alex"]; exists {
		t.Fatal("disabled user's session remained active")
	}

	if err := store.setWebUserEnabled(user.ID, true); err != nil {
		t.Fatal(err)
	}
	s.sessions["alex2"] = session{Username: user.Username, Role: roleOperator, AuthSource: authSourceWebUI, WebUserID: user.ID, Created: now, LastSeen: now}
	req = authorizedRequest(http.MethodPatch, "/v1/admin/users/1/role", `{"role":"viewer"}`)
	req.SetPathValue("id", "1")
	rr = httptest.NewRecorder()
	s.adminSetUserRole(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("role change returned %d: %s", rr.Code, rr.Body.String())
	}
	if _, exists := s.sessions["alex2"]; exists {
		t.Fatal("role-changed user's session remained active")
	}

	oldValidate := systemValidatePass
	systemValidatePass = func(_, _, _ string) error { return nil }
	defer func() { systemValidatePass = oldValidate }()
	s.sessions["alex3"] = session{Username: user.Username, Role: roleViewer, AuthSource: authSourceWebUI, WebUserID: user.ID, Created: now, LastSeen: now}
	req = authorizedRequest(http.MethodPost, "/v1/admin/users/1/password", `{"password":"second-password"}`)
	req.SetPathValue("id", "1")
	rr = httptest.NewRecorder()
	s.adminSetUserPassword(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("password reset returned %d: %s", rr.Code, rr.Body.String())
	}
	if _, exists := s.sessions["alex3"]; exists {
		t.Fatal("password-reset user's session remained active")
	}
}

func TestAdministratorCanResetIndependentOperatorAllowances(t *testing.T) {
	s, store := adminServerWithStore(t)
	user, err := store.createWebUser("Maya", webUserRoleOperator, "operator-password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(`UPDATE operator_usage SET restart_used = 2, backup_used = 2 WHERE user_id = ?`, user.ID); err != nil {
		t.Fatal(err)
	}

	req := authorizedRequest(http.MethodPost, "/v1/admin/users/1/restart-allowance/reset", "")
	req.SetPathValue("id", "1")
	rr := httptest.NewRecorder()
	s.adminResetRestartAllowance(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("restart reset returned %d: %s", rr.Code, rr.Body.String())
	}
	var restartUsed, backupUsed int
	if err := store.db.QueryRow(`SELECT restart_used, backup_used FROM operator_usage WHERE user_id = ?`, user.ID).Scan(&restartUsed, &backupUsed); err != nil {
		t.Fatal(err)
	}
	if restartUsed != 0 || backupUsed != 2 {
		t.Fatalf("restart reset changed wrong counters: restart=%d backup=%d", restartUsed, backupUsed)
	}

	req = authorizedRequest(http.MethodPost, "/v1/admin/users/1/backup-allowance/reset", "")
	req.SetPathValue("id", "1")
	rr = httptest.NewRecorder()
	s.adminResetBackupAllowance(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("backup reset returned %d: %s", rr.Code, rr.Body.String())
	}
	if err := store.db.QueryRow(`SELECT restart_used, backup_used FROM operator_usage WHERE user_id = ?`, user.ID).Scan(&restartUsed, &backupUsed); err != nil {
		t.Fatal(err)
	}
	if restartUsed != 0 || backupUsed != 0 {
		t.Fatalf("backup reset changed wrong counters: restart=%d backup=%d", restartUsed, backupUsed)
	}
}

func TestWebUIAdminManagesOnlyWebAccounts(t *testing.T) {
	withAuthTestDependencies(t)
	systemValidatePass = func(_, _, _ string) error { return nil }
	readAuthMode = func() (authMode, error) {
		t.Fatal("WebUI account consulted system account authentication")
		return "", nil
	}
	systemChangePassword = func(string, string, string) error { t.Fatal("WebUI account updated a system password"); return nil }
	writeSeparateAdmin = func(string) error { t.Fatal("WebUI account updated voxel credentials"); return nil }
	s, store := adminServerWithStore(t)
	admin, err := store.createWebUser("manager", webUserRoleAdmin, "manager-password")
	if err != nil {
		t.Fatal(err)
	}
	actor := s.sessions["token"]
	actor.Username, actor.Role, actor.WebUserID, actor.AuthSource = admin.Username, roleWebUIAdmin, admin.ID, authSourceWebUI
	s.sessions["token"] = actor
	mux := http.NewServeMux()
	registerAdminUserRoutes(mux, s)
	request := func(method, path, body string, status int) {
		t.Helper()
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, authorizedRequest(method, "/v1/admin/users"+path, body))
		if rr.Code != status {
			t.Fatalf("%s %s returned %d, want %d", method, path, rr.Code, status)
		}
	}
	for _, role := range []webUserRole{webUserRoleAdmin, webUserRoleOperator, webUserRoleViewer} {
		username := "new-" + string(role)
		request("POST", "", `{"username":"`+username+`","password":"web-password","role":"`+string(role)+`"}`, 201)
		identity, err := s.authenticateIdentity(username, "web-password")
		if err != nil || identity.Role != principalRole(role) || identity.WebUserID <= 0 || identity.AuthSource != authSourceWebUI {
			t.Fatalf("incorrect new %s identity: %+v, %v", role, identity, err)
		}
		path := "/" + strconv.FormatInt(identity.WebUserID, 10)
		for _, next := range []string{"operator", "viewer", "webui-admin"} {
			request("PATCH", path+"/role", `{"role":"`+next+`"}`, 200)
			user, err := store.webUserByID(identity.WebUserID)
			if err != nil || string(user.Role) != next {
				t.Fatalf("role change to %s failed", next)
			}
		}
		request("PATCH", path+"/role", `{"role":"administrator"}`, 400)
		request("POST", path+"/password", `{"password":"updated-password"}`, 200)
		if _, valid, err := store.checkWebUserPassword(username, "updated-password"); err != nil || !valid {
			t.Fatal("password change did not update WebUI credentials", err)
		}
		request("PATCH", path+"/enabled", `{"enabled":false}`, 200)
		request("PATCH", path+"/enabled", `{"enabled":true}`, 200)
		request("DELETE", path, "", 200)
	}
	request("POST", "", `{"username":"other","password":"web-password","role":"administrator"}`, 400)
	request("POST", "", `{"username":"Voxel","password":"web-password","role":"webui-admin"}`, 400)
	for _, primary := range []string{"/0", "/voxel"} {
		request("PATCH", primary+"/role", `{"role":"viewer"}`, 400)
		request("PATCH", primary+"/enabled", `{"enabled":false}`, 400)
		request("DELETE", primary, "", 400)
	}
	for _, role := range []principalRole{roleOperator, roleViewer} {
		actor.Role = role
		s.sessions["token"] = actor
		request("POST", "", `{"username":"other","role":"webui-admin","password":"password"}`, 403)
	}
}
