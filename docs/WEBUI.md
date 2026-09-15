# Web management

JustVoxel WebUI is designed for simple administration from a trusted local home network.

By default, Web management uses plain HTTP on TCP port `8099`. This is intentional: the appliance can show a local address such as `http://192.168.1.50:8099`, and users can open it directly without installing a private certificate or bypassing a self-signed certificate warning.

Because default local WebUI traffic is not protected by TLS, administrator credentials and sessions should only be used on a network you trust. The absence of TLS in the default local setup is an intentional appliance design choice, not a missing configuration step.

Do not forward TCP port `8099` directly to the public Internet.

If remote or public access is required, place JustVoxel behind a properly secured solution that provides trusted HTTPS or private-network access, such as an appropriately configured reverse proxy, VPN, or private-network service. Those advanced access methods are separate from the default JustVoxel local WebUI and are not configured automatically.

## Local behavior

When Web management is enabled and healthy, the local console and SSH welcome message show the current LAN address using the `http://` scheme and port `8099`.

When Web management is disabled, the WebUI service is stopped and its firewalld service is removed, so the management port is no longer exposed.

The browser-facing WebUI remains unprivileged. Authentication, session validation, password hashing, login throttling, password changes, and privileged operations remain behind the local JustVoxel Management API over its Unix socket.
