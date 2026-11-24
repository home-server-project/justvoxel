# Install JustVoxel

One x86_64 online installer supports **JustVoxel VM** for virtual machines and **JustVoxel HWS** for physical servers. The installer needs network access to download the selected stable edition from GitHub Container Registry (GHCR).

## Get the ISO

[⬇️ Download JustVoxel x86_64 ISO](https://sourceforge.net/projects/justvoxel/files/JustVoxel-x86_64.iso/download)

### Create a bootable USB

For installation on physical hardware, write the ISO to a USB drive. Common options include:

- [balenaEtcher](https://etcher.balena.io/) — graphical image writer for multiple desktop platforms.
- [Rufus](https://rufus.ie/en/) — a common option on Windows.
- [Ventoy](https://www.ventoy.net/en/index.html) — useful when you want to boot several ISO images from one USB drive.
- [Fedora Media Writer](https://fedoraproject.org/workstation/download/) — graphical image writer for Linux, Windows, and macOS.

JustVoxel uses a bootable ISO that supports direct writing. On Linux and other Unix-like systems, tools such as `dd` can write the ISO directly to the USB device.

**Writing an image directly to a USB drive erases everything on it.** Double-check that you selected the intended USB drive before writing. Selecting the wrong device can erase another disk.

Boot the physical machine from the prepared USB drive in UEFI mode.

## Prepare the machine

Plan for about four CPU threads or vCPUs and at least 8 GiB RAM. For normal Minecraft use, 12–16 GiB RAM gives more headroom. World size, player count and plugins affect the resources needed.

- **VM** — Create an x86_64 UEFI VM with one blank virtual disk of at least 40 GiB. Attach the ISO as its boot media. You can add a separate backup disk after installation.
- **HWS** — Back up important files and disconnect non-target disks where practical. A 64 GB or larger system drive is recommended, or a 128 GB or larger SSD/NVMe if worlds will share the system disk. Preparing the USB drive erases its contents. Boot the machine from the USB in UEFI mode.

## Run the installer

1. On **Network**, connect to Ethernet or Wi-Fi. The installer checks access to GHCR. If the check fails, correct the connection and select **Check again**.
2. Set the date and time, then confirm the recommended edition or select the other one. The wizard checks that your selected image is available.
3. On **Installation method**, check the target disk. **Use entire disk erases the selected disk**, including its partitions and files. Choose **Manual partitioning** if you need to supply your own layout.
4. Read the login information, review the edition and storage choices, and start installation. When installation completes, remove the USB or detach the ISO and boot from the installed disk.

## First boot

Log in at the console as `voxel` with the initial password `voxel`. Change it at the first login prompt. Use the appliance WebUI or run `mjust` to start setup. Minecraft server creation and EULA acceptance happen in the appliance after installation.

Reconnect additional storage after verifying the system boots. JustVoxel supports separate local storage and NFS/SMB targets for supported data and backup workflows. See the [storage guide](STORAGE.md), [SSH access guide](SSH.md), and [installer design](INSTALLER-DESIGN.md).
