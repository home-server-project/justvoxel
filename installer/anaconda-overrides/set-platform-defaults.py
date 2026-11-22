"""Apply JustVoxel platform partition defaults during the installer image build.

The platform partitions precede configured default_partitioning. Replace their
specifications here instead of adding duplicate boot partitions to the config.
Fail the image build if the packaged upstream specifications have changed.
"""

from importlib.util import find_spec
from pathlib import Path


package = find_spec("pyanaconda")
platform_file = Path(next(iter(package.submodule_search_locations))) / "modules/storage/platform.py"
source = platform_file.read_text()

replacements = (
    (
        '            mountpoint="/boot",\n'
        '            size=Size("2GiB")\n',
        '            mountpoint="/boot",\n'
        '            fstype="ext4",\n'
        '            size=Size("896MiB"),\n'
        '            max_size=Size("896MiB"),\n'
        '            grow=False\n',
    ),
    (
        '            mountpoint="/boot/efi",\n'
        '            fstype="efi",\n'
        '            size=Size("500MiB"),\n'
        '            max_size=Size("600MiB"),\n'
        '            grow=True\n',
        '            mountpoint="/boot/efi",\n'
        '            fstype="efi",\n'
        '            size=Size("128MiB"),\n'
        '            max_size=Size("128MiB"),\n'
        '            grow=False\n',
    ),
)

for original, replacement in replacements:
    if source.count(original) != 1:
        raise RuntimeError(f"Unexpected Anaconda platform defaults in {platform_file}")
    source = source.replace(original, replacement, 1)

platform_file.write_text(source)
