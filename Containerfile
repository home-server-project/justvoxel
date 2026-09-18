ARG HOME_SERVER_BASE_IMAGE=ghcr.io/home-server-project/home-server-base-10:stable
ARG JUSTVOXEL_VM_REPOSITORY=ghcr.io/home-server-project/justvoxel-vm
ARG JUSTVOXEL_BAREMETAL_REPOSITORY=ghcr.io/home-server-project/justvoxel-baremetal

FROM scratch AS ctx
COPY build_files /build_files
COPY build_artifacts /build_artifacts
COPY system_files /system_files
COPY docs /docs
COPY templates /templates
COPY runtime /runtime
COPY mjust /mjust
COPY cosign.pub /cosign.pub

FROM ${HOME_SERVER_BASE_IMAGE} AS justvoxel-common

LABEL containers.bootc=1 \
      ostree.bootable=1 \
      org.opencontainers.image.vendor="Home Server Project" \
      org.opencontainers.image.source="https://github.com/home-server-project/justvoxel" \
      io.home-server-project.justvoxel.base="home-server-base-10" \
      io.home-server-project.justvoxel.base-channel="stable" \
      io.home-server-project.justvoxel.base-profile="almalinux-10-minimal-plus" \
      io.home-server-project.justvoxel.status="development"

RUN --mount=type=bind,from=ctx,source=/,target=/ctx \
    --mount=type=tmpfs,dst=/run \
    --mount=type=tmpfs,dst=/tmp \
    /ctx/build_files/build-common.sh

RUN bootc container lint
STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]

FROM justvoxel-common AS justvoxel-vm
ARG JUSTVOXEL_VM_REPOSITORY

LABEL org.opencontainers.image.title="JustVoxel VM" \
      org.opencontainers.image.description="Immutable Home Server Base 10 Minecraft server appliance for virtual machines" \
      io.home-server-project.justvoxel.variant="vm"

RUN --mount=type=bind,from=ctx,source=/,target=/ctx \
    --mount=type=tmpfs,dst=/tmp \
    IMAGE_REPOSITORY="${JUSTVOXEL_VM_REPOSITORY}" \
    IMAGE_PRETTY_NAME="JustVoxel VM 10" \
    IMAGE_VARIANT="JustVoxel VM" \
    IMAGE_VARIANT_ID="justvoxel-vm" \
    /ctx/build_files/finalize-image.sh

RUN /usr/libexec/justvoxel/health/common \
    && /usr/libexec/justvoxel/health/vm \
    && bootc container lint --fatal-warnings

FROM justvoxel-common AS justvoxel-baremetal
ARG JUSTVOXEL_BAREMETAL_REPOSITORY

RUN --mount=type=bind,from=ctx,source=/,target=/ctx \
    --mount=type=tmpfs,dst=/run \
    --mount=type=tmpfs,dst=/tmp \
    /ctx/build_files/build-baremetal.sh

LABEL org.opencontainers.image.title="JustVoxel Bare Metal" \
      org.opencontainers.image.description="Immutable Home Server Base 10 Minecraft server appliance for physical hardware" \
      io.home-server-project.justvoxel.variant="baremetal"

RUN --mount=type=bind,from=ctx,source=/,target=/ctx \
    --mount=type=tmpfs,dst=/tmp \
    IMAGE_REPOSITORY="${JUSTVOXEL_BAREMETAL_REPOSITORY}" \
    IMAGE_PRETTY_NAME="JustVoxel Bare Metal 10" \
    IMAGE_VARIANT="JustVoxel Bare Metal" \
    IMAGE_VARIANT_ID="justvoxel-baremetal" \
    /ctx/build_files/finalize-image.sh

RUN /usr/libexec/justvoxel/health/common \
    && /usr/libexec/justvoxel/health/baremetal \
    && bootc container lint --fatal-warnings
