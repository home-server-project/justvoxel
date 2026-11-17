package main

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode"

	"golang.org/x/sys/unix"
)

// The two layouts must never be combined. These moves follow Paper's migration
// guide: https://docs.papermc.io/paper/migration/ (26.1 and pre-26.1 sections).
// All paths are relative to a confined instance data root, never API paths.
type switchWorldMove struct{ From, To string }

func switchWorldInfo(root *os.Root, path string) (os.FileInfo, error) {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i := range parts {
		info, err := root.Lstat(filepath.Join(parts[:i+1]...))
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, errInstanceRecovery
		}
		if i == len(parts)-1 {
			return info, nil
		}
		if !info.IsDir() {
			return nil, errInstanceRecovery
		}
	}
	return nil, errInstanceRecovery
}

func switchLevelName(root *os.Root) (string, error) {
	info, err := switchWorldInfo(root, "server.properties")
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return "", errInstanceRecovery
	}
	file, err := root.Open("server.properties")
	if err != nil {
		return "", err
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return "", errInstanceRecovery
	}
	name, found := "world", false
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimLeft(strings.TrimSuffix(line, "\r"), " \t")
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		// Reject escaped keys/values and continuations rather than guessing how
		// Java Properties would decode a path. Duplicate level names are ambiguous.
		key, value, ok := strings.Cut(line, "=")
		if strings.Contains(key, "\\") || strings.HasSuffix(line, "\\") {
			return "", errInstanceRecovery
		}
		if strings.TrimSpace(key) != "level-name" {
			if strings.HasPrefix(line, "level-name") {
				return "", errInstanceRecovery
			}
			continue
		}
		if !ok || found {
			return "", errInstanceRecovery
		}
		if strings.TrimRight(value, " \t") != value {
			return "", errInstanceRecovery
		}
		found, name = true, strings.TrimLeft(value, " \t")
	}
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, "/\\\x00\r\n:") || strings.ContainsFunc(name, unicode.IsControl) {
		return "", errInstanceRecovery
	}
	return name, nil
}

func inspectSwitchWorld(data, version string) ([]switchWorldMove, error) {
	modern := java25GameReleasePattern.MatchString(version)
	if !modern && !java21GameReleasePattern.MatchString(version) {
		return nil, errInstanceRecovery
	}
	root, err := os.OpenRoot(data)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	level, err := switchLevelName(root)
	if err != nil {
		return nil, err
	}
	info, err := switchWorldInfo(root, filepath.Join(level, "level.dat"))
	if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
		return nil, errInstanceRecovery
	}
	var moves []switchWorldMove
	if modern {
		for _, legacy := range []string{level + "_nether", level + "_the_end", filepath.Join(level, "DIM-1"), filepath.Join(level, "DIM1")} {
			if _, e := root.Lstat(legacy); !errors.Is(e, os.ErrNotExist) {
				return nil, errInstanceRecovery
			}
		}
		for _, name := range []string{"game_rules.dat", "scheduled_events.dat", "wandering_trader.dat", "weather.dat", "world_gen_settings.dat"} {
			moves = append(moves, switchWorldMove{filepath.Join(level, "dimensions", "minecraft", "overworld", "data", "minecraft", name), filepath.Join(level, "data", "minecraft", name)})
		}
	} else {
		if _, e := root.Lstat(filepath.Join(level, "dimensions", "minecraft")); !errors.Is(e, os.ErrNotExist) {
			return nil, errInstanceRecovery
		}
		moves = []switchWorldMove{{filepath.Join(level+"_nether", "DIM-1"), filepath.Join(level, "DIM-1")}, {filepath.Join(level+"_the_end", "DIM1"), filepath.Join(level, "DIM1")}}
	}
	for _, move := range moves {
		info, e := switchWorldInfo(root, move.From)
		if e != nil {
			return nil, e
		}
		if modern {
			if !info.Mode().IsRegular() || info.Size() == 0 {
				return nil, errInstanceRecovery
			}
		} else {
			if !info.IsDir() {
				return nil, errInstanceRecovery
			}
			// Do not move symlinked or special world data into a target runtime.
			count := 0
			if e = fs.WalkDir(root.FS(), move.From, func(_ string, entry fs.DirEntry, err error) error {
				count++
				if err != nil {
					return err
				}
				if count > 250000 || entry.Type()&os.ModeSymlink != 0 || (!entry.IsDir() && !entry.Type().IsRegular()) {
					return errInstanceRecovery
				}
				return nil
			}); e != nil {
				return nil, e
			}
		}
		if _, e = root.Lstat(move.To); !errors.Is(e, os.ErrNotExist) {
			return nil, errInstanceRecovery
		}
		// Existing destination ancestors must also be real directories.
		parent := filepath.Dir(move.To)
		for parent != "." {
			if info, e = switchWorldInfo(root, parent); e == nil && !info.IsDir() {
				return nil, errInstanceRecovery
			} else if e != nil && !errors.Is(e, os.ErrNotExist) {
				return nil, e
			}
			parent = filepath.Dir(parent)
		}
	}
	return moves, nil
}

// Called only on a stopped target, after durable cold-backup/rollback evidence.
// On any failure the caller restores through the existing archive restore path;
// no missing evidence, destination overwrites or partial moves are accepted.
func transformSwitchWorld(data, version string) error {
	moves, err := inspectSwitchWorld(data, version)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(data)
	if err != nil {
		return err
	}
	defer root.Close()
	for _, move := range moves {
		sourceParent, e := switchWorldInfo(root, filepath.Dir(move.From))
		if e != nil {
			return e
		}
		owner, ok := sourceParent.Sys().(*syscall.Stat_t)
		if !ok {
			return errInstanceRecovery
		}
		parts := strings.Split(filepath.Dir(move.To), string(filepath.Separator))
		for i := range parts {
			path := filepath.Join(parts[:i+1]...)
			if err = root.Mkdir(path, sourceParent.Mode().Perm()); err == nil {
				if err = root.Chown(path, int(owner.Uid), int(owner.Gid)); err != nil {
					return err
				}
			} else if !errors.Is(err, os.ErrExist) {
				return err
			}
			if info, e := switchWorldInfo(root, path); e != nil || !info.IsDir() {
				return errInstanceRecovery
			}
		}
		sourceDir, e1 := root.Open(filepath.Dir(move.From))
		if e1 != nil {
			return e1
		}
		targetDir, e2 := root.Open(filepath.Dir(move.To))
		if e2 != nil {
			sourceDir.Close()
			return e2
		}
		// Never replace a destination created since the stopped preflight.
		err = unix.Renameat2(int(sourceDir.Fd()), filepath.Base(move.From), int(targetDir.Fd()), filepath.Base(move.To), unix.RENAME_NOREPLACE)
		if err == nil {
			err = sourceDir.Sync()
		}
		if err == nil {
			err = targetDir.Sync()
		}
		sourceDir.Close()
		targetDir.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
