#!/usr/bin/env python3
"""Source contract for the HWS-only package delta; no image or host changes."""
from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[2]


class HWSPackageComposition(unittest.TestCase):
    def test_gpu_firmware_is_composed_only_in_hws(self):
        packages = (ROOT / "build_files/packages-hws.env").read_text()
        validation = (ROOT / "build_files/validate/hws.sh").read_text()
        containerfile = (ROOT / "Containerfile").read_text()
        vm, hws = containerfile.split("FROM common AS justvoxel-hws", 1)
        for package in ("amd-gpu-firmware", "intel-gpu-firmware"):
            self.assertEqual(len(re.findall(r"(?m)^" + re.escape(package) + r" \\", packages)), 1)
            self.assertIn(package, validation)
            self.assertNotIn(package, vm)
        self.assertNotIn("build-hws.sh", vm)
        self.assertIn("/ctx/build_files/build-hws.sh", hws)
        build = (ROOT / "build_files/build-hws.sh").read_text()
        self.assertIn("source /ctx/build_files/packages-hws.env", build)
        self.assertIn('dnf --enablerepo=epel install -y "${hws_packages[@]}"', build)
        self.assertNotIn("rpm-ostree", build)
        self.assertIn("FROM ${HOME_SERVER_BASE_IMAGE} AS common", vm)
        self.assertIn("FROM common AS justvoxel-vm", vm)
        self.assertNotIn("COPY", hws)
        self.assertNotIn("build-webui", hws)

    def test_workflow_channels_and_image_sources(self):
        dev = (ROOT / "Containerfile.dev").read_text()
        self.assertIn('IMAGE_VARIANT_ID="justvoxel-dev"', dev)
        self.assertNotIn("build-hws.sh", dev)
        product = (ROOT / "Containerfile").read_text()
        assembly = (ROOT / ".github/actions/build-products/action.yml").read_text()
        self.assertIn('build-webui.sh "${GITHUB_SHA}"', assembly)
        for file, name, channel in [
            ("build-testing.yml", "Development", "testing"),
            ("product-testing.yml", "Testing", "testing"),
            ("build.yml", "Stable", "stable"),
        ]:
            workflow = (ROOT / ".github/workflows" / file).read_text()
            self.assertEqual(workflow.splitlines()[0], f"name: {name}")
            self.assertIn("uses: ./.github/actions/prepare-management", workflow)
            self.assertEqual(re.findall(r"justvoxel-management-agent:(\w+)", workflow), [channel])
            self.assertNotIn("justvoxel-dev:" + "stable", workflow)
            if name == "Development":
                pr = workflow.split("  validate-image:", 1)[1].split("  build:", 1)[0]
                self.assertIn("actions/build-management", pr)
                self.assertNotIn("prepare-management", pr)
                self.assertNotIn("packages: write", pr)
            else:
                self.assertIn("actions/build-products", workflow)
                for text in (workflow, assembly, product):
                    for parent in ("JUSTVOXEL_DEV_IMAGE", "justvoxel-dev:testing",
                                   "localhost/justvoxel-dev:build", "actions/build-dev"):
                        self.assertNotIn(parent, text)

    def test_each_image_finalizes_with_its_own_repository(self):
        dev = (ROOT / "Containerfile.dev").read_text()
        product = (ROOT / "Containerfile").read_text()
        assembly = (ROOT / ".github/actions/build-products/action.yml").read_text()
        finalize_hws = (ROOT / "build_files/finalize-hws.sh").read_text()
        self.assertNotIn("JUSTVOXEL_VM_REPOSITORY", dev)
        self.assertNotIn("JUSTVOXEL_VM_REPOSITORY", (ROOT / "jv-build.sh").read_text())
        for source in (product, assembly, finalize_hws):
            self.assertNotIn("JUSTVOXEL_DEV_REPOSITORY", source)
            self.assertNotIn("justvoxel-dev", source)
        for source in (dev, product, finalize_hws):
            self.assertNotIn("IMAGE_ADDITIONAL_TRUST_REPOSITORIES", source)

        vm, hws = product.split("FROM common AS justvoxel-vm", 1)[1].split("FROM common AS justvoxel-hws", 1)
        for source, role in ((dev, "DEV"), (vm, "VM"), (finalize_hws, "HWS")):
            self.assertIn(f'IMAGE_REPOSITORY="${{JUSTVOXEL_{role}_REPOSITORY}}"', source)
            self.assertIn("/ctx/build_files/finalize-image.sh", source)
        self.assertIn('JUSTVOXEL_HWS_REPOSITORY="${JUSTVOXEL_HWS_REPOSITORY}"', hws)
        self.assertIn("/ctx/build_files/finalize-hws.sh", hws)

    def test_verified_management_reuse(self):
        action = (ROOT / ".github/actions/prepare-management/action.yml").read_text()
        action += (ROOT / ".github/actions/prepare-management/extract.sh").read_text()
        for required in ("git ls-files --stage -z", "management-input", "management-base",
                         "management-api", "sha256:[0-9a-f]{64}", "actions/build-management",
                         'cosign verify --new-bundle-format=false --key cosign.pub "${package_ref}"',
                         'env://COSIGN_PRIVATE_KEY "${package_ref}"', "--digestfile",
                         'test -x build_artifacts/management/management-agent'):
            self.assertIn(required, action)
        self.assertNotIn("webui/", action)
        self.assertIn('package_ref="${MANAGEMENT_PACKAGE%:*}@${', action)

    def test_showcase_preserves_capability_checks(self):
        static = ROOT / "webui/internal/server/static"
        hardware = (static / "system-hardware.js").read_text()
        self.assertIn("open.hidden = !development", hardware)
        self.assertIn("if (busy || !snapshot?.hws || !snapshot.display_available) return", hardware)
        self.assertIn("apply.disabled = busy || !available", hardware)
        app = (static / "app.js").read_text()
        self.assertIn("firmwareButton.disabled = !firmwareAvailable", app)
        backend = ROOT / "management/cmd/justvoxel-management-agent"
        self.assertIn('if variant != "justvoxel-hws"', (backend / "ups.go").read_text())
        self.assertIn("if !status.DisplayAvailable", (backend / "system_hardware.go").read_text())


if __name__ == "__main__":
    unittest.main()
