package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const languageCookie = "jv_lang"

type languageOption struct {
	Locale, Name string
}

type languageGroup struct {
	Label   string
	Options []languageOption
}

var languageGroups = []languageGroup{
	{"language.latin", []languageOption{
		{"en", "English"},
		{"fr-CA", "Français (Canada)"},
		{"de", "Deutsch"},
		{"pl", "Polski"},
		{"pt-BR", "Português (Brasil)"},
		{"es-419", "Español (Latinoamérica)"},
	}},
	{"language.cyrillic", []languageOption{
		{"uk", "Українська"},
		{"ru", "Русский"},
	}},
	{"language.eastasian", []languageOption{
		{"zh-Hans", "简体中文"},
	}},
}

var supportedLocales = func() []string {
	var locales []string
	for _, group := range languageGroups {
		for _, option := range group.Options {
			locales = append(locales, option.Locale)
		}
	}
	return locales
}()

// English owns the message keys. Catalogs are embedded with the WebUI, never fetched.
var catalogs = loadCatalogs()
var sourceKeys = func() map[string]string {
	keys := make(map[string]string, len(catalogs["en"]))
	for key, value := range catalogs["en"] {
		keys[value] = key
	}
	return keys
}()
var printfPlaceholderPattern = regexp.MustCompile(`%(?:\[[0-9]+\])?[-+# 0-9.]*[sdqvf]`)
var placeholderPattern = regexp.MustCompile(`\{[A-Za-z][A-Za-z0-9_]*\}`)

func placeholders(s string) string {
	values := placeholderPattern.FindAllString(s, -1)
	sort.Strings(values)
	return strings.Join(values, "\x00") + "\x01" + strings.Join(printfPlaceholderPattern.FindAllString(s, -1), "\x00")
}

func validateCatalog(source, translated map[string]string) error {
	if len(source) != len(translated) {
		return fmt.Errorf("catalog key count differs")
	}
	for key, value := range source {
		other, ok := translated[key]
		if !ok || strings.TrimSpace(other) == "" {
			return fmt.Errorf("missing translation: %s", key)
		}
		if placeholders(value) != placeholders(other) {
			return fmt.Errorf("incompatible placeholders: %s", key)
		}
	}
	return nil
}

func loadCatalogs() map[string]map[string]string {
	result := make(map[string]map[string]string)
	for _, locale := range supportedLocales {
		data, err := assets.ReadFile("locales/" + locale + ".json")
		if err != nil {
			if locale == "en" {
				panic(err)
			}
			result[locale] = map[string]string{}
			continue
		}
		var catalog map[string]string
		if err := json.Unmarshal(data, &catalog); err != nil {
			if locale == "en" {
				panic(err)
			}
			catalog = map[string]string{}
		}
		result[locale] = catalog
	}
	// Bad individual translations fall back to English at lookup. Compatibility is
	// additionally enforced by source tests, so a bad translation cannot strand login.
	return result
}

func translate(locale, key string, args ...any) string {
	source, ok := catalogs["en"][key]
	if !ok {
		return key
	}
	value := translationValue(source, catalogs[locale][key])
	// Named pairs keep values opaque, including filenames, names and provider data.
	replacements := make([]string, 0, len(args))
	for i := 0; i+1 < len(args); i += 2 {
		replacements = append(replacements, "{"+fmt.Sprint(args[i])+"}", fmt.Sprint(args[i+1]))
	}
	if len(replacements) != 0 {
		value = strings.NewReplacer(replacements...).Replace(value)
	}
	return value
}

func supportedLocale(value string) bool {
	for _, locale := range supportedLocales {
		if value == locale {
			return true
		}
	}
	return false
}

func browserLocale(tag string) string {
	primary, _, _ := strings.Cut(tag, "-")
	// Chinese needs a script or region check before matching the primary language.
	if primary == "zh" && tag != "zh" && tag != "zh-cn" && tag != "zh-sg" &&
		tag != "zh-hans" && !strings.HasPrefix(tag, "zh-hans-") {
		return ""
	}
	for _, locale := range supportedLocales {
		language, _, _ := strings.Cut(locale, "-")
		if primary == language {
			return locale
		}
	}
	return ""
}

func requestLocale(r *http.Request) string {
	if cookie, err := r.Cookie(languageCookie); err == nil {
		if supportedLocale(cookie.Value) {
			return cookie.Value
		}
		return "en"
	}
	type preference struct {
		tag     string
		quality float64
	}
	var preferences []preference
	header := r.Header.Get("Accept-Language")
	if len(header) > 4096 {
		return "en"
	}
	for _, item := range strings.Split(header, ",") {
		fields := strings.Split(strings.TrimSpace(item), ";")
		tag := strings.ToLower(strings.TrimSpace(fields[0]))
		if !languageTagPattern.MatchString(tag) {
			return "en"
		}
		quality := 1.0
		if len(fields) > 2 {
			return "en"
		}
		if len(fields) == 2 {
			q := strings.TrimSpace(fields[1])
			if !strings.HasPrefix(q, "q=") {
				return "en"
			}
			var err error
			quality, err = strconv.ParseFloat(strings.TrimPrefix(q, "q="), 64)
			if err != nil || quality < 0 || quality > 1 || !qualityPattern.MatchString(strings.TrimPrefix(q, "q=")) {
				return "en"
			}
		}
		if quality > 0 {
			preferences = append(preferences, preference{tag, quality})
		}
	}
	sort.SliceStable(preferences, func(i, j int) bool { return preferences[i].quality > preferences[j].quality })
	for _, pref := range preferences {
		if locale := browserLocale(pref.tag); locale != "" {
			return locale
		}
	}
	return "en"
}

var languageTagPattern = regexp.MustCompile(`^(?:[a-z]{1,8}(?:-[a-z0-9]{1,8})*|\*)$`)
var qualityPattern = regexp.MustCompile(`^(?:0(?:\.[0-9]{0,3})?|1(?:\.0{0,3})?)$`)

// Reject authorities, encoded redirects, backslashes and control characters.
func languageReturnPath(value string) string {
	u, err := url.Parse(value)
	if err != nil || u.IsAbs() || u.Host != "" || u.Opaque != "" || !strings.HasPrefix(u.Path, "/") || strings.HasPrefix(u.Path, "//") {
		return "/login"
	}
	if strings.ContainsAny(u.Path, "\\\r\n\x00") || strings.ContainsAny(value, "\\\r\n\x00") || strings.Contains(u.Path, "%") {
		return "/login"
	}
	if u.Path == "/language" {
		return "/login"
	}
	u.Fragment = ""
	return u.String()
}

func (a *App) languagePreference(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if err := r.ParseForm(); err != nil {
		http.Error(w, ui(w, "ui.8daf3a877523"), http.StatusBadRequest)
		return
	}
	locale := r.PostForm.Get("language")
	if !supportedLocale(locale) {
		http.Error(w, ui(w, "ui.1c339b78354a"), http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: languageCookie, Value: locale, Path: "/", HttpOnly: true, Secure: a.config.SecureCookies, SameSite: http.SameSiteLaxMode, MaxAge: 365 * 24 * 60 * 60, Expires: time.Now().Add(365 * 24 * time.Hour)})
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, languageReturnPath(r.PostForm.Get("return")), http.StatusSeeOther)
}

type localeWriter struct {
	http.ResponseWriter
	locale, returnPath string
}

func (w *localeWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Select templates per response; never mutate shared template functions in a request.
func (a *App) executeTemplate(w io.Writer, name string, data any) error {
	locale, returnPath := "en", "/login"
	if response, ok := w.(*localeWriter); ok {
		locale, returnPath = response.locale, response.returnPath
	}
	view, err := a.templates.Clone()
	if err != nil {
		return err
	}
	view.Funcs(localeTemplateFuncs(locale, returnPath))
	return view.ExecuteTemplate(w, name, data)
}

func localeTemplateFuncs(locale, returnPath string) template.FuncMap {
	return template.FuncMap{
		"t": func(key string, args ...any) string { return translate(locale, key, args...) },
		"ts": func(value any) string {
			if value == nil {
				return ""
			}
			return translateSource(locale, fmt.Sprint(value))
		},
		"locale":         func() string { return locale },
		"languageGroups": func() []languageGroup { return languageGroups },
		"languageReturn": func() string { return returnPath },
	}
}

func uiText(key string, args ...any) string { return translate("en", key, args...) }
func ui(w http.ResponseWriter, key string, args ...any) string {
	locale := "en"
	if response, ok := w.(*localeWriter); ok {
		locale = response.locale
	}
	return translate(locale, key, args...)
}

func (a *App) languageScript(w http.ResponseWriter, r *http.Request) {
	locale := requestLocale(r)
	selected := make(map[string]string, len(catalogs["en"]))
	for key := range catalogs["en"] {
		selected[key] = translate(locale, key)
	}
	data, err := json.Marshal(struct {
		Locale   string            `json:"locale"`
		Messages map[string]string `json:"messages"`
		Source   map[string]string `json:"source"`
	}{locale, selected, catalogs["en"]})
	if err != nil {
		http.Error(w, ui(w, "ui.852519be0b7d"), 500)
		return
	}
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = fmt.Fprintf(w, "Object.defineProperty(globalThis, 'justVoxelLanguage', {value: Object.freeze(%s)});", data)
}

// Only call at an explicitly owned presentation slot; never use for provider
// names/descriptions, filenames, user names, logs, commands or console output.
func translateSource(locale, value string) string {
	if key, ok := sourceKeys[value]; ok {
		return translate(locale, key)
	}
	return value
}

func translationValue(source, value string) string {
	if strings.TrimSpace(value) == "" || placeholders(value) != placeholders(source) {
		return source
	}
	return value
}
