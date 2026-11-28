package main

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/sys/unix"
)

type switchExecutableIdentity struct {
	Directory, Filename string
}

type switchExecutableDigests struct {
	SHA512, SHA256 string
}

func openSwitchExecutableDirectory(data *os.File, kind string) (*os.File, error) {
	if kind != "plugins" && kind != "mods" {
		return nil, errInstanceRecovery
	}
	fd, err := unix.Openat(int(data.Fd()), kind, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), kind), nil
}

func hashSwitchExecutable(root *os.File, name string) (switchExecutableDigests, error) {
	jar, err := openPluginAt(root, name)
	if err != nil {
		return switchExecutableDigests{}, err
	}
	defer jar.Close()
	var before, after, entry unix.Stat_t
	if unix.Fstat(int(jar.Fd()), &before) != nil || before.Nlink != 1 {
		return switchExecutableDigests{}, errInstanceRecovery
	}
	h512, h256 := sha512.New(), sha256.New()
	n, err := io.Copy(io.MultiWriter(h512, h256), io.LimitReader(jar, pluginMaxJar+1))
	if err != nil || n != before.Size || n > pluginMaxJar ||
		unix.Fstat(int(jar.Fd()), &after) != nil || unix.Fstatat(int(root.Fd()), name, &entry, unix.AT_SYMLINK_NOFOLLOW) != nil ||
		entry.Dev != before.Dev || entry.Ino != before.Ino || before.Ctim != after.Ctim || before.Ctim != entry.Ctim {
		return switchExecutableDigests{}, errInstanceRecovery
	}
	return switchExecutableDigests{hex.EncodeToString(h512.Sum(nil)), hex.EncodeToString(h256.Sum(nil))}, nil
}

func readSwitchExecutableInventory(path string) (map[switchExecutableIdentity]switchExecutableDigests, error) {
	data, err := openPluginDataDirectory(path)
	if err != nil {
		return nil, err
	}
	defer data.Close()
	return scanSwitchExecutables(data)
}

func scanSwitchExecutables(data *os.File) (map[switchExecutableIdentity]switchExecutableDigests, error) {
	found := map[switchExecutableIdentity]switchExecutableDigests{}
	for _, kind := range []string{"plugins", "mods"} {
		dir, err := openSwitchExecutableDirectory(data, kind)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		entries, err := dir.ReadDir(10001)
		if err != nil && !errors.Is(err, io.EOF) || len(entries) > 10000 {
			dir.Close()
			return nil, errInstanceRecovery
		}
		for _, entry := range entries {
			if strings.EqualFold(filepath.Ext(entry.Name()), ".jar") {
				hash, _ := hashSwitchExecutable(dir, entry.Name())
				found[switchExecutableIdentity{kind, entry.Name()}] = hash
			}
		}
		dir.Close()
	}
	return found, nil
}

func (b *instanceRuntimeBackend) switchCloneContentRoot(ctx context.Context, e minecraftSwitchEvidence, archive string) (*os.File, error) {
	record, err := b.readRecord(e.CloneID)
	if err != nil || e.CloneRecord == nil || e.CloneID == e.Source || !switchCloneRecordMatches(record, *e.CloneRecord) ||
		record.Runtime != e.CloneRecord.Runtime || !record.Runtime.PreserveContent || b.verifyLive(record) != nil ||
		b.storage(ctx, "validate", record) != nil || b.deleteStopped(ctx, e.CloneID, false) != nil {
		return nil, errInstanceRecovery
	}
	journal, err := b.restoreJournal(e.CloneID)
	if err != nil || journal.Stage != "completed" || journal.Archive != archive || journal.OriginallyRunning || len(journal.Replacement) != 2 {
		return nil, errInstanceRecovery
	}
	raw, err := b.readPrivate(filepath.Join(b.root, e.CloneID, "config.json"))
	if err != nil || instanceHash(raw) != journal.ConfigSHA256 {
		return nil, errInstanceRecovery
	}
	data, err := openPluginDataDirectory(record.Storage.DataPath)
	if err != nil {
		return nil, err
	}
	var stat unix.Stat_t
	if unix.Fstat(int(data.Fd()), &stat) != nil || uint64(stat.Dev) != journal.Replacement[0] || stat.Ino != journal.Replacement[1] {
		data.Close()
		return nil, errInstanceRecovery
	}
	return data, nil
}

func applySwitchCloneContent(data *os.File, operation string, p minecraftSwitchPlan) (map[switchExecutableIdentity]switchExecutableDigests, string, error) {
	expected := map[switchExecutableIdentity]switchExecutableDigests{}
	excluded := map[switchExecutableIdentity]bool{}
	for _, item := range p.Content {
		id := switchExecutableIdentity{item.Directory, item.Filename}
		if (id.Directory != "plugins" && id.Directory != "mods") || !safePluginFilename(id.Filename) {
			return nil, "", errInstanceRecovery
		}
		switch item.CloneDisposition {
		case "retain", "exclude":
			if !pluginHashPattern.MatchString(item.SHA512) || !instanceSHA256Pattern.MatchString(item.SHA256) {
				return nil, "", errInstanceRecovery
			}
			expected[id] = switchExecutableDigests{item.SHA512, item.SHA256}
			if item.CloneDisposition == "exclude" {
				excluded[id] = true
			}
		case "absent":
			excluded[id] = true
		case "disabled":
			// Retain an existing disabled artifact and its ownership.
		default:
			return nil, "", errInstanceRecovery
		}
	}
	found, err := scanSwitchExecutables(data)
	manifest, raw, manifestErr := readPluginManifestAt(int(data.Fd()), ".justvoxel")
	if err != nil || manifestErr != nil || instanceHash(raw) != p.ManifestHash || !maps.Equal(expected, found) {
		return nil, "", instanceRuntimeFailure("Restored executable content or ownership differs from the reviewed source.")
	}
	for _, item := range p.Content {
		if item.CloneDisposition != "disabled" {
			continue
		}
		for _, entry := range manifest.Artifacts {
			if pluginTarget(entry.Target) == item.Directory && entry.Filename == item.Filename {
				disabled, err := openDisabledPluginDirectoryAt(data, false)
				if err != nil {
					return nil, "", errInstanceRecovery
				}
				err = verifyManagedPluginAt(disabled, entry)
				disabled.Close()
				if err != nil {
					return nil, "", errInstanceRecovery
				}
			}
		}
	}
	if len(excluded) == 0 {
		return expected, instanceHash(raw), nil
	}
	staging, err := openPluginOperationStagingAt(data, operation, true)
	if err != nil {
		return nil, "", err
	}
	defer staging.Close()
	for _, kind := range []string{"plugins", "mods"} {
		var names []string
		for id := range excluded {
			if id.Directory == kind && expected[id].SHA512 != "" {
				names = append(names, id.Filename)
			}
		}
		if len(names) == 0 {
			continue
		}
		slices.Sort(names)
		if err := moveSwitchCloneExclusions(data, staging, kind, names, expected); err != nil {
			return nil, "", err
		}
	}
	removed, changed := map[string]bool{}, false
	manifest.Artifacts = slices.DeleteFunc(manifest.Artifacts, func(entry pluginManifestEntry) bool {
		kind := pluginTarget(entry.Target)
		if !excluded[switchExecutableIdentity{kind, entry.Filename}] {
			return false
		}
		removed[contentManifestIdentity(entry)] = true
		changed = true
		return true
	})
	if changed {
		for i := range manifest.Artifacts {
			entry := &manifest.Artifacts[i]
			entry.RequiredBy = slices.DeleteFunc(entry.RequiredBy, func(parent string) bool {
				return removed[pluginTarget(entry.Target)+":"+pluginParentIdentity(parent)]
			})
		}
		if committed, err := commitPluginManifestAt(int(data.Fd()), raw, manifest, nil); err != nil || !committed {
			return nil, "", errors.Join(errInstanceRecovery, err)
		}
	}
	found, err = scanSwitchExecutables(data)
	_, updated, manifestErr := readPluginManifestAt(int(data.Fd()), ".justvoxel")
	if err != nil || manifestErr != nil || !maps.Equal(expected, found) || changed && instanceHash(updated) != instanceHash(mustSwitchJSON(manifest)) || !changed && instanceHash(updated) != p.ManifestHash {
		return nil, "", errInstanceRecovery
	}
	return expected, instanceHash(updated), nil
}

func moveSwitchCloneExclusions(data, staging *os.File, kind string, names []string, expected map[switchExecutableIdentity]switchExecutableDigests) error {
	from, err := openSwitchExecutableDirectory(data, kind)
	if err != nil {
		return err
	}
	defer from.Close()
	to, err := openPrivatePluginChild(staging, kind, true)
	if err != nil {
		return err
	}
	defer to.Close()
	for _, name := range names {
		id := switchExecutableIdentity{kind, name}
		hash, err := hashSwitchExecutable(from, name)
		if err != nil || hash != expected[id] ||
			unix.Renameat2(int(from.Fd()), name, int(to.Fd()), name, unix.RENAME_NOREPLACE) != nil {
			return errInstanceRecovery
		}
		moved, err := hashSwitchExecutable(to, name)
		if err != nil || moved != hash {
			return errInstanceRecovery
		}
		delete(expected, id)
	}
	return errors.Join(from.Sync(), to.Sync())
}

func verifySwitchCloneContent(path, manifestHash string, expected map[switchExecutableIdentity]switchExecutableDigests, installs []switchCrossplayInstall) (string, error) {
	data, err := openPluginDataDirectory(path)
	if err != nil {
		return "", err
	}
	defer data.Close()
	found, err := scanSwitchExecutables(data)
	_, manifest, manifestErr := readPluginManifestAt(int(data.Fd()), ".justvoxel")
	if err != nil || manifestErr != nil || instanceHash(manifest) != manifestHash {
		return "", errInstanceRecovery
	}
	replacements := map[switchExecutableIdentity]switchExecutableDigests{}
	for _, artifact := range installs {
		id := switchExecutableIdentity{artifact.Directory, artifact.Filename}
		hash := found[id]
		if (artifact.SHA256 == "" && artifact.SHA512 == "") ||
			(artifact.SHA256 != "" && hash.SHA256 != artifact.SHA256) ||
			(artifact.SHA512 != "" && hash.SHA512 != artifact.SHA512) {
			return "", errInstanceRecovery
		}
		delete(found, id)
		replacements[id] = hash
	}
	if !maps.Equal(expected, found) {
		return "", instanceRuntimeFailure("Clone executable content differs from the reviewed result.")
	}
	maps.Copy(found, replacements)
	return switchCloneContentFingerprint(found, manifestHash), nil
}

func switchCloneContentFingerprint(files map[switchExecutableIdentity]switchExecutableDigests, manifestHash string) string {
	entries := make([]string, 0, len(files))
	for id, hash := range files {
		entries = append(entries, id.Directory+"/"+id.Filename+":"+hash.SHA512)
	}
	slices.Sort(entries)
	return instanceHash(mustSwitchJSON(append(entries, manifestHash)))
}

func currentSwitchCloneContentFingerprint(path string) (string, error) {
	data, err := openPluginDataDirectory(path)
	if err != nil {
		return "", err
	}
	defer data.Close()
	files, err := scanSwitchExecutables(data)
	_, raw, manifestErr := readPluginManifestAt(int(data.Fd()), ".justvoxel")
	if err != nil || manifestErr != nil {
		return "", errors.Join(err, manifestErr)
	}
	return switchCloneContentFingerprint(files, instanceHash(raw)), nil
}
