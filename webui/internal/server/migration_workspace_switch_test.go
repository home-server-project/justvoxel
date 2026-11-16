package server

import (
	"bytes"
	"html/template"
	"regexp"
	"strings"
	"testing"
)

func TestMigrationSwitchUsesCompleteSetupCatalogueAndArtwork(t *testing.T) {
	source, err := assets.ReadFile("templates/migration_workspace_switch.html")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := template.New("switch").Parse(string(source))
	if err != nil {
		t.Fatal(err)
	}
	groups := []struct {
		Name    string
		Choices []setupSoftwareChoice
	}{
		{"Plugins / Paper family", simpleSoftware("plugins")},
		{"Mod loaders", simpleSoftware("mods")},
		{"Original", simpleSoftware("original")},
	}
	var rendered bytes.Buffer
	err = parsed.ExecuteTemplate(&rendered, "migration_workspace_switch.html", struct {
		ServerID, CSRF string
		Groups         any
	}{"jv-family1", "csrf", groups})
	if err != nil {
		t.Fatal(err)
	}
	markup := rendered.String()
	for _, id := range []string{"paper", "purpur", "spigot", "pufferfish", "leaf", "fabric", "neoforge", "forge", "quilt", "vanilla", "bedrock"} {
		if strings.Count(markup, `data-switch-target="`+id+`"`) != 1 || !strings.Contains(markup, `/static/server-software/`+id+`.svg`) {
			t.Fatalf("missing or duplicated shared software/artwork: %s", id)
		}
		if _, err = assets.ReadFile("static/server-software/" + id + ".svg"); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Count(markup, `data-switch-target=`) != 11 || strings.Contains(markup, "data-server-selector-slot") || !strings.Contains(markup, `disabled>`) {
		t.Fatal("catalogue hides software, duplicates the selector, or enables unverified actions")
	}
	stylesheet, err := assets.ReadFile("static/server-migration-switch.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"repeat(auto-fit,minmax(min(170px,100%),1fr))", ".is-current", ".is-unsupported", "migration-switch-sticker"} {
		if !strings.Contains(string(stylesheet), want) {
			t.Fatalf("responsive Switch state missing: %s", want)
		}
	}
}

func TestMigrationAndVersionNavigationBoundaries(t *testing.T) {
	header, err := assets.ReadFile("templates/header.html")
	if err != nil {
		t.Fatal(err)
	}
	markup := string(header)
	migration := regexp.MustCompile(`(?s)<nav class="migration-workspace-tabs".*?</nav>`).FindString(markup)
	for _, tab := range []string{"export", "import", "switch"} {
		if !strings.Contains(migration, `data-migration-tab="`+tab+`"`) {
			t.Fatalf("Migration missing %s", tab)
		}
	}
	if strings.Contains(migration, `data-migration-tab="recovery"`) {
		t.Fatal("Recovery became a normal tab")
	}
	version := regexp.MustCompile(`(?s)<nav class="version-workspace-tabs".*?</nav>`).FindString(markup)
	if strings.Count(version, "data-version-tab=") != 2 || !strings.Contains(version, `data-version-tab="container"`) || !strings.Contains(version, `data-version-tab="minecraft"`) || strings.Contains(version, "Switch") {
		t.Fatal("Minecraft Update must contain only Container and Version")
	}
	progress, err := assets.ReadFile("templates/migration_workspace_progress.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(progress), "tab=recovery") || !strings.Contains(string(progress), "CanReviewRecovery") {
		t.Fatal("contextual migration recovery was lost")
	}
	versionScript, err := assets.ReadFile("static/version-workspace.js")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(versionScript), `currentTab === "software"`) || strings.Contains(string(versionScript), "Switch Available") {
		t.Fatal("old software switching still appears in Version")
	}
	for _, want := range []string{"renderContainer", "Minecraft Version", "/api/version/workspace/update", "/api/version/container/update"} {
		if !strings.Contains(string(versionScript), want) {
			t.Fatalf("normal update path removed: %s", want)
		}
	}
}
