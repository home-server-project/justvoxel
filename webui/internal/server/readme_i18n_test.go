package server

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestPublicREADMELanguages(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	root := filepath.Join(filepath.Dir(source), "../../..")
	files := map[string]string{"en": "README.md"}
	for _, locale := range supportedLocales {
		if locale != "en" {
			files[locale] = filepath.Join("readme_i18n", "README_"+locale+".md")
		}
	}
	paragraphs := regexp.MustCompile(`(?s)<p align="center">(.*?)</p>`)
	const noticeMarker = "<!-- justvoxel-translation-notice -->"
	for _, locale := range supportedLocales {
		t.Run(locale, func(t *testing.T) {
			path := files[locale]
			data, err := os.ReadFile(filepath.Join(root, path))
			if err != nil {
				t.Fatal(err)
			}
			content := string(data)
			_, install, found := strings.Cut(content, "\n## 🚀 ")
			if !found {
				t.Fatal("missing Install JustVoxel section")
			}
			install, _, _ = strings.Cut(install, "\n## ")
			checkInstallMedia(t, install)

			if count := strings.Count(content, noticeMarker); count != 1 {
				t.Fatalf("found %d translation notice markers, want 1", count)
			}
			for _, bounds := range paragraphs.FindAllStringIndex(content, -1) {
				paragraph := content[bounds[0]:bounds[1]]
				complete := true
				for _, group := range languageGroups {
					for _, option := range group.Options {
						href, err := filepath.Rel(filepath.Dir(path), files[option.Locale])
						if err != nil {
							t.Fatal(err)
						}
						link := `<a href="` + filepath.ToSlash(href) + `">` + option.Name + `</a>`
						if !strings.Contains(paragraph, link) {
							complete = false
						}
					}
				}
				if complete {
					afterNavigator := strings.TrimLeft(content[bounds[1]:], "\n\r")
					afterMarker, found := strings.CutPrefix(afterNavigator, noticeMarker+"\n")
					if !found {
						t.Fatal("missing translation notice marker directly below centered navigator")
					}
					notice, _, _ := strings.Cut(strings.TrimLeft(afterMarker, "\n\r"), "\n\n")
					for _, url := range []string{
						"https://github.com/home-server-project/justvoxel/issues",
						"https://github.com/home-server-project/justvoxel/pulls",
					} {
						if !strings.Contains(notice, url) {
							t.Errorf("translation notice missing %s", url)
						}
					}
					return
				}
			}
			t.Fatal("missing centered navigator with links to all supported README languages")
		})
	}
	t.Run("installer", func(t *testing.T) {
		data, err := os.ReadFile(filepath.Join(root, "docs", "INSTALLER.md"))
		if err != nil {
			t.Fatal(err)
		}
		_, usbGuide, found := strings.Cut(string(data), "\n### Create a bootable USB\n")
		if !found {
			t.Fatal("missing Create a bootable USB heading")
		}
		usbGuide, _, _ = strings.Cut(usbGuide, "\n#")
		checkInstallMedia(t, usbGuide)
		if !strings.Contains(usbGuide, "in UEFI mode") {
			t.Error("missing guidance to boot the prepared USB in UEFI mode")
		}
	})

	if _, err := os.Stat(filepath.Join(root, "README-DEV.md")); err != nil {
		t.Fatal(err)
	}
	err := fs.WalkDir(os.DirFS(root), ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && entry.Name() == ".git" {
			return fs.SkipDir
		}
		if strings.HasPrefix(entry.Name(), "README-DEV") && strings.HasSuffix(entry.Name(), ".md") && path != "README-DEV.md" {
			t.Errorf("unexpected developer README %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func checkInstallMedia(t *testing.T, section string) {
	t.Helper()
	for _, required := range []string{
		"https://etcher.balena.io/",
		"https://rufus.ie/en/",
		"https://www.ventoy.net/en/index.html",
		"https://fedoraproject.org/workstation/download/",
		"`dd`",
	} {
		if !strings.Contains(section, required) {
			t.Errorf("installation-media guidance missing %s", required)
		}
	}
}
