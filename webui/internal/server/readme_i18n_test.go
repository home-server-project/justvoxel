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
	for _, locale := range supportedLocales {
		t.Run(locale, func(t *testing.T) {
			path := files[locale]
			data, err := os.ReadFile(filepath.Join(root, path))
			if err != nil {
				t.Fatal(err)
			}
			for _, paragraph := range paragraphs.FindAllString(string(data), -1) {
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
					return
				}
			}
			t.Fatal("missing centered navigator with links to all supported README languages")
		})
	}
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
