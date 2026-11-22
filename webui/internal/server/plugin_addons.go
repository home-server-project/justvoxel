package server

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/home-server-project/justvoxel-webui/internal/api"
)

type addonsWorkspaceAPI interface {
	AdminAddonInventory(context.Context, string) (api.AddonInventory, error)
	AdminAddonUpload(context.Context, string, string, io.Reader, bool) (api.AddonPlan, api.PersistentOperationResponse, error)
	AdminAddonRecover(context.Context, string, api.PluginInstallAcknowledgeRequest) (api.PersistentOperationResponse, error)
}

func (a *App) pluginsAddonInventory(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	client, ok := a.api.(addonsWorkspaceAPI)
	if !ok {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.da01c058879a"))
		return
	}
	out, err := client.AdminAddonInventory(r.Context(), session)
	if err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.c0cdc5506392"))
		return
	}
	writeMinecraftWorkspaceJSON(w, 200, out)
}
func (a *App) pluginsAddonPlan(w http.ResponseWriter, r *http.Request) {
	a.pluginsAddonUpload(w, r, false)
}
func (a *App) pluginsAddonInstall(w http.ResponseWriter, r *http.Request) {
	a.pluginsAddonUpload(w, r, true)
}
func (a *App) pluginsAddonUpload(w http.ResponseWriter, r *http.Request, apply bool) {
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	// Header CSRF allows verification before streaming a large body. Use the
	// existing cookie/origin policy without ParseMultipartForm memory/disk staging.
	r.Form = url.Values{"csrf": {r.Header.Get("X-CSRF-Token")}}
	if !a.validCSRF(r) {
		writeMinecraftWorkspaceError(w, 403, ui(w, "ui.74bccc44f472"))
		return
	}
	if r.URL.RawQuery != "" {
		writeMinecraftWorkspaceError(w, 400, ui(w, "ui.324eadbcceec"))
		return
	}
	client, ok := a.api.(addonsWorkspaceAPI)
	if !ok {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.da01c058879a"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, api.AddonUploadLimit)
	reader, err := r.MultipartReader()
	if err != nil {
		writeMinecraftWorkspaceError(w, 400, ui(w, "ui.d26664f51507"))
		return
	}
	// Require type first so the WebUI validates the fixed contract before sending
	// artifact bytes. Remaining fields are checked while the Agent consumes them.
	first, err := reader.NextPart()
	if err != nil || first.FormName() != "type" || first.FileName() != "" {
		writeMinecraftWorkspaceError(w, 400, ui(w, "ui.4c03b99cfdac"))
		return
	}
	raw, err := io.ReadAll(io.LimitReader(first, 65))
	first.Close()
	kind := string(raw)
	if err != nil || !api.ValidAddonKind(kind) {
		writeMinecraftWorkspaceError(w, 400, ui(w, "ui.3981e09a4203"))
		return
	}
	pipeReader, pipeWriter := io.Pipe()
	defer pipeReader.Close()
	writer := multipart.NewWriter(pipeWriter)
	result := make(chan error, 1)
	go func() {
		err := proxyAddonMultipart(reader, writer, kind, apply)
		if err == nil {
			err = writer.Close()
		}
		pipeWriter.CloseWithError(err)
		result <- err
	}()
	plan, operation, err := client.AdminAddonUpload(r.Context(), session, writer.FormDataContentType(), pipeReader, apply)
	pipeReader.Close()
	r.Body.Close()
	// The proxy finishes on body/pipe closure even when the Agent rejects early.
	proxyErr := <-result
	if err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.ee0f3ce8b9fd"))
		return
	}
	if proxyErr != nil {
		writeMinecraftWorkspaceError(w, 400, proxyErr.Error())
		return
	}
	if apply {
		writeMinecraftWorkspaceJSON(w, 202, operation)
	} else {
		writeMinecraftWorkspaceJSON(w, 200, plan)
	}
}
func proxyAddonMultipart(reader *multipart.Reader, writer *multipart.Writer, kind string, apply bool) error {
	if err := writer.WriteField("type", kind); err != nil {
		return err
	}
	seen := map[string]bool{"type": true}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		key := part.FormName()
		if seen[key] || !(key == "file" || apply && (key == "plan_fingerprint" || key == "confirm_players")) {
			return errors.New(uiText("ui.8499fc961d0e"))
		}
		seen[key] = true
		if key == "file" {
			_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
			if err != nil || params["filename"] == "" {
				return errors.New(uiText("ui.6e659e8e1967"))
			}
			target, err := writer.CreateFormFile("file", params["filename"])
			if err != nil {
				return err
			}
			n, err := io.Copy(target, io.LimitReader(part, (128<<20)+1))
			if err != nil {
				return err
			}
			if n <= 0 || n > 128<<20 {
				return errors.New(uiText("ui.60ded17eb49a"))
			}
		} else {
			if part.FileName() != "" {
				return errors.New(uiText("ui.d477b0700e82"))
			}
			raw, err := io.ReadAll(io.LimitReader(part, 513))
			if err != nil || len(raw) > 512 {
				return errors.New(uiText("ui.8a48dea45d5c"))
			}
			if err = writer.WriteField(key, string(raw)); err != nil {
				return err
			}
		}
		part.Close()
	}
	if !seen["file"] || apply && (!seen["plan_fingerprint"] || !seen["confirm_players"]) {
		return errors.New(uiText("ui.a8af58478c0a"))
	}
	return nil
}
func (a *App) pluginsAddonRecover(w http.ResponseWriter, r *http.Request) {
	session, _, ok := a.minecraftWorkspaceIdentity(w, r, "administrator")
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if r.ParseForm() != nil || !a.validCSRF(r) {
		writeMinecraftWorkspaceError(w, 403, ui(w, "ui.b8522acc4a3c"))
		return
	}
	for key, values := range r.PostForm {
		if (key != "csrf" && key != "operation_id") || len(values) != 1 {
			writeMinecraftWorkspaceError(w, 400, ui(w, "ui.5e1c649e1015"))
			return
		}
	}
	if r.URL.RawQuery != "" || !validWorkspaceLogID("plugins", r.FormValue("operation_id")) {
		writeMinecraftWorkspaceError(w, 400, ui(w, "ui.10ce2f02266a"))
		return
	}
	client, ok := a.api.(addonsWorkspaceAPI)
	if !ok {
		writeMinecraftWorkspaceError(w, 503, ui(w, "ui.4e618a5a9947"))
		return
	}
	out, err := client.AdminAddonRecover(r.Context(), session, api.PluginInstallAcknowledgeRequest{OperationID: r.FormValue("operation_id")})
	if err != nil {
		a.writeMinecraftWorkspaceAPIError(w, err, ui(w, "ui.2288864fdd4a"))
		return
	}
	writeMinecraftWorkspaceJSON(w, 200, out)
}
