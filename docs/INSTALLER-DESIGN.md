# JustVoxel installer design

The unified repository owns the VM and HWS products and their shared interactive online installer. For media preparation and first boot, see the [installation guide](INSTALLER.md).

## Build and image identity

The manual ISO workflow accepts only the dispatched main ref and checks out its event SHA. Cosign validates both current product tags using the repository public key before the installer build begins.

- `ghcr.io/home-server-project/justvoxel-vm:stable`
- `ghcr.io/home-server-project/justvoxel-hws:stable`

The workflow builds a local Fedora bootc installer environment, then runs the osbuild image-builder container against that local image to produce a `bootc-generic-iso`. It uploads `JustVoxel-x86_64.iso` and its SHA256 checksum together as a 90-day Actions artifact. See the [workflow](../.github/workflows/iso.yml) and [installer Containerfile](../installer/Containerfile).

The ISO fetches its payload online. Build-time verification checks the tags as they exist during the workflow. At installation time the selected tag can identify a newer image. The installed system continues to track the selected product’s stable channel.

## Network and edition selection

The installer builds its customized Anaconda WebUI from pinned upstream sources. Its network page exposes Cockpit NetworkManager and checks registry connectivity before continuing. The edition page uses `systemd-detect-virt --vm` to recommend VM in a detected virtual machine and HWS otherwise. Users can override either recommendation.

The edition page obtains an anonymous pull token and checks the selected repository’s stable manifest. Once the check succeeds, it sets Anaconda’s bootc source and target over D-Bus. The source uses the registry transport prefix and the installed tracking reference uses the ordinary image name. HWS supplies the initial Kickstart defaults. See the [Anaconda overrides and wizard details](../installer/webui-overrides/README.md).

## Storage and access

**Use entire disk** uses Anaconda’s automatic erase-and-install scenario. Manual partitioning opens the Cockpit storage editor and returns the layout to Anaconda for validation. On UEFI systems the default GPT layout includes a 128 MiB EFI system partition, an 896 MiB ext4 boot partition and a plain ext4 root that grows from 8 GiB to fill the remaining space. There is no separate automatic home or swap partition. Allow room for bootc updates, container images, logs and any Minecraft worlds stored on the system disk.

The Kickstart creates the wheel administrator `voxel`, locks root password login and expires the initial voxel password so that first authentication requires a change. Users configure SSH keys on the installed appliance. See the [Kickstart defaults](../installer/interactive-defaults.ks) and [SSH guide](SSH.md).

The installer deploys the operating system. The administrator sets up Minecraft workloads, data storage and backup targets after booting the appliance.
