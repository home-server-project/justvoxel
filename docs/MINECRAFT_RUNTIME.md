# Minecraft runtime foundation

Step 2 ships an inactive runtime template based on the hardware-verified Home Server Project Paper deployment. It does not create a world, accept the Minecraft EULA, or activate a Minecraft service by itself.

The runtime is based on `docker.io/itzg/minecraft-server:latest` with Paper, Geyser, Floodgate, and ViaVersion. Java uses TCP 25565 by default and Bedrock uses UDP 19132 by default, but mjust will render administrator-selected host ports later.

The template preserves the proven graceful-stop behavior: the itzg shutdown announcement delay is 60 seconds, Podman has 180 seconds to stop, and systemd allows 240 seconds around the stop operation. RCON is enabled inside the container for local administrative commands but is not published as a host port.

Persistent Minecraft data is a configurable host path mounted only at `/data` with a private SELinux `:Z` label. No Mojang server JAR, Paper server JAR, world, EULA acceptance, or Floodgate private key is baked into the bootc image.

The immutable image stores only templates under `/usr/share/justvoxel/templates`. A later mjust implementation will create the mutable `/etc/justvoxel` configuration and `/etc/containers/systemd/minecraft.container` after explicit setup.
