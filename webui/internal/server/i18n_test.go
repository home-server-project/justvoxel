package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/home-server-project/justvoxel-webui/internal/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestEmbeddedWebUICatalogs(t *testing.T) {
	for _, locale := range supportedLocales {
		t.Run(locale, func(t *testing.T) {
			data, err := assets.ReadFile("locales/" + locale + ".json")
			if err != nil {
				t.Fatal(err)
			}
			var messages map[string]string
			if err := json.Unmarshal(data, &messages); err != nil {
				t.Fatal(err)
			}
			if len(messages) == 0 {
				t.Fatal("catalog is empty")
			}
			if err := validateCatalog(catalogs["en"], messages); err != nil {
				t.Fatal(err)
			}
			// Source lookups may choose any canonical key with the same English text.
			bySource := make(map[string]string, len(messages))
			for key, source := range catalogs["en"] {
				if previous, ok := bySource[source]; ok && previous != messages[key] {
					t.Errorf("inconsistent translation for %q at %s", source, key)
				}
				bySource[source] = messages[key]
			}
		})
	}
}

func TestCatalogCompatibilityAndFallback(t *testing.T) {
	source := map[string]string{"example": "Delete {name}", "printf": "%s has %d players"}
	for _, other := range []map[string]string{
		{"example": "Supprimer {name}"},
		{"example": "Supprimer {other}", "printf": "%s a %d joueurs"},
		{"example": "Supprimer {name}", "printf": "%d a %s joueurs"},
	} {
		if validateCatalog(source, other) == nil {
			t.Fatal("accepted incompatible catalog")
		}
	}
	for _, value := range []string{"", "Supprimer {other}"} {
		if got := translationValue("Delete {name}", value); got != "Delete {name}" {
			t.Fatalf("unsafe fallback: %q", got)
		}
	}
	if got := translate("unsupported", "language.close"); got != "Close" {
		t.Fatalf("unsupported catalog: %q", got)
	}
	// Values stay opaque, even when they look like placeholders or interface text.
	if got := translate("fr-CA", sourceKeys["Delete {v0}"], "v0", "{v1}<script>"); got != "Supprimer {v1}<script>" {
		t.Fatal(got)
	}
}

func TestWebUILanguageSelection(t *testing.T) {
	for _, locale := range supportedLocales {
		t.Run("cookie/"+locale, func(t *testing.T) {
			accept := "en"
			if locale == "en" {
				accept = "fr-CA"
			}
			r := httptest.NewRequest("GET", "/login", nil)
			r.Header.Set("Accept-Language", accept)
			r.AddCookie(&http.Cookie{Name: languageCookie, Value: locale})
			if got := requestLocale(r); got != locale {
				t.Fatalf("cookie %s over browser %s: got %s", locale, accept, got)
			}
		})
	}
	for _, tc := range []struct{ accept, want string }{
		{"en-GB", "en"}, {"fr-CA", "fr-CA"}, {"fr", "fr-CA"}, {"fr-FR", "fr-CA"},
		{"de", "de"}, {"de-DE", "de"}, {"pl-PL", "pl"}, {"uk-UA", "uk"}, {"ru-RU", "ru"},
		{"pt-BR", "pt-BR"}, {"pt", "pt-BR"}, {"pt-PT", "pt-BR"},
		{"es-419", "es-419"}, {"es-MX", "es-419"}, {"es-AR", "es-419"},
		{"zh", "zh-Hans"}, {"zh-Hans", "zh-Hans"}, {"zh-Hans-CN", "zh-Hans"},
		{"zh-CN", "zh-Hans"}, {"zh-SG", "zh-Hans"},
		{"zh-Hant", "en"}, {"zh-Hant-CN", "en"}, {"zh-TW", "en"}, {"zh-HK", "en"}, {"zh-MO", "en"},
		{"ja", "en"}, {"ko", "en"},
		{"zh-TW,en;q=0.8", "en"}, {"zh-Hant,de;q=0.8", "de"},
		{"ja,de;q=0.8", "de"}, {"ko,fr-CA;q=0.8", "fr-CA"},
		{"de;q=0.4,pl;q=0.7,zh-Hans;q=0.9", "zh-Hans"}, {"uk;q=0.8,ru;q=0.8", "uk"},
		{"en;q=0.4,fr-CA;q=0.9", "fr-CA"}, {"fr;q=0,en;q=1", "en"},
		{"fr;q=bogus", "en"}, {"fr;q=NaN", "en"}, {"de,pt;q=1.01", "en"},
		{"pl,de_DE", "en"}, {"uk,ru;q=0.8;unknown", "en"}, {"de,", "en"},
	} {
		t.Run("browser/"+tc.accept, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/login", nil)
			r.Header.Set("Accept-Language", tc.accept)
			if got := requestLocale(r); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
	for _, alias := range []string{"invalid", "fr-ca", "fr", "de-DE", "pl-PL", "pt", "es-MX", "uk-UA", "ru-RU", "zh", "zh-CN", "zh-hans", "zh-Hant", "ja", "ko"} {
		t.Run("invalid cookie/"+alias, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/login", nil)
			r.Header.Set("Accept-Language", "fr-CA")
			r.AddCookie(&http.Cookie{Name: languageCookie, Value: alias})
			if got := requestLocale(r); got != "en" || supportedLocale(alias) {
				t.Fatalf("invalid cookie %q: got %s", alias, got)
			}
		})
	}
}

func TestLoginFirstRenderLocaleAndActionOrder(t *testing.T) {
	app, err := New(newAuthFlowAPI(), Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, locale := range supportedLocales {
		t.Run(locale, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/login?oauth_request=example", nil)
			request.Header.Set("Accept-Language", locale)
			response := httptest.NewRecorder()
			app.Handler().ServeHTTP(response, request)
			body := response.Body.String()
			if response.Code != http.StatusOK || !strings.Contains(body, `<html lang="`+locale+`">`) {
				t.Fatalf("unexpected login response: status %d or wrong HTML language", response.Code)
			}
			for _, field := range []struct{ key, suffix string }{
				{"ui.e3b89e9d33f8", `<input name="username"`},
				{"ui.e7cf3ef4f17c", `<input type="password" name="password"`},
			} {
				label := catalogs[locale][field.key]
				if label == "" || !strings.Contains(body, "<label>"+html.EscapeString(label)+field.suffix) {
					t.Fatalf("missing translated label for %s", field.suffix)
				}
			}
			start := strings.Index(body, `class="login-actions"`)
			if start < 0 {
				t.Fatal("missing login action lane")
			}
			lane := strings.SplitN(body[start:], "</div>", 2)[0]
			signIn := catalogs[locale]["ui.bfd402b2f6f3"]
			submit := strings.Index(lane, `<button type="submit">`+html.EscapeString(signIn)+`</button>`)
			globe := strings.Index(lane, "data-language-open")
			if signIn == "" || submit < 0 || globe < 0 || globe > submit {
				t.Fatal("Globe must precede the translated Sign in")
			}
			if !strings.Contains(lane[:submit], `aria-label="`+html.EscapeString(catalogs[locale]["language.choose"])+`"`) ||
				!strings.Contains(lane[:submit], "<svg ") || strings.Contains(lane, "🌐") {
				t.Fatal("missing inline globe or its translated label")
			}
			assertLanguageDialog(t, body, locale, "/login?oauth_request=example")
		})
	}
}

// Inspect the rendered dialog in every locale alongside the existing login
// action-order contract; preference persistence and safe returns are tested below.
func assertLanguageDialog(t *testing.T, body, locale, returnPath string) {
	t.Helper()
	dialog := regexp.MustCompile(`(?s)<dialog[^>]*id="language-picker"[^>]*>.*?</dialog>`).FindString(body)
	for _, want := range []string{
		`class="system-action-dialog language-dialog"`,
		`aria-labelledby="language-heading"`,
		`<div class="system-action-dialog-content">`,
		`<h2 id="language-heading" class="language-dialog-title system-action-dialog-title"><span class="eyebrow">JUSTVOXEL</span><span aria-hidden="true">·</span><span>` + html.EscapeString(catalogs[locale]["language.choose"]) + `</span></h2>`,
		`<form method="post" action="/language">`,
		`<input type="hidden" name="return" value="` + html.EscapeString(returnPath) + `">`,
		`<select name="language" aria-label="` + html.EscapeString(catalogs[locale]["language.choose"]) + `">`,
	} {
		if !strings.Contains(dialog, want) {
			t.Fatalf("language dialog missing %q", want)
		}
	}
	if strings.Count(dialog, "<select ") != 1 || strings.Count(dialog, "<option ") != 9 || strings.Count(dialog, " selected") != 1 {
		t.Fatal("language picker must have one select, nine options and one selected locale")
	}
	groups := regexp.MustCompile(`(?s)<optgroup label="([^"]*)">(.*?)</optgroup>`).FindAllStringSubmatch(dialog, -1)
	wantGroups := []struct {
		key, english string
		options      []languageOption
	}{
		{"language.latin", "Latin", []languageOption{
			{"en", "English"}, {"fr-CA", "Français (Canada)"}, {"de", "Deutsch"},
			{"pl", "Polski"}, {"pt-BR", "Português (Brasil)"}, {"es-419", "Español (Latinoamérica)"},
		}},
		{"language.cyrillic", "Cyrillic", []languageOption{{"uk", "Українська"}, {"ru", "Русский"}}},
		{"language.eastasian", "East Asian", []languageOption{{"zh-Hans", "简体中文"}}},
	}
	if len(groups) != len(wantGroups) || strings.Count(dialog, "<optgroup") != len(wantGroups) {
		t.Fatal("language picker must have exactly three optgroups")
	}
	optionPattern := regexp.MustCompile(`<option [^>]*>[^<]*</option>`)
	for i, group := range wantGroups {
		if catalogs["en"][group.key] != group.english || groups[i][1] != html.EscapeString(catalogs[locale][group.key]) {
			t.Fatalf("%s: wrong label for %s: %q", locale, group.key, groups[i][1])
		}
		options := optionPattern.FindAllString(groups[i][2], -1)
		if len(options) != len(group.options) {
			t.Fatalf("%s: wrong number of options in %s", locale, group.key)
		}
		for j, option := range group.options {
			selected := ""
			if option.Locale == locale {
				selected = " selected"
			}
			want := `<option value="` + option.Locale + `" lang="` + option.Locale + `"` + selected + `>` + option.Name + `</option>`
			if options[j] != want {
				t.Fatalf("language option got %q, want %q", options[j], want)
			}
		}
	}
	for _, excluded := range []string{`value="ja"`, `value="ko"`, `value="zh-Hant"`, "日本語", "한국어", "Japanese", "Korean", "繁體中文"} {
		if strings.Contains(dialog, excluded) {
			t.Fatalf("unsupported language in dialog: %q", excluded)
		}
	}
	actions := regexp.MustCompile(`(?s)<div class="action-row">(.*?)</div>`).FindString(dialog)
	cancel := `<button class="secondary" type="button" data-language-close>` + html.EscapeString(catalogs[locale]["ui.19766ed6ccb2"]) + `</button>`
	apply := `<button type="submit">` + html.EscapeString(catalogs[locale]["ui.31e392d1c037"]) + `</button>`
	if !strings.Contains(actions, cancel) || !strings.Contains(actions, apply) || strings.Index(actions, cancel) > strings.Index(actions, apply) {
		t.Fatal("language actions must be secondary Cancel followed by primary Apply")
	}
	if strings.Count(dialog, "<button ") != 2 || strings.Count(dialog, `type="submit"`) != 1 || regexp.MustCompile(`<button[^>]*name="language"`).MatchString(dialog) {
		t.Fatal("only Apply may submit; language options must not be buttons")
	}
	if strings.Index(dialog, "</select>") > strings.Index(dialog, actions) || !strings.Contains(dialog, actions+"\n    </form>") {
		t.Fatal("language actions must be at the bottom of the form")
	}
	if strings.Contains(dialog, "onchange") || strings.Contains(dialog, "oninput") {
		t.Fatal("language selection must wait for Apply")
	}
}

func TestLanguageDialogUsesCompactSharedStyles(t *testing.T) {
	data, err := assets.ReadFile("static/app.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(data)
	for _, want := range []string{
		`.language-dialog{width:max-content;max-width:calc(100% - 2rem)}`,
		`.system-action-dialog{width:min(470px,calc(100% - 2rem));padding:0;`,
		`.system-action-dialog-content{padding:1.4rem}`,
		`.system-action-dialog .action-row{justify-content:flex-end}`,
		`.system-action-dialog .system-action-dialog-title{display:flex;flex-wrap:wrap;align-items:baseline;gap:.32rem;margin:0;font-size:.98rem;line-height:normal}`,
		`.system-action-dialog-title .eyebrow{margin:0;font-size:.68rem;font-weight:400;white-space:nowrap}`,
		`.language-dialog-field{margin-top:1rem}`,
		`.language-dialog select{width:100%;max-width:100%;margin-top:0}`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing compact language dialog style: %s", want)
		}
	}
	// Language actions inherit width, min-width, min-height, padding, font size,
	// and right alignment from the shared button and action-row styles.
	for _, rule := range regexp.MustCompile(`([^{}]+)\{([^{}]*)\}`).FindAllStringSubmatch(css, -1) {
		if !strings.Contains(rule[1], ".language-dialog") {
			continue
		}
		if strings.Contains(rule[1], "button") || strings.Contains(rule[1], ".secondary") || strings.Contains(rule[1], ".action-row") {
			t.Fatalf("language actions must use shared button and action-row styles: %s", rule[0])
		}
	}
}

func TestLanguageDialogWaitsForApply(t *testing.T) {
	script, err := assets.ReadFile("static/i18n.js")
	if err != nil {
		t.Fatal(err)
	}
	program := `
const assert = require("node:assert/strict");
const vm = require("node:vm");
const listeners = new Map();
let submissions = 0, opens = 0, closes = 0;
const field = {value: "/login"};
const select = {value: "en", closest: () => null};
const form = {submit() { submissions++; }, requestSubmit() { submissions++; }};
const dialog = {
  querySelector(selector) {
    if (selector === '[name="return"]') return field;
    if (selector === 'select[name="language"]') return select;
    if (selector === 'form') return form;
    throw new Error("unexpected selector: " + selector);
  },
  showModal() { opens++; }, close() { closes++; }
};
const context = {
  justVoxelLanguage: {messages: {}, source: {}},
  location: {pathname: "/", search: "?workspace=backups&server=jv-family1"},
  document: {
    addEventListener(type, callback) {
      if (!listeners.has(type)) listeners.set(type, []);
      listeners.get(type).push(callback);
    },
    querySelector(selector) { assert.equal(selector, '[data-language-dialog]'); return dialog; }
  }
};
vm.runInNewContext(` + strconv.Quote(string(script)) + `, context);
function dispatch(type, target) {
  for (const callback of listeners.get(type) || []) callback({target});
}
dispatch('click', {closest: selector => selector === '[data-language-open]' ? {} : null});
assert.equal(opens, 1);
assert.equal(field.value, "/?workspace=backups&server=jv-family1");
select.value = "fr-CA";
dispatch('input', select); dispatch('change', select); dispatch('click', select);
assert.equal(submissions, 0, "changing language must wait for Apply");
assert.equal(closes, 0);
dispatch('click', {closest: selector => selector === '[data-language-close]' ? {} : selector === 'dialog' ? dialog : null});
assert.equal(closes, 1);
assert.equal(submissions, 0, "Cancel must not submit");
`
	command := nodeTestCommand()
	command.Stdin = strings.NewReader(program)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("language dialog behavior: %v\n%s", err, output)
	}
}

func TestControlCenterLanguageRolesAndPowerCapabilities(t *testing.T) {
	app, err := New(newAuthFlowAPI(), Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"administrator", "operator", "viewer"} {
		response := httptest.NewRecorder()
		writer := &localeWriter{ResponseWriter: response, locale: "fr-CA", returnPath: "/?workspace=system"}
		if err := app.executeTemplate(writer, "app-header", struct {
			pageData
			Identity api.SessionInfo
		}{Identity: api.SessionInfo{Role: role}}); err != nil {
			t.Fatal(err)
		}
		body := response.Body.String()
		start := strings.Index(body, `class="control-session-actions"`)
		if start < 0 {
			t.Fatal("missing right-side action group")
		}
		lane := strings.SplitN(body[start:], "</div>", 2)[0]
		if strings.Index(lane, "data-language-open") < 0 || strings.Index(lane, `action="/logout"`) < 0 || strings.Index(lane, "data-language-open") > strings.Index(lane, `action="/logout"`) {
			t.Fatal("Globe must precede Logout")
		}
		if strings.Contains(lane, "nav-admin-only") || strings.Contains(lane, "data-primary-admin-only") {
			t.Fatal("language must be available to every role")
		}
		for _, control := range []string{`data-system-action="reboot"`, `data-system-action="firmware-reboot"`, `data-system-action="poweroff"`, `data-display-open`} {
			if !strings.Contains(body[:start], control) {
				t.Fatalf("lost power capability hook %s", control)
			}
		}
		firmware := regexp.MustCompile(`<button[^>]*data-system-action="firmware-reboot"[^>]*>`).FindString(body)
		display := regexp.MustCompile(`<button[^>]*data-display-open[^>]*>`).FindString(body)
		if !strings.Contains(firmware, "hidden disabled") || !strings.Contains(display, "hidden") {
			t.Fatal("hardware controls must await capability discovery")
		}
		if !strings.Contains(body[:start], `class="control-power-actions nav-admin-only" data-system-power`) {
			t.Fatal("power controls lost their administrator capability gate")
		}
		if !strings.Contains(body[:start], "data-power-indicators hidden") {
			t.Fatal("lost heading battery indicator")
		}
	}
}

func TestLanguagePreferenceBeforeLoginAndSafeReturn(t *testing.T) {
	for _, secure := range []bool{false, true} {
		app, err := New(newAuthFlowAPI(), Config{SecureCookies: secure})
		if err != nil {
			t.Fatal(err)
		}
		for _, target := range []string{"/login?oauth_request=abc", "/?workspace=backups&server=jv-family1", "https://evil.example/", "//evil.example/", "/%2f%2fevil.example/", "/%5cevil.example/", "/%252f%252fevil.example/", "/language", "/\r\nLocation: bad"} {
			form := url.Values{"language": {"fr-CA"}, "return": {target}}
			request := httptest.NewRequest("POST", "/language", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			app.Handler().ServeHTTP(response, request)
			if response.Code != http.StatusSeeOther {
				t.Fatalf("pre-login preference: %d", response.Code)
			}
			location := response.Header().Get("Location")
			if location != languageReturnPath(target) {
				t.Fatalf("unsafe return: %q", location)
			}
			cookies := response.Result().Cookies()
			if len(cookies) != 1 {
				t.Fatalf("unexpected cookies: %v", cookies)
			}
			cookie := cookies[0]
			if cookie.Name != languageCookie || cookie.Value != "fr-CA" || cookie.Path != "/" || cookie.SameSite != http.SameSiteLaxMode || cookie.Secure != secure || !cookie.HttpOnly || cookie.MaxAge < 86400 {
				t.Fatalf("bad language cookie: %#v", cookie)
			}
		}
		for _, locale := range supportedLocales {
			form := url.Values{"language": {locale}}
			request := httptest.NewRequest("POST", "/language", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			app.Handler().ServeHTTP(response, request)
			cookies := response.Result().Cookies()
			if response.Code != http.StatusSeeOther || len(cookies) != 1 || cookies[0].Value != locale {
				t.Fatalf("valid manual locale %q rejected", locale)
			}
		}
		for _, alias := range []string{"fr", "de-DE", "pt", "es-MX", "zh-CN", "zh-hans", "zh-Hant", "ja", "ko"} {
			form := url.Values{"language": {alias}}
			request := httptest.NewRequest("POST", "/language", strings.NewReader(form.Encode()))
			request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			response := httptest.NewRecorder()
			app.Handler().ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || len(response.Result().Cookies()) != 0 {
				t.Fatalf("invalid manual locale %q accepted", alias)
			}
		}
	}
}

func TestCatalogBrowserContractAndExternalBoundary(t *testing.T) {
	app, err := New(newAuthFlowAPI(), Config{})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/i18n.js", nil)
	r.AddCookie(&http.Cookie{Name: languageCookie, Value: "fr-CA"})
	response := httptest.NewRecorder()
	app.Handler().ServeHTTP(response, r)
	if !strings.Contains(response.Body.String(), `"locale":"fr-CA"`) || !strings.Contains(response.Body.String(), `"language.close":"Fermer"`) || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("browser catalog mismatch")
	}
	script, err := assets.ReadFile("static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"project.description", "project.title", "identity.version", "entry.description", "result.output"} {
		for _, function := range []string{"jvT(", "jvS(", "jvH("} {
			if bytes.Contains(script, []byte(function+field)) {
				t.Fatalf("external content translated: %s%s", function, field)
			}
		}
	}
	// Source lookup is restricted to authored presentation slots, never opaque
	// provider/user/runtime fields. Interpolating these fields into jvT is safe.
	entries, err := assets.ReadDir("static")
	if err != nil {
		t.Fatal(err)
	}
	opaqueSource := regexp.MustCompile(`jvS\(\s*(?:project|plugin|provider|entry|server|world|identity|result)\.(?:name|title|description|author|authors|username|filename|path|version|id|output|log)\b`)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".js") {
			continue
		}
		data, err := assets.ReadFile("static/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if match := opaqueSource.Find(data); match != nil {
			t.Fatalf("external source translation in %s: %s", entry.Name(), match)
		}
	}
	var body bytes.Buffer
	raw := `backup_English.zip <world> Français`
	if err := app.executeTemplate(&body, "login.html", pageData{Title: raw, Message: raw}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body.String(), html.EscapeString(raw)) {
		t.Fatal("opaque values changed")
	}
}

// Existing source contracts inspect structure and canonical English copy. Resolve
// catalog calls for those inspections, while leaving all control flow intact.
func englishPresentationAsset(path string) ([]byte, error) {
	data, err := assets.ReadFile(path)
	if err != nil {
		return nil, err
	}
	source := string(data)
	var missing error
	message := func(key string) string {
		value, ok := catalogs["en"][key]
		if !ok {
			missing = fmt.Errorf("unknown English catalog key: %s", key)
		}
		return value
	}
	if strings.HasSuffix(path, ".html") {
		source = regexp.MustCompile(`\{\{t "([^"]+)" "([^"]+)" (\.[A-Za-z0-9.]+|\(ts \.[A-Za-z0-9.]+\))\}\}`).ReplaceAllStringFunc(source, func(call string) string {
			parts := regexp.MustCompile(`^\{\{t "([^"]+)" "([^"]+)" (.*)\}\}$`).FindStringSubmatch(call)
			expression := strings.TrimSuffix(strings.TrimPrefix(parts[3], "(ts "), ")")
			return strings.ReplaceAll(html.EscapeString(message(parts[1])), "{"+parts[2]+"}", "{{"+expression+"}}")
		})
		source = regexp.MustCompile(`\{\{t "([^"]+)"\}\}`).ReplaceAllStringFunc(source, func(call string) string {
			key := strings.Split(call, `"`)[1]
			return html.EscapeString(message(key))
		})
		source = regexp.MustCompile(`\{\{ts ([^}]+)\}\}`).ReplaceAllString(source, `{{$1}}`)
	} else if strings.HasSuffix(path, ".js") {
		source = regexp.MustCompile(`\$\{jvH\("([^"]+)"\)\}`).ReplaceAllStringFunc(source, func(call string) string {
			key := strings.Split(call, `"`)[1]
			return html.EscapeString(message(key))
		})
		singleQuotes := strings.Count(source, "'") > strings.Count(source, `"`)
		source = regexp.MustCompile(`jv[TE]\("([^"]+)"\)`).ReplaceAllStringFunc(source, func(call string) string {
			key := strings.Split(call, `"`)[1]
			value := message(key)
			encoded := strconv.Quote(value)
			if singleQuotes {
				return "'" + strings.ReplaceAll(strings.ReplaceAll(value, "\\", "\\\\"), "'", "\\'") + "'"
			}
			return encoded
		})
		// Preserve exact canonical template-string contracts for interpolated copy.
		// Generated calls use one object of parenthesized named expressions.
		source = regexp.MustCompile(`jvT\("([^"]+)", \{([^\n{}]*)\}\)`).ReplaceAllStringFunc(source, func(call string) string {
			parts := regexp.MustCompile(`^jvT\("([^"]+)", \{(.*)\}\)$`).FindStringSubmatch(call)
			values := regexp.MustCompile(`([A-Za-z][A-Za-z0-9_]*): \((.*?)\)(?:, |$)`).FindAllStringSubmatch(parts[2], -1)
			replacements := make([]string, 0, len(values)*2)
			for _, value := range values {
				replacements = append(replacements, "{"+value[1]+"}", "${"+value[2]+"}")
			}
			if len(replacements) == 0 {
				return call
			}
			text := strings.ReplaceAll(strings.ReplaceAll(message(parts[1]), "\\", "\\\\"), "`", "\\`")
			return "`" + strings.NewReplacer(replacements...).Replace(text) + "`"
		})
	}
	return []byte(source), missing
}

func TestTemplateAndBrowserCatalogReferences(t *testing.T) {
	keyPattern := regexp.MustCompile(`(?:\bt |jv[THSC]\()"(ui\.[a-f0-9]+|language\.[a-z]+|two_factor\.[a-z_]+)"`)
	for _, directory := range []string{"templates", "static"} {
		entries, err := assets.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !(strings.HasSuffix(entry.Name(), ".html") || strings.HasSuffix(entry.Name(), ".js")) {
				continue
			}
			data, err := assets.ReadFile(directory + "/" + entry.Name())
			if err != nil {
				t.Fatal(err)
			}
			for _, match := range keyPattern.FindAllSubmatch(data, -1) {
				key := string(match[1])
				if _, ok := catalogs["en"][key]; !ok {
					t.Errorf("%s references unknown English key %s", entry.Name(), key)
				}
			}
		}
	}
}

func TestMachineGatewayContractsRemainEnglishAcrossLocales(t *testing.T) {
	for _, locale := range supportedLocales {
		t.Run(locale, func(t *testing.T) {
			app, client := newGatewayTestApp(t, 4)
			for _, path := range []string{"/api/v1/openapi.json", "/.well-known/oauth-protected-resource/mcp"} {
				request := externalGatewayRequest("GET", "https://voxel.example"+path, "")
				request.Header.Set("Accept-Language", locale)
				request.AddCookie(&http.Cookie{Name: languageCookie, Value: locale})
				response := httptest.NewRecorder()
				app.Handler().ServeHTTP(response, request)
				if response.Code != 200 {
					t.Fatalf("%s: %d %s", path, response.Code, response.Body.String())
				}
				var payload map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
					t.Fatal(err)
				}
				if strings.HasSuffix(path, "openapi.json") {
					info := payload["info"].(map[string]any)
					if info["title"] != "JustVoxel AI & API Connections" {
						t.Fatal(info)
					}
					operation := payload["paths"].(map[string]any)["/api/v1/status"].(map[string]any)["get"].(map[string]any)
					responses := operation["responses"].(map[string]any)
					for code, want := range map[string]string{"200": "Sanitized appliance result", "202": "Accepted operation; retain its ID and inspect status", "400": "Invalid request", "401": "Invalid, disabled, revoked or expired credential", "403": "Current permission or instance scope denied", "409": "Safety check, confirmation, recovery or operation conflict", "429": "Operator allowance or cooldown"} {
						if responses[code].(map[string]any)["description"] != want {
							t.Fatalf("%s description changed: %v", code, responses[code])
						}
					}
				} else if payload["resource_name"] != "JustVoxel AI & API Connections" {
					t.Fatal(payload)
				}
			}
			response := httptest.NewRecorder()
			writer := &localeWriter{ResponseWriter: response, locale: locale}
			app.gatewayUnauthorized(writer, httptest.NewRequest("GET", "https://voxel.example/mcp", nil))
			var payload struct {
				Error struct{ Code, Message string }
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if response.Code != 401 || payload.Error.Code != "authentication_required" || payload.Error.Message != "A valid enabled JustVoxel connection is required." {
				t.Fatal(response.Body.String())
			}
			for _, tc := range []struct{ arguments, token, want string }{
				{`{"unknown":true}`, "external-test-credential", "Invalid operation arguments."},
				{`{}`, "invalid", "Operation denied or unavailable. Refresh connection permissions and tools."},
			} {
				result, err := gatewayMCPExecute(context.Background(), client, tc.token, "appliance_status", json.RawMessage(tc.arguments))
				if err != nil || result == nil || !result.IsError || len(result.Content) != 1 {
					t.Fatalf("unexpected MCP result: %v %v", result, err)
				}
				if text, ok := result.Content[0].(*mcp.TextContent); !ok || text.Text != tc.want {
					t.Fatalf("MCP contract changed: %v", result.Content)
				}
			}
			oauth, _ := newLocalOAuthApp(t)
			for _, path := range []string{"/.well-known/oauth-protected-resource/mcp/" + testOAuthConnection} {
				request := httptest.NewRequest("GET", "https://voxel.example"+path, nil)
				request.Header.Set("Accept-Language", locale)
				request.AddCookie(&http.Cookie{Name: languageCookie, Value: locale})
				response := httptest.NewRecorder()
				oauth.Handler().ServeHTTP(response, request)
				var metadata map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &metadata); err != nil {
					t.Fatal(err)
				}
				if response.Code != 200 || metadata["resource_name"] != "JustVoxel AI & API Connections" {
					t.Fatal(response.Code, metadata)
				}
			}
			oauth.config.ExternalScheme = "http"
			response = httptest.NewRecorder()
			writer = &localeWriter{ResponseWriter: response, locale: locale}
			if oauth.oauthHTTPS(writer, httptest.NewRequest("GET", "http://voxel.example/oauth/authorize", nil)) {
				t.Fatal("HTTP OAuth accepted")
			}
			var failure map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &failure); err != nil {
				t.Fatal(err)
			}
			if response.Code != 400 || failure["error"] != "invalid_request" || failure["error_description"] != "JustVoxel OAuth requires HTTPS." {
				t.Fatal(failure)
			}
		})
	}
}

func TestConsentExternalValuesRemainOpaque(t *testing.T) {
	app, err := New(newAuthFlowAPI(), Config{})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	writer := &localeWriter{ResponseWriter: response, locale: "fr-CA"}
	// These deliberately collide with English presentation messages.
	data := struct{ Provider, Name, Level, Scope, ClientName, RequestID, CSRF string }{
		Provider: "Close", Name: "Plugins", ClientName: "Delete", Level: "2", Scope: "jv-example", RequestID: "id", CSRF: "csrf",
	}
	if err := app.executeTemplate(writer, "oauth-consent.html", data); err != nil {
		t.Fatal(err)
	}
	body := response.Body.String()
	for _, value := range []string{`<h2>Close `, `<dd>Plugins</dd>`, `<dd>Delete</dd>`, `<dd>jv-example</dd>`, `<html lang="fr-CA">`} {
		if !strings.Contains(body, value) {
			t.Fatalf("external value changed or locale missing: %q", value)
		}
	}
}

func TestLocalizedServerPresentationKeepsOpaqueValuesAndCanonicalState(t *testing.T) {
	app, err := New(newAuthFlowAPI(), Config{})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	writer := &localeWriter{ResponseWriter: response, locale: "fr-CA"}
	data := minecraftServersPageData{Configured: true, Servers: []minecraftServerSummary{{
		Name: "Plugins", ContextID: "jv-example1", Software: "Close", Version: "Delete", State: "Stopped", StateClass: "danger",
	}}}
	if err := app.executeTemplate(writer, "minecraft_servers.html", data); err != nil {
		t.Fatal(err)
	}
	body := response.Body.String()
	for _, want := range []string{`data-server-name="Plugins"`, `<h2>Plugins</h2>`, `<span>Close · Delete</span>`, `data-server-id="jv-example1"`, `data-server-state="Stopped"`, `>` + translate("fr-CA", sourceKeys["Stopped"]) + `</strong>`} {
		if !strings.Contains(body, want) {
			t.Fatalf("lost external value or canonical state: %s", want)
		}
	}
}
