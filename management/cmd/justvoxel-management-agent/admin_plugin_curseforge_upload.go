package main

import (
	"crypto/sha1"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const pluginInstallUploadLimit = pluginMaxDownload + (64 << 10)

type curseForgeUpload struct {
	filename, path string
	size           int64
}

// Each part is bound to one exact CurseForge project and file in the reviewed plan.
func curseForgeUploadField(a pluginInstallArtifact) string {
	return "jar_" + a.ProjectID + "_" + a.VersionID
}

func parseCurseForgeInstallUpload(w http.ResponseWriter, r *http.Request, directory string) (pluginInstallApplyRequest, map[string]curseForgeUpload, error) {
	var request pluginInstallApplyRequest
	if r.URL.RawQuery != "" || r.ContentLength > pluginInstallUploadLimit {
		return request, nil, errors.New("Invalid or oversized JAR upload request")
	}
	if err := http.NewResponseController(w).SetReadDeadline(time.Now().Add(5 * time.Minute)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return request, nil, errors.New("JAR upload is unavailable")
	}
	r.Body = http.MaxBytesReader(w, r.Body, pluginInstallUploadLimit)
	reader, err := r.MultipartReader()
	if err != nil {
		return request, nil, errors.New("Invalid multipart JAR upload")
	}
	fields := map[string]string{}
	uploads := map[string]curseForgeUpload{}
	var total int64
	for {
		part, err := reader.NextRawPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return request, nil, errors.New("Invalid or oversized JAR upload")
		}
		key := part.FormName()
		filename, hasFilename, err := uploadPartFilename(part)
		if err != nil || len(part.Header.Values("Content-Disposition")) != 1 || len(part.Header.Values("Content-Transfer-Encoding")) != 0 {
			return request, nil, errors.New("Invalid upload field")
		}
		switch key {
		case "source", "project_id", "version_id", "plan_fingerprint", "confirm_players":
			if _, seen := fields[key]; seen || hasFilename {
				return request, nil, errors.New("Duplicate or invalid install field")
			}
			value, err := io.ReadAll(io.LimitReader(part, 513))
			if err != nil || len(value) > 512 {
				return request, nil, errors.New("Invalid install field")
			}
			fields[key] = string(value)
		default:
			project, version, ok := strings.Cut(strings.TrimPrefix(key, "jar_"), "_")
			if !strings.HasPrefix(key, "jar_") || !ok || !positiveCurseForgeID(project) || !positiveCurseForgeID(version) || !safePluginFilename(filename) {
				return request, nil, errors.New("Invalid JAR field or filename")
			}
			if _, seen := uploads[key]; seen || len(uploads) >= pluginMaxArtifacts {
				return request, nil, errors.New("Duplicate or too many JAR uploads")
			}
			file, err := os.CreateTemp(directory, "upload-")
			if err != nil {
				return request, nil, errors.New("Private JAR staging unavailable")
			}
			n, copyErr := copyBoundedPluginUpload(file, part)
			closeErr := file.Close()
			total += n
			if copyErr != nil || closeErr != nil || total > pluginMaxDownload {
				return request, nil, errors.New("Empty, unreadable or oversized JAR upload")
			}
			uploads[key] = curseForgeUpload{filename: filename, path: file.Name(), size: n}
		}
	}
	// Count any bytes after the closing boundary against the request limit.
	if _, err := io.Copy(io.Discard, r.Body); err != nil {
		return request, nil, errors.New("Invalid or oversized JAR upload")
	}
	confirm := fields["confirm_players"]
	if len(fields) != 5 || fields["source"] != "CurseForge" || (confirm != "true" && confirm != "false") || len(uploads) == 0 {
		return request, nil, errors.New("Incomplete reviewed CurseForge upload")
	}
	return pluginInstallApplyRequest{
		Source: fields["source"], ProjectID: fields["project_id"], VersionID: fields["version_id"],
		PlanFingerprint: fields["plan_fingerprint"], ConfirmPlayers: confirm == "true",
	}, uploads, nil
}

func checkCurseForgeUploads(plan pluginInstallPlan, uploads map[string]curseForgeUpload) error {
	required := 0
	for _, a := range plan.Artifacts {
		if !a.UploadRequired || a.Satisfied != "" {
			continue
		}
		required++
		if _, ok := uploads[curseForgeUploadField(a)]; !ok {
			return fmt.Errorf("Choose the exact JAR for %s", a.Filename)
		}
	}
	if len(uploads) != required {
		return errors.New("Only JARs required by this review can be uploaded")
	}
	return nil
}

func verifyCurseForgeUpload(a pluginInstallArtifact, upload curseForgeUpload, directory string) (string, error) {
	if a.Source != "CurseForge" || !a.UploadRequired || !safePluginFilename(a.Filename) || upload.filename != a.Filename || a.Size <= 0 || a.Size > pluginMaxJar || !curseForgeSHA1.MatchString(a.SHA1) {
		return "", fmt.Errorf("Choose the exact JAR named %s", a.Filename)
	}
	if upload.size != a.Size {
		return "", fmt.Errorf("%s does not match the expected file size", a.Filename)
	}
	// The path comes only from CreateTemp in the Agent's private staging directory.
	file, err := os.Open(upload.path)
	if err != nil {
		return "", errors.New("Private JAR staging unavailable")
	}
	sha1Hash, sha512Hash := sha1.New(), sha512.New()
	n, readErr := io.Copy(io.MultiWriter(sha1Hash, sha512Hash), io.LimitReader(file, a.Size+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || n != a.Size || hex.EncodeToString(sha1Hash.Sum(nil)) != a.SHA1 {
		return "", fmt.Errorf("%s does not match the exact CurseForge file", a.Filename)
	}
	if err := os.Rename(upload.path, filepath.Join(directory, a.Filename)); err != nil {
		return "", errors.New("Verified JAR could not be staged")
	}
	return hex.EncodeToString(sha512Hash.Sum(nil)), nil
}
