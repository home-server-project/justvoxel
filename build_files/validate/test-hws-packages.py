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
        vm, hws = containerfile.split("FROM ${JUSTVOXEL_DEV_IMAGE} AS justvoxel-hws", 1)
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
        self.assertIn("justvoxel-dev:testing", containerfile)


if __name__ == "__main__":
    unittest.main()
