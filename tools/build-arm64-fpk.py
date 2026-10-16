#!/usr/bin/env python3
"""
Assemble an ARM64 (aarch64) FPK directory from the x86 FPK template + ARM64 Go binaries.

Usage:
    python build-arm64-fpk.py [--aria2c PATH] [--manager PATH] [--hook PATH] [--out DIR]

If --aria2c is not provided, a placeholder file is created instead.
The Go binaries default to build-output/aria2-manager-arm64 and build-output/aria2-hook-arm64.

After assembly, run fnpack to produce the .fpk file:
    tools/fnpack.exe build -d build-output/fpk-arm64
"""

import argparse
import hashlib
import os
import shutil
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
X86_FPK = REPO_ROOT / "fpk"
BUILD_OUTPUT = REPO_ROOT / "build-output"

DEFAULT_MANAGER = BUILD_OUTPUT / "aria2-manager-arm64"
DEFAULT_HOOK = BUILD_OUTPUT / "aria2-hook-arm64"


def sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest().upper()


def verify_elf_arm64(path: Path, label: str) -> bool:
    """Verify the file is an ELF aarch64 binary."""
    if not path.exists():
        print(f"ERROR: {label} not found at {path}")
        return False
    with open(path, "rb") as f:
        header = f.read(20)
    if header[:4] != b"\x7fELF":
        print(f"ERROR: {label} is not an ELF file")
        return False
    import struct
    e_machine = struct.unpack_from("<H", header, 18)[0]
    if e_machine != 0xB7:  # EM_AARCH64
        print(f"ERROR: {label} is not aarch64 (machine=0x{e_machine:x}, expected 0xb7)")
        return False
    print(f"OK: {label} is ELF aarch64, SHA-256={sha256_file(path)}")
    return True


def main():
    parser = argparse.ArgumentParser(description="Assemble ARM64 FPK directory")
    parser.add_argument("--aria2c", type=Path, default=None,
                        help="Path to ARM64 aria2c binary (if built)")
    parser.add_argument("--manager", type=Path, default=DEFAULT_MANAGER,
                        help=f"Path to ARM64 aria2-manager (default: {DEFAULT_MANAGER})")
    parser.add_argument("--hook", type=Path, default=DEFAULT_HOOK,
                        help=f"Path to ARM64 aria2-hook (default: {DEFAULT_HOOK})")
    parser.add_argument("--out", type=Path, default=BUILD_OUTPUT / "fpk-arm64",
                        help="Output directory (default: build-output/fpk-arm64)")
    args = parser.parse_args()

    # Verify Go binaries
    if not verify_elf_arm64(args.manager, "aria2-manager"):
        sys.exit(1)
    if not verify_elf_arm64(args.hook, "aria2-hook"):
        sys.exit(1)

    # Verify aria2c if provided
    has_aria2c = args.aria2c is not None
    if has_aria2c:
        if not verify_elf_arm64(args.aria2c, "aria2c"):
            print("WARNING: aria2c provided but not valid ARM64 ELF — will use placeholder")
            has_aria2c = False

    if not has_aria2c:
        print("\nWARNING: No valid ARM64 aria2c provided.")
        print("A placeholder file will be created. Replace it with the real binary")
        print("built by the GitHub Actions workflow (build-static-core.yml).")
        print("Download the 'aria2c-arm64' artifact from the Actions tab.\n")

    # Stage the FPK directory
    out_dir = args.out
    if out_dir.exists():
        shutil.rmtree(out_dir)
    shutil.copytree(X86_FPK, out_dir)

    # Replace Go binaries
    server_dir = out_dir / "app" / "server"
    shutil.copy2(args.manager, server_dir / "aria2-manager")
    shutil.copy2(args.hook, server_dir / "aria2-hook")
    os.chmod(server_dir / "aria2-manager", 0o755)
    os.chmod(server_dir / "aria2-hook", 0o755)

    # Replace aria2c
    if has_aria2c:
        shutil.copy2(args.aria2c, server_dir / "aria2c")
        os.chmod(server_dir / "aria2c", 0o755)
    else:
        # Write placeholder
        placeholder = server_dir / "aria2c"
        placeholder.write_text(
            "PLACEHOLDER: Replace with real ARM64 aria2c binary.\n"
            "Build it via GitHub Actions workflow: build-static-core.yml (arm64).\n"
            "Download the 'aria2c-arm64' artifact and replace this file.\n"
        )
        os.chmod(placeholder, 0o644)

    # Update manifest: platform = arm64
    manifest_path = out_dir / "manifest"
    manifest_content = manifest_path.read_text()
    manifest_content = manifest_content.replace(
        "platform              = x86",
        "platform              = arm64"
    )
    manifest_path.write_text(manifest_content)

    # Print summary
    print("\n=== FPK Directory Assembled ===")
    print(f"Output: {out_dir}")
    print(f"\nManifest:")
    print(manifest_path.read_text())

    print(f"\nServer directory:")
    for f in sorted(server_dir.iterdir()):
        size = f.stat().st_size
        print(f"  {f.name:30s} {size:>12,} B")

    # Generate inner SHA256SUMS
    sums_path = BUILD_OUTPUT / "fpk-arm64.inner.SHA256SUMS"
    print(f"\nInner SHA-256 checksums:")
    with open(sums_path, "w") as sf:
        for root, dirs, files in os.walk(out_dir):
            dirs.sort()
            for fname in sorted(files):
                fpath = Path(root) / fname
                rel = fpath.relative_to(out_dir)
                digest = sha256_file(fpath)
                line = f"{digest}  {rel}"
                print(f"  {line}")
                sf.write(line + "\n")

    if not has_aria2c:
        print(f"\n{'='*60}")
        print("NEXT STEPS:")
        print("1. Push to GitHub and trigger 'Build Static Core' workflow")
        print("   (select arch=arm64)")
        print("2. Download the 'aria2c-arm64' artifact")
        print("3. Re-run this script with --aria2c <path/to/aria2c>")
        print("4. Build FPK: tools/fnpack.exe build -d build-output/fpk-arm64")
        print(f"{'='*60}")
    else:
        print(f"\nReady to build FPK:")
        print(f"  tools/fnpack.exe build -d {out_dir}")


if __name__ == "__main__":
    main()
