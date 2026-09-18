ARG JUSTVOXEL_BASE_IMAGE=ghcr.io/home-server-project/justvoxel-base:testing
ARG JUSTVOXEL_BASE_REPOSITORY=ghcr.io/home-server-project/justvoxel-base
ARG JUSTVOXEL_HWE_REPOSITORY=ghcr.io/home-server-project/justvoxel-hwe

FROM scratch AS ctx
COPY build_files /build_files
COPY cosign.pub /cosign.pub

FROM ${JUSTVOXEL_BASE_IMAGE} AS justvoxel-hwe
ARG JUSTVOXEL_BASE_REPOSITORY
ARG JUSTVOXEL_HWE_REPOSITORY

LABEL containers.bootc=1 \
      ostree.bootable=1 \
      org.opencontainers.image.vendor="Home Server Project" \
      org.opencontainers.image.source="https://github.com/home-server-project/justvoxel" \
      org.opencontainers.image.title="JustVoxel HWE" \
      org.opencontainers.image.description="JustVoxel Minecraft server appliance with physical-hardware support" \
      io.home-server-project.justvoxel.role="product" \
      io.home-server-project.justvoxel.variant="hwe" \
      io.home-server-project.justvoxel.product-base="justvoxel-base" \
      io.home-server-project.justvoxel.product-base-channel="testing" \
      io.home-server-project.justvoxel.status="development"

RUN --mount=type=bind,from=ctx,source=/,target=/ctx \
    --mount=type=tmpfs,dst=/run \
    --mount=type=tmpfs,dst=/tmp \
    /ctx/build_files/build-hwe.sh

RUN --mount=type=bind,from=ctx,source=/,target=/ctx \
    --mount=type=tmpfs,dst=/tmp \
    JUSTVOXEL_BASE_REPOSITORY="${JUSTVOXEL_BASE_REPOSITORY}" \
    JUSTVOXEL_HWE_REPOSITORY="${JUSTVOXEL_HWE_REPOSITORY}" \
    /ctx/build_files/finalize-hwe.sh

RUN /usr/libexec/justvoxel/health/common \
    && /usr/libexec/justvoxel/health/hwe \
    && bootc container lint --fatal-warnings

STOPSIGNAL SIGRTMIN+3
CMD ["/sbin/init"]
