package main

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func openTestWebUIStore(t *testing.T) (*webUIStore, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "webui.db")
	store, err := openWebUIStore(path)
	if err != nil {
		t.Fatalf("open WebUI store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.close(); err != nil {
			t.Errorf("close WebUI store: %v", err)
		}
	})
	return store, path
}

// Create a deployed schema using only the migrations that existed at that version.
func openTestWebUIStoreAtVersion(t *testing.T, version int) (*webUIStore, string) {
	t.Helper()
	migrations := webUIStoreMigrations
	defer func() { webUIStoreMigrations = migrations }()
	webUIStoreMigrations = migrations[:version]
	return openTestWebUIStore(t)
}

func TestWebUIAdminUpgradePreservesAccountDependencies(t *testing.T) {
	store, _ := openTestWebUIStoreAtVersion(t, 13)
	user, err := store.createWebUser("existing", webUserRoleOperator, "existing-password")
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`UPDATE operator_usage SET restart_used=2, backup_used=1 WHERE user_id=?`,
		`INSERT INTO notifications(created_at,kind,user_id,title,message) VALUES ('before','limit',?,'title','message')`,
		`INSERT INTO web_user_two_factor(user_id,secret,source,recovery_hashes,last_step) VALUES (?,'test-secret','administrator','[]',123)`,
	} {
		if _, err := store.db.Exec(statement, user.ID); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.recordAuditEvent(administratorSession(time.Now()), "before", user.Username, true, ""); err != nil {
		t.Fatal(err)
	}
	if err := applyWebUIStoreMigrations(store.db); err != nil {
		t.Fatal(err)
	}
	got, valid, err := store.checkWebUserPassword(user.Username, "existing-password")
	if err != nil || !valid || got.ID != user.ID || got.Role != user.Role {
		t.Fatal("credential changed on upgrade", err)
	}
	var count int
	err = store.db.QueryRow(`SELECT COUNT(*) FROM operator_usage u
		JOIN notifications n ON n.user_id=u.user_id JOIN web_user_two_factor f ON f.user_id=u.user_id
		WHERE u.user_id=? AND restart_used=2 AND backup_used=1 AND n.message='message'
		AND f.last_step=123 AND f.secret='test-secret' AND f.source='administrator' AND f.recovery_hashes='[]'`, user.ID).Scan(&count)
	if err != nil || count != 1 {
		t.Fatal("lost account dependencies", err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE actor_role='administrator' AND action='before'`).Scan(&count); err != nil || count != 1 {
		t.Fatal("lost audit history", err)
	}
	if err := store.setWebUserRole(user.ID, webUserRoleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := store.recordAuditEvent(session{Username: user.Username, Role: roleWebUIAdmin}, "after", "", true, ""); err != nil {
		t.Fatal(err)
	}
}

func TestWebUIStoreStartsWithoutWebUsers(t *testing.T) {
	store, path := openTestWebUIStore(t)
	users, err := store.listWebUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 0 {
		t.Fatalf("fresh database unexpectedly contains %d WebUI users", len(users))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("database mode = %04o, want 0600", got)
	}
}

func TestWebUIStoreAllowsMultipleWebAccounts(t *testing.T) {
	store, _ := openTestWebUIStore(t)
	base := time.Date(2026, time.September, 17, 1, 2, 3, 0, time.UTC)
	store.now = func() time.Time { return base }

	accounts := []struct {
		username string
		role     webUserRole
	}{
		{"Manager", webUserRoleAdmin},
		{"Ilya", webUserRoleOperator},
		{"Alex", webUserRoleOperator},
		{"Maya", webUserRoleViewer},
		{"Noah", webUserRoleViewer},
		{"Sofia", webUserRoleViewer},
	}
	for i, account := range accounts {
		if _, err := store.createWebUser(account.username, account.role, "web-password-"+account.username); err != nil {
			t.Fatalf("create account %d (%s): %v", i, account.username, err)
		}
	}

	users, err := store.listWebUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != len(accounts) {
		t.Fatalf("got %d WebUI users, want %d", len(users), len(accounts))
	}
}

func TestWebUIStoreReservesVoxelForPrimaryAdministrator(t *testing.T) {
	store, _ := openTestWebUIStore(t)
	for _, username := range []string{"voxel", "Voxel", " VOXEL "} {
		if _, err := store.createWebUser(username, webUserRoleOperator, "secret-password"); !errors.Is(err, errWebUserReserved) {
			t.Fatalf("create %q returned %v, want reserved-name error", username, err)
		}
	}
}

func TestWebUIStoreAcceptsOnlyFixedWebUserRoles(t *testing.T) {
	store, _ := openTestWebUIStore(t)
	if _, err := store.createWebUser("alex", webUserRole("administrator"), "secret-password"); !errors.Is(err, errWebUserInvalidRole) {
		t.Fatalf("administrator WebUI row returned %v, want invalid-role error", err)
	}
	if _, err := store.createWebUser("custom", webUserRole("custom"), "secret-password"); !errors.Is(err, errWebUserInvalidRole) {
		t.Fatalf("custom WebUI role returned %v, want invalid-role error", err)
	}
}

func TestWebUIStorePasswordLifecycleAndDisable(t *testing.T) {
	store, _ := openTestWebUIStore(t)
	user, err := store.createWebUser("Ilya", webUserRoleOperator, "first-password")
	if err != nil {
		t.Fatal(err)
	}

	got, ok, err := store.checkWebUserPassword("ilya", "first-password")
	if err != nil || !ok || got.ID != user.ID {
		t.Fatalf("initial credential failed: user=%#v ok=%v err=%v", got, ok, err)
	}
	if _, ok, err := store.checkWebUserPassword("Ilya", "wrong-password"); err != nil || ok {
		t.Fatalf("wrong credential result ok=%v err=%v", ok, err)
	}

	if err := store.setWebUserPassword(user.ID, "second-password"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.checkWebUserPassword("Ilya", "first-password"); err != nil || ok {
		t.Fatalf("old credential still accepted: ok=%v err=%v", ok, err)
	}
	if _, ok, err := store.checkWebUserPassword("Ilya", "second-password"); err != nil || !ok {
		t.Fatalf("new credential rejected: ok=%v err=%v", ok, err)
	}

	if err := store.setWebUserEnabled(user.ID, false); err != nil {
		t.Fatal(err)
	}
	if disabled, ok, err := store.checkWebUserPassword("Ilya", "second-password"); err != nil || ok || disabled.Enabled {
		t.Fatalf("disabled account authenticated: user=%#v ok=%v err=%v", disabled, ok, err)
	}
}

func TestWebUIStoreRoleChangeAndDelete(t *testing.T) {
	store, _ := openTestWebUIStore(t)
	user, err := store.createWebUser("Maya", webUserRoleViewer, "viewer-password")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.setWebUserRole(user.ID, webUserRoleOperator); err != nil {
		t.Fatal(err)
	}
	updated, err := store.webUserByID(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Role != webUserRoleOperator {
		t.Fatalf("role = %q, want operator", updated.Role)
	}
	if err := store.deleteWebUser(user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.webUserByID(user.ID); !errors.Is(err, errWebUserNotFound) {
		t.Fatalf("deleted user lookup returned %v", err)
	}
}

func TestWebUIStoreMigrationPersistsAcrossReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "webui.db")
	store, err := openWebUIStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.createWebUser("Alex", webUserRoleViewer, "viewer-password"); err != nil {
		t.Fatal(err)
	}
	if err := store.close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := openWebUIStore(path)
	if err != nil {
		t.Fatalf("reopen migrated database: %v", err)
	}
	defer reopened.close()
	users, err := reopened.listWebUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].Username != "Alex" {
		t.Fatalf("persisted users = %#v", users)
	}
}

func TestWebUIStoreRejectsUnknownFutureSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "webui.db")
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations(version, name, applied_at) VALUES(999, 'future', '2026-09-17T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	if store, err := openWebUIStore(path); err == nil {
		_ = store.close()
		t.Fatal("future schema unexpectedly opened")
	}
}

func TestWebUIStoreFoundationTablesExist(t *testing.T) {
	store, _ := openTestWebUIStore(t)
	for _, table := range []string{
		"web_users",
		"operator_usage",
		"operator_global_state",
		"audit_events",
		"notifications",
		"system_monitor_profile",
		"schema_migrations",
	} {
		var name string
		err := store.db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}
}

func TestWebUIStoreFactoryResetReturnsUserStateToDefaults(t *testing.T) {
	store, _ := openTestWebUIStore(t)
	base := time.Date(2026, time.September, 25, 20, 30, 0, 0, time.UTC)
	store.now = func() time.Time { return base }

	user, err := store.createWebUser("Alex", webUserRoleOperator, "operator-password")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(
		`INSERT INTO audit_events(occurred_at, actor_username, actor_role, action, target, success, context)
		 VALUES(?, 'voxel', 'administrator', 'test', 'factory-reset', 1, '')`,
		formatWebUIStoreTime(base),
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(
		`INSERT INTO notifications(created_at, kind, user_id, title, message)
		 VALUES(?, 'test', ?, 'notice', 'message')`,
		formatWebUIStoreTime(base),
		user.ID,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.Exec(
		`UPDATE operator_global_state SET last_restart_at = ?, last_backup_at = ?, updated_at = ? WHERE id = 1`,
		formatWebUIStoreTime(base), formatWebUIStoreTime(base), formatWebUIStoreTime(base),
	); err != nil {
		t.Fatal(err)
	}
	changed := defaultSystemMonitorProfile()
	changed.ProcessCount = 15
	changed.Alerts = false
	if err := store.writeSystemMonitorProfile(changed); err != nil {
		t.Fatal(err)
	}

	if err := store.resetFactoryState(); err != nil {
		t.Fatal(err)
	}

	users, err := store.listWebUsers()
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 0 {
		t.Fatalf("factory reset left WebUI users: %#v", users)
	}
	for _, table := range []string{"operator_usage", "audit_events", "notifications"} {
		var count int
		if err := store.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("factory reset left %d rows in %s", count, table)
		}
	}

	var restartAt, backupAt sql.NullString
	if err := store.db.QueryRow(
		`SELECT last_restart_at, last_backup_at FROM operator_global_state WHERE id = 1`,
	).Scan(&restartAt, &backupAt); err != nil {
		t.Fatal(err)
	}
	if restartAt.Valid || backupAt.Valid {
		t.Fatalf("factory reset left operator global timestamps: restart=%v backup=%v", restartAt, backupAt)
	}

	profile, err := store.readSystemMonitorProfile()
	if err != nil {
		t.Fatal(err)
	}
	if profile != defaultSystemMonitorProfile() {
		t.Fatalf("factory reset monitor profile = %#v, want default %#v", profile, defaultSystemMonitorProfile())
	}
}
