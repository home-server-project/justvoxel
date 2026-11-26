ARG HOME_SERVER_BASE_IMAGE=ghcr.io/home-server-project/home-server-base-10:stable
ARG JUSTVOXEL_HWS_REPOSITORY=ghcr.io/home-server-project/justvoxel-hws
ARG JUSTVOXEL_VM_REPOSITORY=ghcr.io/home-server-project/justvoxel-vm
ARG SUPERFILE_PACKAGE_IMAGE=ghcr.io/home-server-project/superfile:stable
ARG PLAYIT_PACKAGE_IMAGE=ghcr.io/home-server-project/playit:stable
ARG GLANCES_PACKAGE_IMAGE=ghcr.io/home-server-project/glances:stable
ARG MINECRAFT_NETWORK_PACKAGE_IMAGE=ghcr.io/home-server-project/minecraft-network-plugins:stable

FROM ${SUPERFILE_PACKAGE_IMAGE} AS superfile-package
FROM ${GLANCES_PACKAGE_IMAGE} AS glances-package
FROM ${PLAYIT_PACKAGE_IMAGE} AS playit-package
FROM ${MINECRAFT_NETWORK_PACKAGE_IMAGE} AS minecraft-network-package

FROM scratch AS ctx
COPY build_files /build_files
COPY build_artifacts/management/management-agent /build_artifacts/management/management-agent
COPY build_artifacts/webui /build_artifacts/webui
COPY system_files /system_files
COPY docs /docs
COPY templates /templates
COPY runtime /runtime
COPY mjust /mjust
COPY --from=superfile-package /rpms /superfile-rpms
COPY --from=glances-package /rpms /glances-rpms
COPY --from=playit-package /rpms /playit-rpms
COPY --from=minecraft-network-package /artifacts/limboapi.jar /minecraft-network-artifacts/limboapi.jar
COPY --from=minecraft-network-package /artifacts/justvoxel-lobby.jar /minecraft-network-artifacts/justvoxel-lobby.jar
COPY cosign.pub /cosign.pub

FROM ${HOME_SERVER_BASE_IMAGE} AS common
LABEL io.home-server-project.justvoxel.base="home-server-base-10" \
      io.home-server-project.justvoxel.base-channel="stable" \
      io.home-server-project.justvoxel.base-profile="almalinux-10-minimal-plus"

RUN --mount=type=bind,from=ctx,source=/,target=/ctx \
    --mount=type=tmpfs,dst=/run \
    --mount=type=tmpfs,dst=/tmp \
    /ctx/build_files/build-common.sh false

FROM common AS justvoxel-vm
ARG JUSTVOXEL_VM_REPOSITORY
ARG PRODUCT_STATUS=development

LABEL containers.bootc=1 \
      ostree.bootable=1 \
      org.opencontainers.image.vendor="Home Server Project" \
      org.opencontainers.image.source="https://github.com/home-server-project/justvoxel" \
      org.opencontainers.image.title="JustVoxel VM" \
      org.opencontainers.image.description="JustVoxel Minecraft server appliance for virtual machines" \
      io.home-server-project.justvoxel.role="product" \
      io.home-server-project.justvoxel.variant="vm" \
      io.home-server-project.justvoxel.status="${PRODUCT_STATUS}"

RUN --mount=type=bind,from=ctx,source=/,target=/ctx \
    --mount=type=tmpfs,dst=/tmp \
    IMAGE_REPOSITORY="${JUSTVOXEL_VM_REPOSITORY}" \
    IMAGE_PRETTY_NAME="JustVoxel VM 10" \
    IMAGE_VARIANT="JustVoxel VM" \
    IMAGE_VARIANT_ID="justvoxel-vm" \
    /ctx/build_files/finalize-image.sh

RUN test "$(cat /usr/lib/justvoxel/variant)" = "justvoxel-vm" \
    && /usr/libexec/justvoxel/health/common \
    && bootc container lint --fatal-warnings

STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]

FROM common AS justvoxel-hws
ARG PRODUCT_STATUS=development
ARG JUSTVOXEL_HWS_REPOSITORY

LABEL containers.bootc=1 \
      ostree.bootable=1 \
      org.opencontainers.image.vendor="Home Server Project" \
      org.opencontainers.image.source="https://github.com/home-server-project/justvoxel" \
      org.opencontainers.image.title="JustVoxel HWS" \
      org.opencontainers.image.description="JustVoxel Minecraft server appliance with physical-hardware support" \
      io.home-server-project.justvoxel.role="product" \
      io.home-server-project.justvoxel.variant="hws" \
      io.home-server-project.justvoxel.status="${PRODUCT_STATUS}"

RUN --mount=type=bind,from=ctx,source=/,target=/ctx \
    --mount=type=tmpfs,dst=/run \
    --mount=type=tmpfs,dst=/tmp \
    /ctx/build_files/build-hws.sh

RUN --mount=type=bind,from=ctx,source=/,target=/ctx \
    --mount=type=tmpfs,dst=/tmp \
    JUSTVOXEL_HWS_REPOSITORY="${JUSTVOXEL_HWS_REPOSITORY}" \
    /ctx/build_files/finalize-hws.sh

RUN /usr/libexec/justvoxel/health/common \
    && /usr/libexec/justvoxel/health/hws \
    && bootc container lint --fatal-warnings

STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]
