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
	legend := `<div class="migration-switch-legend">
<span><span class="plugins-badge plugins-badge-disabled migration-switch-status-dot" aria-hidden="true">C</span> Current</span>
<span><span class="plugins-badge migration-switch-status-dot" aria-hidden="true">N</span> Not Supported</span>
<span><span class="plugins-badge plugins-badge-enabled migration-switch-status-dot" aria-hidden="true">A</span> Available</span>
</div>
<p class="muted">Switch server software at the same Minecraft version. Review compatibility before converting this server or creating a separate clone.</p>`
	if !strings.Contains(markup, legend) {
		t.Fatal("Switch legend must show Current, Not Supported and Available in that order immediately above the introduction")
	}
	introEnd := strings.Index(markup, legend) + len(legend)
	catalogue := strings.Index(markup, `migration-switch-catalogue"`)
	lastGroupEnd := strings.LastIndex(markup, `</fieldset>`) + len(`</fieldset>`)
	content := strings.Index(markup, `<div class="migration-switch-content migration-switch-stack">`)
	operation := strings.Index(markup, `<div class="migration-switch-stack" data-switch-operation role="status"></div>`)
	state := strings.Index(markup, `data-switch-state`)
	review := strings.Index(markup, `data-switch-review`)
	if strings.Count(markup, `data-switch-operation`) != 1 || catalogue < introEnd ||
		lastGroupEnd <= catalogue || content < lastGroupEnd || operation < content ||
		strings.TrimSpace(markup[lastGroupEnd:content]) != `</div>` || state <= operation || review <= state {
		t.Fatal("Switch must show the introduction and complete catalogue before the shared area for operation status, compatibility state and review")
	}
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
	cardBody := regexp.MustCompile(`^\s*<span class="migration-switch-icon-column">\s*` +
		`<span class="server-software-icon" aria-hidden="true"><img src="/static/server-software/([^"]+)\.svg" alt="" width="40" height="40"></span>\s*` +
		`<span class="plugins-badge migration-switch-status-dot" data-switch-status aria-hidden="true">N</span>\s*</span>\s*` +
		`<span class="server-software-copy"><strong>([^<]+)</strong><small>([^<]+)</small></span>\s*$`)
	var names []string
	for _, card := range cards {
		body := cardBody.FindStringSubmatch(card[2])
		if body == nil || body[1] != card[1] {
			t.Fatalf("card must contain the matching icon, one permanent status marker below it and plain Setup copy: %s", card[1])
		}
		names = append(names, body[2])
		for _, group := range setupSoftwareGroups() {
			for _, choice := range group.Software {
				if choice.ID == card[1] && (body[2] != choice.Name || body[3] != choice.Description) {
					t.Fatalf("Switch changed Setup copy: %s", choice.ID)
				}
			}
		}
		if !strings.Contains(card[0], "is-pending") || !strings.Contains(card[0], " disabled>") {
			t.Fatalf("unverified card must start pending and disabled: %s", card[1])
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
	if strings.Count(markup, `data-switch-target=`) != 11 || strings.Contains(markup, "data-server-selector-slot") {
		t.Fatal("catalogue hides software or duplicates the selector")
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
	for _, want := range []string{
		".server-software-card{grid-template-columns:40px minmax(0,1fr);gap:.7rem;height:104px;min-height:104px;max-height:104px",
		".migration-switch-icon-column{display:grid;justify-items:center;gap:.25rem;width:40px}",
		".migration-switch-status-dot{display:inline-grid;place-items:center;width:22px;height:22px;box-sizing:border-box;border-radius:999px;padding:0;font-weight:800;line-height:1}",
		".is-pending .migration-switch-status-dot{visibility:hidden}",
		".migration-switch-status-dot:not(.plugins-badge-enabled,.plugins-badge-disabled){background:#343a40}",
		".is-current{border-color:#e5bb40;background:color-mix(in srgb,#e5bb40 12%,var(--jv-surface-sunken));opacity:1}",
		".is-unsupported{filter:brightness(.82);opacity:.72;cursor:default}",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("Switch overlay state missing: %s", want)
		}
	}
	for _, obsolete := range []string{"migration-switch-title-row", "migration-switch-sticker", "display:none", "text-overflow", "13ch"} {
		if strings.Contains(css, obsolete) || strings.Contains(markup, obsolete) {
			t.Fatalf("Switch status must reserve only the marker slot and leave software copy unrestricted: %s", obsolete)
		}
	}
}

func TestMigrationSwitchSelectionAndReviewSource(t *testing.T) {
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
		t.Fatal("pending markers or protected-clone card semantics regressed")
	}
	selectTarget := regexp.MustCompile(`(?s)const select = .*?const load =`).FindString(string(script))
	selection := strings.Index(selectTarget, `node.classList.toggle("is-selected", node.dataset.switchTarget === target)`)
	request := strings.Index(selectTarget, `await request("plan", { target })`)
	guard := strings.Index(selectTarget, `if (!current() || ticket !== sequence) return;`)
	update := strings.Index(selectTarget, "showCard(")
	if selection < 0 || request <= selection || guard <= request || update <= guard ||
		!strings.Contains(selectTarget[:selection], "const ticket = ++sequence;") || strings.Count(selectTarget, `"is-selected"`) != 1 {
		t.Fatal("selection must update once before the request and stale responses must be ignored before updating the card or review")
	}
	if strings.Contains(string(script), "Open Plugins & Mods") {
		t.Fatal("review must use the Plugins & Mods action label")
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
