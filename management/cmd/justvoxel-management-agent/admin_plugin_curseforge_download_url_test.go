package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestCurseForgeOfficialDownloadURLResolution(t *testing.T) {
	const artifactURL = "https://edge.forgecdn.net/files/123/456/example.jar"
	redirectPolicy := curseForgeClient.CheckRedirect
	for _, tc := range []struct {
		name, inline, body, invalid      string
		status                           int
		wantOK, wantFallback, wantUpload bool
	}{
		{name: "inline", inline: artifactURL, wantOK: true},
		{name: "official fallback", body: `{"data":"` + artifactURL + `"}`, wantOK: true, wantFallback: true},
		{name: "wrong host", body: `{"data":"https://attacker.invalid/files/123/456/example.jar"}`, wantFallback: true},
		{name: "host suffix", body: `{"data":"https://edge.forgecdn.net.evil/files/123/456/example.jar"}`, wantFallback: true},
		{name: "query", body: `{"data":"` + artifactURL + `?key=private-fixture-key"}`, wantFallback: true},
		{name: "http", body: `{"data":"http://edge.forgecdn.net/files/123/456/example.jar"}`, wantFallback: true},
		{name: "empty fallback", body: `{"data":""}`, wantFallback: true, wantOK: true, wantUpload: true},
		{name: "not found", status: http.StatusNotFound, body: "private-fixture-key private-response", wantFallback: true, wantOK: true, wantUpload: true},
		{name: "forbidden", status: http.StatusForbidden, body: "private-fixture-key private-response", wantFallback: true, wantOK: true, wantUpload: true},
		{name: "redirect", status: http.StatusFound, wantFallback: true},
		{name: "transport failure", invalid: "transport", wantFallback: true},
		{name: "wrong JSON type", body: `{"data":{"url":"` + artifactURL + `"}}`, wantFallback: true},
		{name: "trailing JSON", body: `{"data":"` + artifactURL + `"} {}`, wantFallback: true},
		{name: "oversized response", body: strings.Repeat(" ", (2<<20)+1), wantFallback: true},
		{name: "missing SHA1", invalid: "SHA1", body: `{"data":"` + artifactURL + `"}`},
		{name: "zero size", invalid: "size", body: `{"data":"` + artifactURL + `"}`},
		{name: "oversized file", invalid: "oversized", body: `{"data":"` + artifactURL + `"}`},
		{name: "invalid filename", invalid: "filename", body: `{"data":"` + artifactURL + `"}`},
		{name: "wrong project", invalid: "project", body: `{"data":"` + artifactURL + `"}`},
		{name: "wrong file", invalid: "file", body: `{"data":"` + artifactURL + `"}`},
		{name: "unsafe inline is rejected", inline: "https://attacker.invalid/example.jar", body: `{"data":"` + artifactURL + `"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, server := curseForgeCatalogFixture(t, "fabric")
			curseForgeClient.CheckRedirect = redirectPolicy
			base := curseForgeClient.Transport
			lookups := 0
			pluginCatalogClient.Transport = pluginTestTransport(func(r *http.Request) (*http.Response, error) {
				lookups++
				if !strings.HasPrefix(r.URL.Path, "/v2/version_file/") || r.Header.Get("x-api-key") != "" {
					t.Fatal("unsafe Modrinth lookup", r.URL)
				}
				return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader(""))}, nil
			})
			calls := 0
			curseForgeClient.Transport = pluginTestTransport(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet || r.URL.Scheme != "https" || r.URL.Host != "api.curseforge.com" || r.Header.Get("x-api-key") != "private-fixture-key" || strings.Contains(r.URL.String(), "private-fixture-key") || r.URL.User != nil {
					t.Fatal("catalog credential boundary violated")
				}
				if strings.HasSuffix(r.URL.Path, "/download-url") {
					calls++
					if r.URL.Path != "/v1/mods/10/files/20/download-url" || r.URL.RawQuery != "" {
						t.Fatal("fallback did not use exact validated identity")
					}
					if tc.invalid == "transport" {
						return nil, errors.New("private-fixture-key private-response")
					}
					status := tc.status
					if status == 0 {
						status = http.StatusOK
					}
					return &http.Response{StatusCode: status, Header: http.Header{"Location": {"https://attacker.invalid/stolen"}}, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
				}
				response, err := base.RoundTrip(r)
				if err != nil || r.URL.Path != "/v1/mods/10/files/20" {
					return response, err
				}
				defer response.Body.Close()
				var result struct {
					Data curseForgeFile `json:"data"`
				}
				if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
					t.Fatal(err)
				}
				result.Data.URL, result.Data.Filename = tc.inline, "example.jar"
				switch tc.invalid {
				case "SHA1":
					result.Data.Hashes = nil
				case "size":
					result.Data.Size = 0
				case "oversized":
					result.Data.Size = pluginMaxJar + 1
				case "filename":
					result.Data.Filename = "../example.jar"
				case "project":
					result.Data.ModID = 11
				case "file":
					result.Data.ID = 21
				}
				raw, err := json.Marshal(result)
				if err != nil {
					t.Fatal(err)
				}
				response.Body = io.NopCloser(bytes.NewReader(raw))
				return response, nil
			})
			page, err := discoverCurseForgePlugins(context.Background(), server, pluginDiscoverQuery{Limit: 8, Compatible: true, Release: true})
			if err != nil || len(page.Projects) != 1 || page.Projects[0].InstallAvailable || page.Projects[0].ReviewAvailable != (tc.invalid != "project" && tc.invalid != "file") {
				t.Fatal("unexpected Discover eligibility")
			}
			if page.Projects[0].ReviewAvailable && page.Projects[0].Compatibility != "Compatible Minecraft/loader release. Dedicated-server eligibility is checked during Review." {
				t.Fatal("dedicated-server compatibility wording changed")
			}
			wantCalls := 0
			if tc.wantFallback {
				wantCalls = 1
			}
			if calls != 0 {
				t.Fatal("Discover resolved an install artifact URL", calls)
			}
			// Planning must resolve independently of Discover, including direct requests.
			calls = 0
			plan, planErr := resolveCurseForgeInstallPlan(context.Background(), server, f.data, "10", "20")
			if (planErr == nil) != tc.wantOK || calls != wantCalls {
				t.Fatal("unexpected independent plan resolution", calls)
			}
			if planErr != nil && (strings.Contains(planErr.Error(), "private-fixture-key") || strings.Contains(planErr.Error(), "private-response")) {
				t.Fatal("private API failure exposed")
			}
			public, err := json.Marshal(struct {
				Page pluginCatalogPage
				Plan pluginInstallPlan
			}{page, plan})
			if err != nil || bytes.Contains(public, []byte("private-fixture-key")) || bytes.Contains(public, []byte("private-response")) {
				t.Fatal("private metadata exposed")
			}
			if !tc.wantOK {
				if len(plan.Artifacts) != 0 {
					t.Fatal("failed URL resolution synthesized an artifact")
				}
				return
			}
			if len(plan.Artifacts) != 1 || plan.Fingerprint != fingerprintPluginPlan(plan) {
				t.Fatal("resolved artifact evidence not bound to plan")
			}
			a := plan.Artifacts[0]
			if tc.wantUpload {
				if !a.UploadRequired || a.URL != "" || lookups != 1 {
					t.Fatal("unavailable artifact did not require exact upload", a, lookups)
				}
			} else if a.UploadRequired || a.URL != artifactURL || !validPluginDownloadMetadata(a) || lookups != 0 {
				t.Fatal("official artifact stopped using automatic download", a, lookups)
			}
			for _, field := range []string{"URL", "SHA1", "size", "filename", "project", "file"} {
				changed := plan
				changed.Artifacts = append([]pluginInstallArtifact(nil), plan.Artifacts...)
				switch field {
				case "URL":
					changed.Artifacts[0].URL = "https://mediafilez.forgecdn.net/files/123/456/example.jar"
				case "SHA1":
					changed.Artifacts[0].SHA1 = strings.Repeat("0", 40)
				case "size":
					changed.Artifacts[0].Size++
				case "filename":
					changed.Artifacts[0].Filename = "other.jar"
				case "project":
					changed.Artifacts[0].ProjectID = "11"
				case "file":
					changed.Artifacts[0].VersionID = "21"
				}
				if fingerprintPluginPlan(changed) == plan.Fingerprint {
					t.Fatal("resolved evidence omitted from fingerprint", field)
				}
			}
		})
	}
}

func TestCurseForgeDownloadURLKeyStaysOutOfOtherProviders(t *testing.T) {
	for _, provider := range []string{"Modrinth", "Hangar"} {
		t.Run(provider, func(t *testing.T) {
			curseForgeCatalogFixture(t, "paper")
			var resolve func() pluginInstallPlan
			if provider == "Modrinth" {
				f := newPluginInstallFixture(t)
				resolve = func() pluginInstallPlan { return f.plan(t) }
			} else {
				f := newHangarInstallFixture(t, "paper")
				resolve = func() pluginInstallPlan { return f.hangarPlan(t) }
			}
			catalogCalls, artifactCalls := 0, 0
			check := func(r *http.Request) {
				if r.Header.Get("x-api-key") != "" || strings.Contains(r.URL.String(), "private-fixture-key") {
					t.Fatal("CurseForge key reached another provider")
				}
			}
			catalog, download := pluginCatalogClient.Transport, pluginDownloadClient.Transport
			pluginCatalogClient.Transport = pluginTestTransport(func(r *http.Request) (*http.Response, error) {
				catalogCalls++
				check(r)
				return catalog.RoundTrip(r)
			})
			pluginDownloadClient.Transport = pluginTestTransport(func(r *http.Request) (*http.Response, error) {
				artifactCalls++
				check(r)
				return download.RoundTrip(r)
			})
			if err := reviewRootPluginArtifact(context.Background(), resolve()); err != nil {
				t.Fatal(err)
			}
			if catalogCalls == 0 || artifactCalls == 0 {
				t.Fatal("provider API and artifact boundaries were not both exercised")
			}
		})
	}
}

func TestCurseForgeEscapedArtifactFilename(t *testing.T) {
	const prefix = "https://edge.forgecdn.net/files/123/456/"
	for _, filename := range []string{"ageingspawners-1.0.0%2B26.3-fabric.jar", "ageingspawners-1.0.0+26.3-fabric.jar"} {
		if !safeCurseForgeArtifactURL(prefix + filename) {
			t.Error("legitimate filename rejected", filename)
		}
	}
	for _, raw := range []string{
		prefix + "dir%2Fmod.jar", prefix + "dir%2fmod.jar", prefix + "dir%5Cmod.jar",
		prefix + "%2e%2e%2fmod.jar", prefix + "%2E%2E%5cmod.jar",
		prefix + "../mod.jar", prefix + "%00mod.jar", prefix + "%ZZ.jar", prefix + "mod%.jar",
		prefix + "mod.jar?", prefix + "mod.jar?token=x", prefix + "mod.jar#", prefix + "mod.jar#x",
		"https://edge.forgecdn.net/%66iles/123/456/mod.jar",
		"https://edge.forgecdn.net/files/%31%32%33/456/mod.jar",
		"https://edge.forgecdn.net/files/123/%34%35%36/mod.jar",
		"https://edge.forgecdn.net/files/0/456/mod.jar",
		"https://edge.forgecdn.net/files/123/0/mod.jar",
		"https://user@edge.forgecdn.net/files/123/456/mod.jar",
		"https://edge.forgecdn.net.evil/files/123/456/mod.jar",
		"https://evilforgecdn.net/files/123/456/mod.jar",
		"https://127.0.0.1/files/123/456/mod.jar",
		"https://edge.forgecdn.net:444/files/123/456/mod.jar",
		"http://edge.forgecdn.net/files/123/456/mod.jar",
	} {
		if safeCurseForgeArtifactURL(raw) {
			t.Error("hostile artifact URL accepted", raw)
		}
	}
}

func TestCurseForgeExactModrinthAlternateIdentity(t *testing.T) {
	for _, state := range []string{"exact", "lookup failure", "invalid project", "invalid version", "game", "loader", "client_only", "singleplayer_only", "unknown environment", "hash", "missing hash", "unsafe URL", "filename", "size", "SHA512", "beta", "other file", "invalid input"} {
		t.Run(state, func(t *testing.T) {
			f, server := curseForgeCatalogFixture(t, "fabric")
			plan, err := resolveCurseForgeInstallPlan(context.Background(), server, f.data, "10", "20")
			if err != nil || len(plan.Artifacts) != 1 {
				t.Fatal(plan, err)
			}
			a := plan.Artifacts[0]
			file := pluginArtifactFile{Filename: "official.jar", URL: "https://cdn.modrinth.com/data/exact/version/official.jar", Size: a.Size}
			file.Hashes.SHA1, file.Hashes.SHA512 = a.SHA1, strings.Repeat("b", 128)
			v := pluginCatalogVersion{ProjectID: "DifferentOfficialProject", ID: "ExactVersion", Number: "1", Type: "release", Environment: "server_only", GameVersions: []string{server.MinecraftVersion}, Loaders: []string{"fabric"}, Files: []pluginArtifactFile{file}}
			switch state {
			case "invalid project":
				v.ProjectID = "../root"
			case "invalid version":
				v.ID = "../version"
			case "game":
				v.GameVersions = []string{"1.20.1"}
			case "loader":
				v.Loaders = []string{"forge"}
			case "client_only", "singleplayer_only":
				v.Environment = state
			case "unknown environment":
				v.Environment = ""
			case "hash":
				v.Files[0].Hashes.SHA1 = strings.Repeat("0", 40)
			case "missing hash":
				v.Files[0].Hashes.SHA1 = ""
			case "unsafe URL":
				v.Files[0].URL = "https://cdn.modrinth.com.evil/data/official.jar"
			case "filename":
				v.Files[0].Filename = "../official.jar"
			case "size":
				v.Files[0].Size++
			case "SHA512":
				v.Files[0].Hashes.SHA512 = ""
			case "beta":
				v.Type = "beta"
			case "other file":
				// A valid hash on a different unsafe file is not evidence for
				// the safe file whose hash does not match.
				v.Files[0].Hashes.SHA1 = strings.Repeat("0", 40)
				file.URL = "https://attacker.invalid/official.jar"
				v.Files = append(v.Files, file)
			}
			old := pluginCatalogClient
			t.Cleanup(func() { pluginCatalogClient = old })
			calls := 0
			pluginCatalogClient = &http.Client{Transport: pluginTestTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Scheme != "https" || r.URL.Host != "api.modrinth.com" || r.URL.Path != "/v2/version_file/"+a.SHA1 || r.URL.Query().Get("algorithm") != "sha1" || len(r.URL.Query()) != 1 || r.Header.Get("x-api-key") != "" {
					t.Fatal("alternate lookup guessed identity or crossed credential boundary", r.URL)
				}
				if state == "lookup failure" {
					return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader(""))}, nil
				}
				raw, err := json.Marshal(v)
				return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw))}, err
			})}
			input := a
			input.URL = ""
			if state == "invalid input" {
				input.SHA1 = "../not-a-hash"
			}
			alternate := exactCurseForgeModrinthVersion(context.Background(), input, server)
			if (alternate != nil) != (state == "exact") {
				t.Fatal("unproven alternate identity", alternate)
			}
			if state == "invalid input" {
				if calls != 0 {
					t.Fatal("invalid hash reached provider")
				}
			} else if calls != 1 {
				t.Fatal("lookup was not bounded", calls)
			}
			if alternate != nil && (alternate.ProjectID != v.ProjectID || alternate.ID != v.ID || len(alternate.Files) != 1 || alternate.Files[0].Hashes.SHA1 != a.SHA1) {
				t.Fatal("alternate did not preserve exact official identity", alternate)
			}
			// Alternate evidence, including lookup failure, never rewrites the
			// CurseForge plan, dependency resolver, fingerprint or ownership.
			if plan.Source != "CurseForge" || plan.Artifacts[0].Source != "CurseForge" || plan.Artifacts[0].ProjectID != "10" || plan.Artifacts[0].VersionID != "20" || plan.Fingerprint != fingerprintPluginPlan(plan) {
				t.Fatal("alternate corrupted authoritative install plan", plan)
			}
			if err := reviewRootPluginArtifact(context.Background(), plan); err != nil {
				t.Fatal("alternate blocked CurseForge review", err)
			}
			if journal := f.execute(t, plan, false); journal.State != operationSucceeded {
				t.Fatal(journal)
			}
			manifest, _, err := readPluginManifest(f.data)
			if err != nil || len(manifest.Artifacts) != 1 || manifest.Artifacts[0].Source != "CurseForge" || manifest.Artifacts[0].ProjectID != "10" || manifest.Artifacts[0].VersionID != "20" {
				t.Fatal("alternate corrupted ownership", manifest, err)
			}
		})
	}
}

func TestCurseForgeEscapedFilenameRedirect(t *testing.T) {
	for _, tc := range []struct {
		filename string
		want     bool
	}{
		{"ageingspawners-1.0.0%2B26.3-fabric.jar", true},
		{"dir%2fmod.jar", false},
		{"dir%5cmod.jar", false},
		{"%2e%2e%2fmod.jar", false},
	} {
		t.Run(tc.filename, func(t *testing.T) {
			_, server := curseForgeCatalogFixture(t, "fabric")
			file, err := exactCurseForgeFile(context.Background(), "10", "20")
			if err != nil {
				t.Fatal(err)
			}
			a, err := curseForgeArtifact(context.Background(), file, "10", server, curseForgeContext{Game: 1, Class: 6}, false)
			if err != nil {
				t.Fatal(err)
			}
			base := pluginDownloadClient.Transport
			calls := 0
			pluginDownloadClient.Transport = pluginTestTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return &http.Response{StatusCode: 302, Header: http.Header{"Location": {"https://edge.forgecdn.net/files/123/456/" + tc.filename}}, Body: io.NopCloser(strings.NewReader(""))}, nil
				}
				if !tc.want || r.URL.EscapedPath() != "/files/123/456/"+tc.filename {
					t.Fatal("unsafe redirect contacted or filename changed", r.URL)
				}
				return base.RoundTrip(r)
			})
			_, err = downloadVerifiedPluginArtifact(context.Background(), a, t.TempDir())
			if (err == nil) != tc.want || calls != map[bool]int{true: 2, false: 1}[tc.want] {
				t.Fatal("redirect did not use corrected artifact validation", calls, err)
			}
		})
	}
}

func restrictedFabricJar(t *testing.T, id, side string) []byte {
	t.Helper()
	var raw bytes.Buffer
	archive := zip.NewWriter(&raw)
	file, err := archive.Create("fabric.mod.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(file).Encode(map[string]any{"schemaVersion": 1, "id": id, "version": "1", "environment": side}); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return raw.Bytes()
}

func restrictedCurseForgeFixture(t *testing.T, rootJar, dependencyJar []byte, alternate bool) (*pluginInstallFixture, *curseForgeFile) {
	t.Helper()
	f, _ := curseForgeCatalogFixture(t, "fabric")
	root, err := exactCurseForgeFile(context.Background(), "10", "20")
	if err != nil {
		t.Fatal(err)
	}
	setBytes := func(file *curseForgeFile, raw []byte) {
		sum := sha1.Sum(raw)
		file.URL, file.Size, file.Hashes[0].Value = "", int64(len(raw)), hex.EncodeToString(sum[:])
	}
	dependency := root
	dependency.Hashes = slices.Clone(root.Hashes)
	setBytes(&root, rootJar)
	if dependencyJar != nil {
		root.Dependencies = []curseForgeDependency{{ModID: 11, Relation: 3}}
		dependency.ID, dependency.ModID, dependency.Filename, dependency.DisplayName = 21, 11, "dependency.jar", "Dependency release"
		setBytes(&dependency, dependencyJar)
	}

	jsonReply := func(payload any) (*http.Response, error) {
		raw, err := json.Marshal(payload)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw))}, err
	}
	base := curseForgeClient.Transport
	curseForgeClient.Transport = pluginTestTransport(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1/mods/10/files/20":
			return jsonReply(map[string]any{"data": root})
		case "/v1/mods/11":
			return jsonReply(map[string]any{"data": map[string]any{"id": 11, "gameId": 1, "classId": 6, "name": "Dependency", "isAvailable": true}})
		case "/v1/mods/11/files":
			return jsonReply(map[string]any{"data": []curseForgeFile{dependency}, "pagination": curseForgePagination{0, 25, 1, 1}})
		case "/v1/mods/11/files/21":
			return jsonReply(map[string]any{"data": dependency})
		case "/v1/mods/10/files/20/download-url", "/v1/mods/11/files/21/download-url":
			return &http.Response{StatusCode: 403, Body: io.NopCloser(strings.NewReader("private-response"))}, nil
		default:
			return base.RoundTrip(r)
		}
	})
	const mirrorURL = "https://cdn.modrinth.com/data/Alternate/versions/Exact/mirror.jar"
	pluginCatalogClient.Transport = pluginTestTransport(func(r *http.Request) (*http.Response, error) {
		hash := strings.TrimPrefix(r.URL.Path, "/v2/version_file/")
		if r.URL.Host != "api.modrinth.com" || hash == r.URL.Path || r.URL.Query().Get("algorithm") != "sha1" || r.Header.Get("x-api-key") != "" {
			t.Fatal("Modrinth lookup crossed the provider boundary", r.URL)
		}
		if !alternate || hash != root.Hashes[0].Value {
			return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader(""))}, nil
		}
		file := pluginArtifactFile{URL: mirrorURL, Filename: "mirror.jar", Size: root.Size}
		sum := sha512.Sum512(rootJar)
		file.Hashes.SHA1, file.Hashes.SHA512 = hash, hex.EncodeToString(sum[:])
		return jsonReply(pluginCatalogVersion{ID: "Exact", ProjectID: "Alternate", Number: "Mirror release", Type: "release", Environment: "server_only", GameVersions: []string{"1.21.8"}, Loaders: []string{"fabric"}, Files: []pluginArtifactFile{file}})
	})
	pluginDownloadClient.Transport = pluginTestTransport(func(r *http.Request) (*http.Response, error) {
		if !alternate || r.URL.String() != mirrorURL || r.Header.Get("x-api-key") != "" {
			t.Fatal("unexpected artifact transport or credential", r.URL)
		}
		return &http.Response{StatusCode: 200, ContentLength: int64(len(rootJar)), Body: io.NopCloser(bytes.NewReader(rootJar))}, nil
	})
	return f, &root
}

func reviewRestrictedCurseForge(t *testing.T, f *pluginInstallFixture) pluginInstallPlan {
	t.Helper()
	rr := httptest.NewRecorder()
	f.s.adminPluginInstallPlan(rr, pluginTestRequest(http.MethodPost, "/", `{"source":"CurseForge","project_id":"10","version_id":"20"}`))
	var plan pluginInstallPlan
	if rr.Code != http.StatusOK || json.Unmarshal(rr.Body.Bytes(), &plan) != nil || !plan.InstallAvailable || !operationFingerprintPattern.MatchString(plan.Fingerprint) {
		t.Fatal("review failed", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "private-response") || strings.Contains(rr.Body.String(), "cdn.modrinth.com") {
		t.Fatal("private provider data exposed", rr.Body.String())
	}
	assertNoCurseForgeInstallMutation(t, f)
	return plan
}

func assertNoCurseForgeInstallMutation(t *testing.T, f *pluginInstallFixture) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(f.data, "mods"))
	if err != nil || len(entries) != 0 || len(f.s.operations.operations) != 0 || len(f.commands) != 0 || f.backup {
		t.Fatal("Minecraft files or maintenance state changed", entries, f.commands, err)
	}
}

type reviewedJarPart struct {
	field, filename string
	raw             []byte
}

func applyRestrictedCurseForge(t *testing.T, f *pluginInstallFixture, fingerprint string, confirm, oversized bool, parts ...reviewedJarPart) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range map[string]string{"source": "CurseForge", "project_id": "10", "version_id": "20", "plan_fingerprint": fingerprint, "confirm_players": strconv.FormatBool(confirm)} {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatal(err)
		}
	}
	for _, part := range parts {
		file, err := writer.CreateFormFile(part.field, part.filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(part.raw); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	r := pluginTestRequest(http.MethodPost, "/", body.String())
	r.Header.Set("Content-Type", writer.FormDataContentType())
	if oversized {
		r.ContentLength = pluginInstallUploadLimit + 1
	}
	rr := httptest.NewRecorder()
	f.s.adminPluginInstall(rr, r)
	return rr
}

func runCurseForgeWorkerSynchronously(t *testing.T) *string {
	t.Helper()
	old := startPluginInstallWorker
	t.Cleanup(func() { startPluginInstallWorker = old })
	var operationID string
	startPluginInstallWorker = func(s *server, id string, plan pluginInstallPlan, confirm bool, actor session) {
		operationID = id
		if err := executePluginInstall(context.Background(), s, id, plan, confirm, actor); err != nil {
			t.Fatal(err)
		}
		if plan.stagedDirectory != "" {
			if _, err := os.Stat(plan.stagedDirectory); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("temporary staging was retained", err)
			}
		}
	}
	return &operationID
}

func assertCurseForgeInstallOwnership(t *testing.T, f *pluginInstallFixture, operationID string, wantDependencies bool) {
	t.Helper()
	current, err := f.s.operations.get(operationID)
	if err != nil || current.State != operationSucceeded || !f.backup {
		t.Fatal("normal install transaction did not complete", current, err)
	}
	manifest, _, err := readPluginManifest(f.data)
	wantCount := 1
	if wantDependencies {
		wantCount++
	}
	if err != nil || len(manifest.Artifacts) != wantCount {
		t.Fatal("missing ownership", manifest, err)
	}
	for _, entry := range manifest.Artifacts {
		if entry.Source != "CurseForge" || entry.Target != "mods" || !pluginHashPattern.MatchString(entry.SHA512) {
			t.Fatal("wrong ownership", entry)
		}
		if entry.Requested {
			if entry.ProjectID != "10" || entry.VersionID != "20" || entry.VersionNumber != "1" || entry.Filename != "root.jar" {
				t.Fatal("original release identity changed", entry)
			}
		} else if entry.ProjectID != "11" || entry.VersionID != "21" || !slices.Contains(entry.RequiredBy, pluginIdentity("CurseForge", "10")) {
			t.Fatal("dependency ownership changed", entry)
		}
	}
}

func TestCurseForgeRestrictedDownloadUsesExactModrinthBytes(t *testing.T) {
	f, _ := restrictedCurseForgeFixture(t, restrictedFabricJar(t, "rootmod", "*"), nil, true)
	plan := reviewRestrictedCurseForge(t, f)
	if len(plan.Artifacts) != 1 || plan.Artifacts[0].UploadRequired {
		t.Fatal("exact alternate should be automatic", plan)
	}
	id := runCurseForgeWorkerSynchronously(t)
	raw, _ := json.Marshal(pluginInstallApplyRequest{Source: "CurseForge", ProjectID: "10", VersionID: "20", PlanFingerprint: plan.Fingerprint})
	rr := httptest.NewRecorder()
	f.s.adminPluginInstall(rr, pluginTestRequest(http.MethodPost, "/", string(raw)))
	if rr.Code != http.StatusAccepted {
		t.Fatal(rr.Code, rr.Body.String())
	}
	assertCurseForgeInstallOwnership(t, f, *id, false)
}

func TestCurseForgeApplyRequiresExactUploadsForRootAndDependency(t *testing.T) {
	rootJar := restrictedFabricJar(t, "rootmod", "*")
	dependencyJar := restrictedFabricJar(t, "depmod", "*")
	f, _ := restrictedCurseForgeFixture(t, rootJar, dependencyJar, false)
	f.players = 1
	plan := reviewRestrictedCurseForge(t, f)
	if !plan.ConfirmationRequired || len(plan.Artifacts) != 2 {
		t.Fatal("missing required uploads or confirmation", plan)
	}
	for i, a := range plan.Artifacts {
		if !a.UploadRequired || a.Source != "CurseForge" || a.Size <= 0 || a.Filename == "" || i == 1 && (a.Name != "Dependency" || !slices.Contains(a.RequiredBy, pluginIdentity("CurseForge", "10"))) {
			t.Fatal("incomplete Review upload evidence", a)
		}
	}
	root := reviewedJarPart{"jar_10_20", "root.jar", rootJar}
	dependency := reviewedJarPart{"jar_11_21", "dependency.jar", dependencyJar}
	raw, _ := json.Marshal(pluginInstallApplyRequest{Source: "CurseForge", ProjectID: "10", VersionID: "20", PlanFingerprint: plan.Fingerprint, ConfirmPlayers: true})
	rr := httptest.NewRecorder()
	f.s.adminPluginInstall(rr, pluginTestRequest(http.MethodPost, "/", string(raw)))
	if rr.Code != http.StatusBadRequest {
		t.Fatal("JSON Apply bypassed the required uploads", rr.Code, rr.Body.String())
	}
	assertNoCurseForgeInstallMutation(t, f)
	if rr := applyRestrictedCurseForge(t, f, plan.Fingerprint, true, false, root); rr.Code != http.StatusBadRequest {
		t.Fatal("missing dependency accepted", rr.Code, rr.Body.String())
	}
	assertNoCurseForgeInstallMutation(t, f)
	if rr := applyRestrictedCurseForge(t, f, plan.Fingerprint, false, false, root, dependency); rr.Code != http.StatusConflict || !strings.Contains(rr.Body.String(), "confirmation_required") {
		t.Fatal("missing player confirmation accepted", rr.Code, rr.Body.String())
	}
	assertNoCurseForgeInstallMutation(t, f)
	id := runCurseForgeWorkerSynchronously(t)
	if rr := applyRestrictedCurseForge(t, f, plan.Fingerprint, true, false, root, dependency); rr.Code != http.StatusAccepted {
		t.Fatal(rr.Code, rr.Body.String())
	}
	assertCurseForgeInstallOwnership(t, f, *id, true)
}

func TestCurseForgeInvalidUploadsFailBeforeMaintenance(t *testing.T) {
	for _, name := range []string{"hash", "size", "filename", "path", "missing", "duplicate", "extra", "oversized", "malformed", "loader", "client-only", "metadata", "fingerprint"} {
		t.Run(name, func(t *testing.T) {
			raw := restrictedFabricJar(t, "rootmod", "*")
			switch name {
			case "malformed":
				raw = []byte("not a JAR")
			case "loader":
				raw = resetModJar(t, "forge")
			case "client-only":
				raw = restrictedFabricJar(t, "rootmod", "client")
			}
			f, evidence := restrictedCurseForgeFixture(t, raw, nil, false)
			plan := reviewRestrictedCurseForge(t, f)
			fingerprint := plan.Fingerprint
			parts := []reviewedJarPart{{"jar_10_20", "root.jar", slices.Clone(raw)}}
			switch name {
			case "hash":
				parts[0].raw[0] ^= 1
			case "size":
				parts[0].raw = parts[0].raw[:len(raw)-1]
			case "filename":
				parts[0].filename = "different.jar"
			case "path":
				parts[0].filename = "../root.jar"
			case "missing":
				parts = nil
			case "duplicate":
				parts = append(parts, parts[0])
			case "extra":
				parts = append(parts, reviewedJarPart{"jar_12_22", "extra.jar", raw})
			case "metadata":
				evidence.Hashes[0].Value = strings.Repeat("0", 40)
			case "fingerprint":
				fingerprint = "sha256:" + strings.Repeat("0", 64)
			}
			rr := applyRestrictedCurseForge(t, f, fingerprint, false, name == "oversized", parts...)
			want := http.StatusBadRequest
			if name == "metadata" || name == "fingerprint" {
				want = http.StatusConflict
			}
			if rr.Code != want {
				t.Fatal(rr.Code, rr.Body.String())
			}
			assertNoCurseForgeInstallMutation(t, f)
		})
	}
}

func TestCurseForgeRestrictedPluginCanOpenReview(t *testing.T) {
	f, server := curseForgeCatalogFixture(t, "paper")
	file, err := exactCurseForgeFile(context.Background(), "10", "20")
	if err != nil {
		t.Fatal(err)
	}
	file.URL = ""
	base := curseForgeClient.Transport
	curseForgeClient.Transport = pluginTestTransport(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/v1/mods/10/files/20":
			raw, err := json.Marshal(map[string]any{"data": file})
			return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(raw))}, err
		case "/v1/mods/10/files/20/download-url":
			return &http.Response{StatusCode: 403, Body: io.NopCloser(strings.NewReader(""))}, nil
		default:
			return base.RoundTrip(r)
		}
	})
	lookups := 0
	pluginCatalogClient.Transport = pluginTestTransport(func(*http.Request) (*http.Response, error) {
		lookups++
		return &http.Response{StatusCode: 404, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	page, err := discoverCurseForgePlugins(context.Background(), server, pluginDiscoverQuery{Limit: 8, Compatible: true, Release: true})
	if err != nil || len(page.Projects) != 1 || !page.Projects[0].ReviewAvailable || page.Projects[0].InstallAvailable || lookups != 0 {
		t.Fatal("restricted plugin should open Review without looking up alternate bytes in Discover", page, err)
	}
	rr := httptest.NewRecorder()
	f.s.adminPluginInstallPlan(rr, pluginTestRequest(http.MethodPost, "/", `{"source":"CurseForge","project_id":"10","version_id":"20"}`))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"upload_required":true`) || lookups != 1 || len(f.s.operations.operations) != 0 || len(f.commands) != 0 || f.backup {
		t.Fatal("restricted plugin Review did not request the exact JAR without maintenance", rr.Code, rr.Body.String())
	}
}
