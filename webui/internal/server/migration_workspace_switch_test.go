package server

import (
	"bytes"
	"html"
	"html/template"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestMigrationSwitchUsesCompleteSetupCatalogueAndArtwork(t *testing.T) {
	source, err := englishPresentationAsset("templates/migration_workspace_switch.html")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := template.New("switch").Funcs(localeTemplateFuncs("en", "/login")).Parse(string(source))
	if err != nil {
		t.Fatal(err)
	}
	var rendered bytes.Buffer
	err = parsed.ExecuteTemplate(&rendered, "migration_workspace_switch.html", struct {
		ServerID, CSRF string
		Groups         []setupSoftwareGroup
	}{"jv-family1", "csrf", setupSoftwareGroups()})
	if err != nil {
		t.Fatal(err)
	}
	markup := html.UnescapeString(rendered.String())
	page, err := os.ReadFile("migration_workspace_switch.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), "csrfFromRequest(r), setupSoftwareGroups()}") || strings.Contains(string(page), "simpleSoftware(") {
		t.Fatal("Switch must use setupSoftwareGroups directly without reconstructing the catalogue")
	}
	var headings []string
	for _, match := range regexp.MustCompile(`<legend>([^<]+)</legend>`).FindAllStringSubmatch(markup, -1) {
		headings = append(headings, match[1])
	}
	if !slices.Equal(headings, []string{"Plugins · Cross-play", "Mod loaders · Cross-play", "Vanilla Java · Bedrock"}) {
		t.Fatalf("Switch headings differ from Advanced Setup: %v", headings)
	}
	cards := regexp.MustCompile(`(?s)<button[^>]+data-switch-target="([^"]+)"[^>]*>(.*?)</button>`).FindAllStringSubmatch(markup, -1)
	var names []string
	for _, card := range cards {
		copy := regexp.MustCompile(`<span class="server-software-copy"><span class="migration-switch-title-row"><strong>([^<]+)</strong><span class="migration-switch-sticker" data-switch-sticker>Not Supported</span></span><small>([^<]+)</small></span>`).FindStringSubmatch(card[2])
		if copy == nil {
			t.Fatalf("software name/description presentation changed: %s", card[1])
		}
		names = append(names, copy[1])
		if copy[1] == "Vanilla Java" || copy[1] == "native Bedrock" {
			t.Fatalf("Switch renamed the Setup card: %s", copy[1])
		}
		for _, group := range setupSoftwareGroups() {
			for _, choice := range group.Software {
				if choice.ID == card[1] && (copy[1] != choice.Name || copy[2] != choice.Description) {
					t.Fatalf("Switch changed Setup copy: %s", choice.ID)
				}
			}
		}
		if !strings.Contains(card[0], "is-pending") || strings.Count(card[2], "data-switch-sticker") != 1 {
			t.Fatalf("permanent pending title badge missing: %s", card[1])
		}
		if strings.Contains(card[2], "<p") || strings.Count(card[2], "<small") != 1 {
			t.Fatalf("unexpected per-card paragraph or reason: %s", card[1])
		}
	}
	if !slices.Equal(names, []string{"Paper", "Purpur", "Spigot", "Pufferfish", "Leaf", "Fabric", "NeoForge", "Forge", "Quilt", "Vanilla", "Bedrock"}) {
		t.Fatalf("Switch card names differ from Advanced Setup: %v", names)
	}
	if strings.Contains(string(source), "data-switch-reason") {
		t.Fatal("visible compatibility reasons must not appear inside software cards")
	}
	for _, class := range []string{"setup-software-lanes", "setup-server-types setup-server-types-compact", "setup-server-type server-software-card"} {
		if !strings.Contains(markup, class) {
			t.Fatalf("shared Setup class missing: %s", class)
		}
	}
	for _, id := range []string{"paper", "purpur", "spigot", "pufferfish", "leaf", "fabric", "neoforge", "forge", "quilt", "vanilla", "bedrock"} {
		if strings.Count(markup, `data-switch-target="`+id+`"`) != 1 || !strings.Contains(markup, `/static/server-software/`+id+`.svg`) {
			t.Fatalf("missing or duplicated shared software/artwork: %s", id)
		}
		if _, err = englishPresentationAsset("static/server-software/" + id + ".svg"); err != nil {
			t.Fatal(err)
		}
	}
	if strings.Count(markup, `data-switch-target=`) != 11 || strings.Contains(markup, "data-server-selector-slot") || !strings.Contains(markup, `disabled>`) {
		t.Fatal("catalogue hides software, duplicates the selector, or enables unverified actions")
	}
	stylesheet, err := englishPresentationAsset("static/server-migration-switch.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(stylesheet)
	setupStylesheet, err := englishPresentationAsset("static/setup.css")
	if err != nil {
		t.Fatal(err)
	}
	// These declarations must remain identical to the authoritative Setup rules.
	for _, rule := range []string{
		"grid-template-columns:1fr;grid-auto-rows:auto;align-content:start;align-items:start",
		"grid-template-columns:repeat(5,minmax(0,1fr));grid-auto-rows:1fr;align-content:start;min-width:0;margin:0",
		"padding:.65rem .75rem;box-sizing:border-box;align-items:start;align-content:start",
		"width:40px;min-width:40px;height:40px",
		".server-software-copy strong{font-size:1rem}",
		".server-software-copy small{font-size:.82rem}",
	} {
		if !strings.Contains(css, rule) || !strings.Contains(string(setupStylesheet), rule) {
			t.Fatalf("Switch and Advanced Setup presentation diverged: %s", rule)
		}
	}
	for _, breakpoint := range []struct{ width, columns string }{{"1000", "repeat(2,minmax(0,1fr))"}, {"430", "1fr"}} {
		rule := "@media(max-width:" + breakpoint.width + "px){\n  "
		lane := ".setup-software-lanes .setup-server-types{grid-template-columns:" + breakpoint.columns + "}"
		if !strings.Contains(css, rule+".migration-switch "+lane) || !strings.Contains(string(setupStylesheet), rule+lane) {
			t.Fatalf("Switch breakpoint differs from Advanced Setup: %s", breakpoint.width)
		}
	}
	if strings.Contains(css, "auto-fit") {
		t.Fatal("Switch lanes must retain empty desktop tracks")
	}
	for _, want := range []string{".is-current", ".is-unsupported", ".migration-switch-title-row{display:flex;", "height:104px;min-height:104px;max-height:104px", "is-pending .migration-switch-sticker{visibility:hidden}", "text-overflow:ellipsis;white-space:nowrap", "background:#e5bb40", "background:#555b61"} {
		if !strings.Contains(css, want) {
			t.Fatalf("Switch overlay state missing: %s", want)
		}
	}
	stickerRule := regexp.MustCompile(`\.migration-switch \.migration-switch-sticker\{[^}]+\}`).FindString(css)
	if strings.Contains(stickerRule, "position:absolute") || strings.Contains(stickerRule, "display:none") || !strings.Contains(stickerRule, "white-space:nowrap") {
		t.Fatal("badge must stay in the title row and reserve its slot")
	}
	script, err := englishPresentationAsset("static/server-migration-switch.js")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(script), "data-switch-reason") || !strings.Contains(string(script), "node.title = plan.reason") || !strings.Contains(string(script), `review.appendChild(paragraph(plan.reason, "notice warning"))`) {
		t.Fatal("compatibility reasons belong in tooltips and selected-target review only")
	}
	showCard := regexp.MustCompile(`(?s)const showCard = .*?const reconnect`).FindString(string(script))
	for _, mutation := range []string{"createElement", "appendChild", "replaceChildren", "innerHTML", "style.height", "<small"} {
		if strings.Contains(showCard, mutation) {
			t.Fatalf("compatibility response changes card structure: %s", mutation)
		}
	}
	if !strings.Contains(showCard, `node.classList.toggle("is-pending", false)`) || !strings.Contains(showCard, `Boolean(plan.convert || plan.clone)`) {
		t.Fatal("pending badges or protected-clone card semantics regressed")
	}

}

func TestMigrationAndVersionNavigationBoundaries(t *testing.T) {
	header, err := englishPresentationAsset("templates/header.html")
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
	progress, err := englishPresentationAsset("templates/migration_workspace_progress.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(progress), "tab=recovery") || !strings.Contains(string(progress), "CanReviewRecovery") {
		t.Fatal("contextual migration recovery was lost")
	}
	versionScript, err := englishPresentationAsset("static/version-workspace.js")
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
