package server

import (
	"context"
	"errors"
	"github.com/home-server-project/justvoxel-webui/internal/api"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

type setupSoftwareChoice struct{ ID, Name, Description string }

func simpleSoftware(family string) []setupSoftwareChoice {
	switch family {
	case "plugins":
		return []setupSoftwareChoice{{"paper", "Paper", uiText("ui.53f743143895")}, {"purpur", "Purpur", uiText("ui.53f743143895")}, {"spigot", "Spigot", uiText("ui.53f743143895")}, {"pufferfish", "Pufferfish", uiText("ui.53f743143895")}, {"leaf", "Leaf", uiText("ui.53f743143895")}}
	case "mods":
		return []setupSoftwareChoice{{"fabric", "Fabric", uiText("ui.f13c262f14d9")}, {"neoforge", "NeoForge", uiText("ui.f13c262f14d9")}, {"forge", "Forge", uiText("ui.55843cb14754")}, {"quilt", "Quilt", uiText("ui.55843cb14754")}}
	case "original":
		return []setupSoftwareChoice{{"vanilla", "Vanilla", uiText("ui.499c75715ac6")}, {"bedrock", "Bedrock", uiText("ui.3967fbfbcbd4")}}
	case "bedrock":
		return []setupSoftwareChoice{{"bedrock", "Bedrock", uiText("ui.e6a9810433d3")}}
	case "vanilla":
		return []setupSoftwareChoice{{"vanilla", "Vanilla", uiText("ui.ca7c1afad299")}}
	}
	return nil
}

func setupNeedsExactRuntime(software string) bool {
	return software != "paper" && software != "purpur" && software != "vanilla"
}

func setupRuntimeImage(software, version string) string {
	image := ""
	if regexp.MustCompile(`^1\.(20\.[56]|21(\.([0-9]|1[01]))?)$`).MatchString(version) {
		image = "java21"
	}
	if regexp.MustCompile(`^26\.[1-9][0-9]*(\.(0|[1-9][0-9]*))?$`).MatchString(version) {
		image = "java25"
	}
	if image != "" && software == "spigot" {
		image += "-jdk"
	}
	return image
}

// Keep the user's channel/version choice separate from the exact runtime sent
// to the authoritative planner. New software never gets a moving update policy.
func effectiveSetupMinecraft(mc setupMinecraftDraft) setupMinecraftDraft {
	if mc.ResolvedVersion != "" {
		mc.Version, mc.VersionPolicy = mc.ResolvedVersion, "pinned"
	}
	if mc.RuntimeImageTag != "" {
		mc.ImageTag = mc.RuntimeImageTag
	}
	return mc
}

func (a *App) resolveSetupRuntime(ctx context.Context, session string, draft *setupDraft, automaticCrossplay bool) error {
	if draft.Minecraft.ServerType == "bedrock" {
		client, ok := a.api.(interface {
			AdminSetupNativeBedrockPreview(context.Context, string, int) (api.AdminVersionStatus, error)
		})
		if !ok {
			return errors.New(uiText("ui.916ba8225584"))
		}
		players, err := strconv.Atoi(draft.Server.MaxPlayers)
		if err != nil || players < 1 || players > 1000 {
			return errors.New(uiText("ui.5dc62ee66d3b"))
		}
		status, err := client.AdminSetupNativeBedrockPreview(ctx, session, players)
		if err != nil || !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(\.[0-9]+)?$`).MatchString(status.SelectedCandidate) || status.ImageTag != "stable" || status.BedrockTCPPort < 1 || status.BedrockUDPStart < 1 || status.BedrockUDPEnd > 65535 || status.BedrockUDPEnd-status.BedrockUDPStart+1 < players {
			return errors.New(uiText("ui.67fa3d28dccf"))
		}
		if draft.Minecraft.ResolvedVersion != "" && (draft.Minecraft.ResolvedVersion != status.SelectedCandidate || draft.Minecraft.BedrockTCPPort != status.BedrockTCPPort || draft.Minecraft.BedrockUDPStart != status.BedrockUDPStart || draft.Minecraft.BedrockUDPEnd != status.BedrockUDPEnd) {
			return errors.New(uiText("ui.b2fb4f8d2de6"))
		}
		draft.Server.BedrockEnabled = false
		draft.Minecraft.JavaPort, draft.Minecraft.JavaMemory, draft.Minecraft.BedrockPort = "0", "", "0"
		draft.Minecraft.VersionPolicy, draft.Minecraft.Version, draft.Minecraft.ResolvedVersion = "pinned", status.SelectedCandidate, status.SelectedCandidate
		draft.Minecraft.ImageTag, draft.Minecraft.RuntimeImageTag, draft.Minecraft.LoaderVersion = "stable", "stable", ""
		draft.Minecraft.BedrockTCPPort, draft.Minecraft.BedrockUDPStart, draft.Minecraft.BedrockUDPEnd = status.BedrockTCPPort, status.BedrockUDPStart, status.BedrockUDPEnd
		return nil
	}
	if automaticCrossplay {
		draft.Server.BedrockEnabled = false
	}
	draft.Minecraft.ResolvedVersion, draft.Minecraft.RuntimeImageTag, draft.Minecraft.LoaderVersion = "", "", ""
	if err := validateSetupVersion(draft.Minecraft); err != nil {
		return err
	}
	if err := validateSetupConnectionDraft(*draft); err != nil {
		return err
	}
	client, ok := a.api.(setupVersionPreviewAPI)
	if !ok {
		return errors.New(uiText("ui.6b1be7716cd4"))
	}
	bedrock := draft.Server.BedrockEnabled
	if automaticCrossplay {
		bedrock = false
	}
	status, err := client.AdminSetupVersionPreview(ctx, session, draft.Minecraft.VersionPolicy, draft.Minecraft.Version, bedrock, draft.Minecraft.ServerType)
	if err != nil || status.SelectedCandidate == "" {
		return errors.New(uiText("ui.149789b66d59"))
	}
	if draft.Minecraft.VersionPolicy == "pinned" && status.SelectedCandidate != draft.Minecraft.Version {
		return errors.New(uiText("ui.5feedd259f95"))
	}
	if automaticCrossplay && recommendedServerSupportsBedrock(draft.Minecraft.ServerType) {
		// Ask about this exact release; cross-play must never choose a different
		// game/loader pair or prevent the verified Java server from being created.
		crossplay, crossErr := client.AdminSetupVersionPreview(ctx, session, "pinned", status.SelectedCandidate, true, draft.Minecraft.ServerType)
		bedrock = crossErr == nil && crossplay.SelectedCandidate == status.SelectedCandidate && crossplay.LoaderVersion == status.LoaderVersion && crossplay.CrossplayCompatible && (draft.Minecraft.ServerType != "fabric" && draft.Minecraft.ServerType != "neoforge" || crossplay.ManagedCrossplayAvailable)
	}
	if bedrock && !automaticCrossplay && (!status.CrossplayCompatible || ((draft.Minecraft.ServerType == "fabric" || draft.Minecraft.ServerType == "neoforge") && !status.ManagedCrossplayAvailable)) {
		return errors.New(uiText("ui.b3f2d3d83df9"))
	}
	if setupNeedsExactRuntime(draft.Minecraft.ServerType) {
		required := setupRuntimeImage(draft.Minecraft.ServerType, status.SelectedCandidate)
		if required == "" || status.ImageTag != required {
			return errors.New(uiText("ui.4f91c4c63533"))
		}
		if draft.Minecraft.ServerType != "spigot" && draft.Minecraft.ServerType != "pufferfish" && draft.Minecraft.ServerType != "leaf" && !regexp.MustCompile(`^[0-9]+(\.[0-9]+){1,3}$`).MatchString(status.LoaderVersion) {
			return errors.New(uiText("ui.e100c7053df0"))
		}
		if (draft.Minecraft.ServerType == "pufferfish" || draft.Minecraft.ServerType == "leaf") && status.LoaderVersion != "" {
			return errors.New(uiText("ui.bed81074f24f"))
		}
		draft.Minecraft.RuntimeImageTag, draft.Minecraft.LoaderVersion = required, status.LoaderVersion
	}
	draft.Server.BedrockEnabled = bedrock
	if automaticCrossplay || setupNeedsExactRuntime(draft.Minecraft.ServerType) {
		draft.Minecraft.ResolvedVersion = status.SelectedCandidate
	}
	return nil
}

func (a *App) setupWizardSimple(w http.ResponseWriter, r *http.Request) {
	session, client, identity, ok := a.setupWizardRequest(w, r, true)
	if !ok {
		return
	}
	if a.redirectCurrentSetupOperation(w, r, session, client) {
		return
	}
	draft, exists := setupDrafts.get(a, session)
	if !exists || draft.Mode != "recommended" {
		draft = setupDraft{AddServer: isAddServerRequest(r), Mode: "recommended", SimpleState: 1}
	}
	fail := func(message string) {
		setupDrafts.save(a, session, draft)
		w.WriteHeader(http.StatusBadRequest)
		a.renderSetupWizard(w, identity, draft, csrfFromRequest(r), message)
	}
	if r.FormValue("direction") == "back" {
		if draft.SimpleState > 1 {
			draft.SimpleState--
		}
		if draft.SimpleState == 1 {
			draft.Minecraft.ServerType = ""
		}
	} else {
		switch draft.SimpleState {
		case 1:
			family := r.FormValue("family")
			if len(simpleSoftware(family)) == 0 {
				fail(ui(w, "ui.57379d1f0bb3"))
				return
			}
			draft.SimpleFamily, draft.SimpleState = family, 2
			draft.Minecraft.ServerType = ""
		case 2:
			selected := r.FormValue("server_type")
			valid := false
			for _, choice := range simpleSoftware(draft.SimpleFamily) {
				if choice.ID == selected {
					valid = true
				}
			}
			if !valid {
				fail(ui(w, "ui.d3f83a613cc5"))
				return
			}
			draft.Minecraft.ServerType, draft.SimpleState = selected, 3
		case 3:
			name, gameMode := strings.TrimSpace(r.FormValue("display_name")), r.FormValue("game_mode")
			draft.DisplayName, draft.Minecraft.GameMode = name, gameMode
			if err := a.validateAddServerName(r.Context(), session, name); err != nil {
				fail(err.Error())
				return
			}
			if !validSetupGameMode(gameMode) || draft.Minecraft.ServerType == "bedrock" && gameMode == "spectator" {
				fail(ui(w, "ui.9b4e69acae19"))
				return
			}
			resolved, err := a.newSetupDraft(r.Context(), session, client, draft.AddServer)
			if err != nil {
				fail(ui(w, "ui.1d5d096ce8c5"))
				return
			}
			resolved.Mode, resolved.SimpleState, resolved.SimpleFamily = "recommended", 3, draft.SimpleFamily
			resolved.DisplayName, resolved.Minecraft.ServerType, resolved.Minecraft.GameMode = name, draft.Minecraft.ServerType, gameMode
			resolved.Minecraft.VersionPolicy = "recommended"
			if err = a.resolveSetupRuntime(r.Context(), session, &resolved, true); err != nil {
				fail(err.Error())
				return
			}
			if err = a.resolveRecommendedSetupStorage(r.Context(), session, &resolved); err != nil {
				fail(err.Error())
				return
			}
			if err = validateSetupMinecraft(resolved.Minecraft, resolved.Defaults); err != nil {
				fail(err.Error())
				return
			}
			resolved.Server.Complete, resolved.Storage.Complete, resolved.Backups.Complete = true, true, true
			resolved.Minecraft.ConnectionsComplete, resolved.Minecraft.VersionComplete, resolved.Minecraft.ResourcesComplete, resolved.Minecraft.Complete = true, true, true, true
			resolved.CurrentStep = setupReviewStep
			draft = resolved
			a.recordSetupDraftDiagnosticBestEffort(r.Context(), client, session, "simple WebUI setup defaults accepted", draft)
		default:
			fail(ui(w, "ui.e74fead2afaa"))
			return
		}
	}
	setupReviews.delete(a, session)
	setupDrafts.save(a, session, draft)
	http.Redirect(w, r, setupBasePath(r), http.StatusSeeOther)
}

type setupSoftwareGroup struct {
	Name, Description string
	Software          []setupSoftwareChoice
}

func setupSoftwareGroups() []setupSoftwareGroup {
	return []setupSoftwareGroup{
		{Name: uiText("ui.53f743143895"), Software: simpleSoftware("plugins")},
		{Name: uiText("ui.6a5397e7245c"), Software: simpleSoftware("mods")},
		{Name: uiText("ui.10ef48de6660"), Software: simpleSoftware("original")},
	}
}
