#!/usr/bin/env python3
"""Pre-download specialist model weights into the local HuggingFace cache.

Why a script instead of letting the server download on first request?
- First-request inference becomes deterministic instead of "30s + 16GB
  download surprise". CI runs and demos benefit from this.
- The download path is monitorable: you see the progress bar, not a hung
  server log.
- Failed downloads (bad token, network blip) become loud at provisioning
  time, not at first inference.

Run from the host or inside the specialists container — either way the
weights land in ``~/.cache/huggingface`` and persist via the volume mount.

Usage::

    python data/download_specialists.py --models lightonocr nano-vl
    python data/download_specialists.py --models lightonocr           # just one
    python data/download_specialists.py --hf-token hf_xxx              # if gated
"""
from __future__ import annotations

import argparse
import os
import sys

# Map our short names → HF model IDs.
_MODELS = {
    "lightonocr":      "lightonai/LightOnOCR-2-1B-bbox",
    "lightonocr-base": "lightonai/LightOnOCR-2-1B",
    "nano-vl":         "nvidia/Llama-3.1-Nemotron-Nano-VL-8B-V1",
    # Parse v1.1-TC is the production table specialist (token-compressed,
    # 20% faster, preserves page order). The non-TC base is kept for
    # comparison / fallback if a future Parse 1.2 reintroduces a non-TC
    # variant we want to A/B against.
    "parse-tc":        "nvidia/NVIDIA-Nemotron-Parse-v1.1-TC",
    "parse-base":      "nvidia/NVIDIA-Nemotron-Parse-v1.1",
}


def main(argv: list[str] | None = None) -> int:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument(
        "--models",
        nargs="+",
        choices=sorted(_MODELS),
        required=True,
        help="One or more specialist nicknames to download.",
    )
    p.add_argument(
        "--hf-token",
        default=os.environ.get("HF_TOKEN", ""),
        help="HuggingFace token for gated models. Defaults to $HF_TOKEN.",
    )
    p.add_argument(
        "--cache-dir",
        default=os.environ.get(
            "HF_HOME",
            os.path.expanduser("~/.cache/huggingface"),
        ),
        help="Override cache location. Defaults to $HF_HOME or ~/.cache/huggingface.",
    )
    args = p.parse_args(argv)

    try:
        from huggingface_hub import snapshot_download
    except ImportError:
        print(
            "ERROR: huggingface_hub not installed. Run inside the specialists "
            "container OR `pip install huggingface_hub`.",
            file=sys.stderr,
        )
        return 1

    if args.hf_token:
        os.environ["HF_TOKEN"] = args.hf_token

    os.environ["HF_HOME"] = args.cache_dir

    failures: list[tuple[str, str]] = []
    for nickname in args.models:
        repo_id = _MODELS[nickname]
        print(f"\n=== Downloading {nickname} → {repo_id} ===")
        try:
            path = snapshot_download(
                repo_id=repo_id,
                cache_dir=os.path.join(args.cache_dir, "hub"),
                token=args.hf_token or None,
            )
            print(f"  OK   path={path}")
        except Exception as e:  # noqa: BLE001
            print(f"  FAIL {e}")
            failures.append((nickname, str(e)))

    if failures:
        print(f"\n{len(failures)} download(s) failed:")
        for nick, err in failures:
            print(f"  - {nick}: {err}")
        return 1

    print(f"\nAll {len(args.models)} download(s) complete.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
