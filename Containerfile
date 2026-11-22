ARG PRODUCT_STATUS=development
ARG PARENT_CHANNEL=testing
ARG JUSTVOXEL_DEV_IMAGE=ghcr.io/home-server-project/justvoxel-dev:testing
ARG JUSTVOXEL_DEV_REPOSITORY=ghcr.io/home-server-project/justvoxel-dev
ARG JUSTVOXEL_HWS_REPOSITORY=ghcr.io/home-server-project/justvoxel-hws

FROM scratch AS ctx
COPY build_files /build_files
COPY cosign.pub /cosign.pub

FROM ${JUSTVOXEL_DEV_IMAGE} AS justvoxel-vm
ARG PARENT_CHANNEL
ARG PRODUCT_STATUS

LABEL containers.bootc=1 \
      ostree.bootable=1 \
      org.opencontainers.image.vendor="Home Server Project" \
      org.opencontainers.image.source="https://github.com/home-server-project/justvoxel" \
      org.opencontainers.image.title="JustVoxel VM" \
      org.opencontainers.image.description="JustVoxel Minecraft server appliance for virtual machines" \
      io.home-server-project.justvoxel.role="product" \
      io.home-server-project.justvoxel.variant="vm" \
      io.home-server-project.justvoxel.product-base="justvoxel-dev" \
      io.home-server-project.justvoxel.product-base-channel="${PARENT_CHANNEL}" \
      io.home-server-project.justvoxel.status="${PRODUCT_STATUS}"

RUN test "$(cat /usr/lib/justvoxel/variant)" = "justvoxel-vm"

STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]

FROM ${JUSTVOXEL_DEV_IMAGE} AS justvoxel-hws
ARG PARENT_CHANNEL
ARG PRODUCT_STATUS
ARG JUSTVOXEL_DEV_REPOSITORY
ARG JUSTVOXEL_HWS_REPOSITORY

LABEL containers.bootc=1 \
      ostree.bootable=1 \
      org.opencontainers.image.vendor="Home Server Project" \
      org.opencontainers.image.source="https://github.com/home-server-project/justvoxel" \
      org.opencontainers.image.title="JustVoxel HWS" \
      org.opencontainers.image.description="JustVoxel Minecraft server appliance with physical-hardware support" \
      io.home-server-project.justvoxel.role="product" \
      io.home-server-project.justvoxel.variant="hws" \
      io.home-server-project.justvoxel.product-base="justvoxel-dev" \
      io.home-server-project.justvoxel.product-base-channel="${PARENT_CHANNEL}" \
      io.home-server-project.justvoxel.status="${PRODUCT_STATUS}"

RUN --mount=type=bind,from=ctx,source=/,target=/ctx \
    --mount=type=tmpfs,dst=/run \
    --mount=type=tmpfs,dst=/tmp \
    /ctx/build_files/build-hws.sh

RUN --mount=type=bind,from=ctx,source=/,target=/ctx \
    --mount=type=tmpfs,dst=/tmp \
    JUSTVOXEL_DEV_REPOSITORY="${JUSTVOXEL_DEV_REPOSITORY}" \
    JUSTVOXEL_HWS_REPOSITORY="${JUSTVOXEL_HWS_REPOSITORY}" \
    /ctx/build_files/finalize-hws.sh

RUN /usr/libexec/justvoxel/health/hws \
    && bootc container lint --fatal-warnings

STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]
