# JustVoxel capability model

Development (`justvoxel-dev`), VM (`justvoxel-vm`) and HWS (`justvoxel-hws`) share one appliance implementation. Each builds from Home Server Base stable and repository source. Development showcases the full WebUI. HWS adds the physical-hardware package and configuration delta.

It includes:

- Home Server Base 10 / AlmaLinux 10 bootc foundation
- Podman and Quadlet support
- NetworkManager + `nmtui`
- OpenSSH
- firewalld
- SELinux container policy and administration tools
- JustVoxel Management Agent
- JustVoxel WebUI
- Management API v1 and Unix-socket privilege boundary
- `mjust`
- Minecraft runtime templates and helpers
- backup, restore, import/export, migration, storage, and system-management logic
- common appliance validation
- VM guest tooling

## One appliance implementation

JustVoxel does not maintain separate WebUI, Management Agent, Management API, or mjust implementations for VM and physical-hardware deployments.

The common appliance code owns the behavior. Hardware-dependent functionality can be present in that common code and requires real capabilities to operate. Development also shows the interface for unavailable hardware.

Examples include operations that need firmware interfaces, UPS/NUT support, hardware sensors, storage-health tooling, or other physical-device access.

The capability check belongs in the appliance management layer so unsupported controls can be hidden or refused cleanly without creating a second management stack.

## VM product

The VM image is directly suitable for VM validation and includes guest tooling required by supported hypervisors.

Hardware-dependent features must not make ordinary VM operation depend on physical-device packages or hardware being present.

## Physical-hardware capability

When the running system supplies the required hardware support, the same JustVoxel WebUI, Management Agent, mjust workflows, and validation model can expose the corresponding physical-machine functionality.

Deployment-specific hardware configuration remains local state. UPS identifiers, NUT driver selection, credentials, shutdown policy, device paths, and similar machine-specific values are not baked into the shared image.

The common code already contains capability-gated HWS behavior such as UPS
and firmware controls. The HWS product supplies hardware-only packages and
native units, including NUT; it does not fork the WebUI, Management API, or mjust
architecture.
