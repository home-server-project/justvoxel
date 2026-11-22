# SSH access after installation

Complete the first login as `voxel` and change the bootstrap password. For remote terminal access, find the appliance IP address in your router, hypervisor, or local console.

## Public key and private key

An SSH key pair has two parts:

- a **private key** that stays on your own computer
- a **public key** that may be copied to servers and installation media

Authorize the public key on the installed appliance.

Never upload, paste, or commit your private SSH key to GitHub, an installer repository, an ISO, or another public location.

Common OpenSSH key files look like this:

```text
~/.ssh/id_ed25519      private key - keep this secret
~/.ssh/id_ed25519.pub  public key - safe to copy to a server
```

The `.pub` file is the one JustVoxel needs.

## Check whether you already have a key

### Linux or macOS

Open a terminal and run:

```bash
ls ~/.ssh/*.pub
```

If you see a file such as `id_ed25519.pub`, you may already have a usable public key.

To display it:

```bash
cat ~/.ssh/id_ed25519.pub
```

A normal Ed25519 public key starts with `ssh-ed25519`.

### Windows PowerShell

Open PowerShell and run:

```powershell
Get-ChildItem $HOME\.ssh\*.pub
```

If `id_ed25519.pub` exists, display it with:

```powershell
Get-Content $HOME\.ssh\id_ed25519.pub
```

## Create a new Ed25519 key

If you do not already have a key you want to use, create one with OpenSSH.

### Linux or macOS

```bash
ssh-keygen -t ed25519
```

### Windows PowerShell

```powershell
ssh-keygen -t ed25519
```

Accept the normal file location unless you have a reason to use a custom path.

OpenSSH may ask for a passphrase. A passphrase protects the private key if somebody gets access to the key file. Whether to use one is your decision, but the private key itself must always remain private.

After generation, copy the contents of the `.pub` file, not the private-key file.

## Authorize your key after installation

From a client with OpenSSH’s `ssh-copy-id`, run the following, replacing `SERVER_IP` with the appliance address. Authenticate with your updated password.

```bash
ssh-copy-id -i ~/.ssh/id_ed25519.pub voxel@SERVER_IP
```

## Connect after installation

Find the JustVoxel machine's IP address from your router, hypervisor, or local console, then connect from a computer that has the matching private key.

Normally:

```bash
ssh voxel@SERVER_IP
```

If your private key uses a custom filename and is not loaded into an SSH agent:

```bash
ssh -i /path/to/private-key voxel@SERVER_IP
```

On Windows PowerShell the normal command is the same:

```powershell
ssh voxel@SERVER_IP
```

## If SSH warns after reinstalling JustVoxel

A fresh installation creates a new SSH host identity.

If you reinstall JustVoxel on a machine that keeps the same IP address, your client may warn that the remote host identification has changed. If you intentionally reinstalled that machine and have verified that you are connecting to the correct IP, remove the old saved host key and connect again.

Linux, macOS, and Windows OpenSSH clients can use:

```bash
ssh-keygen -R SERVER_IP
```

Then reconnect and review the new host-key prompt before accepting it.
