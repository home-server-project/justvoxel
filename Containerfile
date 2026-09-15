ARG ALMA_REPOS_IMAGE=quay.io/almalinuxorg/10-base:10
ARG BOOTC_IMAGECTL_IMAGE=quay.io/centos-bootc/centos-bootc:stream10
ARG ALMA_BUILDER_IMAGE=quay.io/almalinuxorg/10-kitten-base:10-kitten
ARG JUSTVOXEL_VM_REPOSITORY=ghcr.io/home-server-project/justvoxel-vm
ARG JUSTVOXEL_BAREMETAL_REPOSITORY=ghcr.io/home-server-project/justvoxel-baremetal

# AlmaLinux 10 minimal-plus root filesystem. rpm-ostree is build-stage plumbing
# required by the upstream rootfs construction path; JustVoxel does not use
# rpm-ostree layering or host-side rpm-ostree administration.
FROM ${ALMA_REPOS_IMAGE} AS repos
FROM ${BOOTC_IMAGECTL_IMAGE} AS imagectl
FROM ${ALMA_BUILDER_IMAGE} AS rootfs-builder

RUN dnf install -y podman bootc ostree rpm-ostree \
    && dnf clean all

COPY --from=imagectl /usr/share/doc/bootc-base-imagectl/ /usr/share/doc/bootc-base-imagectl/
COPY --from=imagectl /usr/libexec/bootc-base-imagectl /usr/libexec/bootc-base-imagectl
RUN chmod +x /usr/libexec/bootc-base-imagectl

RUN rm -rf /etc/yum.repos.d/*
COPY --from=repos /etc/yum.repos.d/*.repo /etc/yum.repos.d/
COPY --from=repos /etc/pki/rpm-gpg/RPM-GPG-KEY-AlmaLinux-10 /etc/pki/rpm-gpg/

COPY build_files/almalinux-10-minimal-plus.yaml \
    /usr/share/doc/bootc-base-imagectl/manifests/almalinux-10-minimal-plus.yaml

RUN /usr/libexec/bootc-base-imagectl build-rootfs \
    --reinject \
    --manifest=almalinux-10-minimal-plus \
    /target-rootfs

FROM scratch AS ctx
COPY build_files /build_files
COPY build_artifacts /build_artifacts
COPY system_files /system_files
COPY docs /docs
COPY templates /templates
COPY runtime /runtime
COPY mjust /mjust
COPY cosign.pub /cosign.pub

FROM scratch AS justvoxel-common
COPY --from=rootfs-builder /target-rootfs/ /

LABEL containers.bootc=1 \
      ostree.bootable=1 \
      org.opencontainers.image.vendor="Home Server Project" \
      org.opencontainers.image.source="https://github.com/home-server-project/justvoxel" \
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
      org.opencontainers.image.description="Immutable AlmaLinux 10 minimal-plus Minecraft server appliance for virtual machines" \
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
      org.opencontainers.image.description="Immutable AlmaLinux 10 minimal-plus Minecraft server appliance for physical hardware" \
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
