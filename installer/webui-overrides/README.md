# JustVoxel WebUI source overrides

The installer Containerfile builds Anaconda WebUI 68 from
`rhinstaller/anaconda-webui` commit
`a8745f07a18f303375351f3ba577d73108f5163d`, with Cockpit helpers from
`92e0108df1034207bcf7852c7747707d7da044ac`.
The upstream WebUI gitlink pins `rhinstaller/node-cache` to
`1a725c2d19e1bd6ed77702953fd7c97160a9a037`; its `.package-lock.json` is the
lockfile that upstream's `tools/node-modules make_package_lock_json` copies.
The builder extracts that lockfile, confirms the matching package manifest,
and uses `npm ci`. Build tools and sources stay in the Fedora 44 builder stage.
Only `dist/` replaces `/usr/share/cockpit/anaconda-webui/` in the final stage;
the distro RPM continues to provide Firefox and Anaconda/Cockpit integration.

`justvoxel.patch` modifies only wizard composition, the persistent header title,
the storage-editor launcher
and its prop plumbing, Installation method choices and automatic storage
application and scenario availability evaluation, the storage sidebar product
sentence, and the review Account row.
`src/components/justvoxel/JustVoxelPages.jsx` supplies the edition and login
information pages. `src/components/network/` backports the dedicated Network
wizard page introduced by Anaconda WebUI 70 while keeping the Fedora 44 WebUI 68
backend/frontend baseline. Upstream source headers remain intact when the patch
applies. No upstream repository, dependencies, or compiled assets are vendored
here.

Visible navigation is Welcome, Network, Date and time, JustVoxel edition,
Installation method, Login information, and Review and install. Installation
progress keeps upstream's final non-navigation behavior. `90-justvoxel.conf`
hides both storage configuration pages and the stock account/software pages.

The Network page embeds Cockpit NetworkManager so Ethernet, Wi-Fi, and supported
manual NetworkManager configuration stay upstream-owned. The installer image
explicitly includes `cockpit-networkmanager`, `NetworkManager-wifi`, and the same
Fedora wireless firmware families used by uCore: Atheros, Broadcom brcmfmac,
Intel legacy/DVM/MVM, MediaTek mt7xxx, NXP wireless, Realtek, and TI WiLink.
The page checks `https://ghcr.io/v2/` after initial load and after a Cockpit
network checkpoint completes. HTTP 200 or the registry's expected unauthenticated
HTTP 401 response counts as reachable. The wizard cannot advance until GHCR is
reachable; a manual Check again action is available after failures.

The edition page detects virtualization with `systemd-detect-virt --vm`. A
detected VM recommends and preselects JustVoxel VM; physical hardware, no VM
result, or a detection error falls back to JustVoxel HWS. Both choices remain
available for manual override, and the user's choice is kept for the installer
session. Before the edition becomes valid, the page requests an anonymous GHCR
pull token for the selected repository and checks the selected `:stable`
manifest. A private, missing, or unreachable image blocks the wizard. After that
check succeeds, the page updates Anaconda's active BOOTC source configuration
over its existing D-Bus API so both `sourceImgRef` and `targetImgRef` follow the
selected `:stable` channel: `justvoxel-vm:stable` or `justvoxel-hws:stable`. Login
information and the review Account value describe the voxel/voxel
bootstrap account. Account creation, locked root, and mandatory first-login
password change remain in `interactive-defaults.ks`.

Manual partitioning uses the same launch hook as the existing kebab-menu entry.
It preserves the upstream confirmation warning, Cockpit editor, and
`CheckStorageDialog` return path. A valid configured layout remains the upstream
`use-configured-storage` manual scenario. Technical manual scenarios are not
duplicated as radio choices. The only automatic choice is Use entire disk, using
upstream `erase-all` and its availability rules; dual-boot, reclaim, and reinstall
choices are not exposed. The availability hook evaluates only `erase-all`,
`mount-point-mapping`, and `use-configured-storage`; Fedora home-reuse/reinstall
and free-space availability hooks never execute. Their scenario metadata stays
available to upstream helpers. Use entire disk is selected by default unless a
valid manual layout is active. Selecting Manual partitioning again reopens the editor. Automatic
storage is applied and validated on Installation method before advancing, with
upstream errors and warning confirmation preserved because Storage configuration
is hidden.

Automatic partitioning uses GPT and plain ext4 root, growing from 8 GiB without
a maximum, with no separate home or disk swap. Anaconda/Blivet supplies the
platform boot partitions. The installer-only build helper in
`../anaconda-overrides/set-platform-defaults.py` replaces Anaconda's platform
specifications with a fixed 128 MiB EFI system partition and a fixed 896 MiB ext4
`/boot`. It fails the build if the expected upstream specifications change.
`default_partitioning` remains root-only, avoiding duplicate boot partitions.
Encryption is not enabled by default.
The installer environment uses edition-neutral JustVoxel presentation text while
Fedora platform identity and the installed payload remain unchanged. The selected
VM or HWS edition is shown by the edition page rather than hard-coded into the
installer identity.

`../branding.css` darkens only the header gradient behind the logo. The
installer-only Plymouth script theme in `../plymouth/` centers the provided
`logo.png` scaled to 720x240 on dark green, without loading dots or a spinner. The
Containerfile installs the script plugin and selects the theme before dracut;
the Plymouth module includes the theme in the initramfs.
Dracut explicitly includes `/.buildstamp` using `--include`. The upstream EDD
probe and its harmless warning remain unchanged.

This edit-only change has not been built, tested, or booted. Review and manual
Mode B validation must cover the wizard order, Ethernet DHCP, manual network
configuration, Wi-Fi discovery/connection on supported hardware, GHCR blocking
and retry behavior, VM detection with VM default, physical/unknown detection
with HWS default, manual edition override, selected BOOTC source/target refs,
product text, Firefox chrome,
header/logo contrast, Plymouth logo and boot warnings, bootstrap account,
automatic layout on blank disks and disks with an existing OS (including exact
EFI and `/boot` sizes), and the storage editor warning/validation/return behavior
before accepting the next ISO.
