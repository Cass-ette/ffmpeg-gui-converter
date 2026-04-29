#!/usr/bin/env python3

from __future__ import annotations

from collections import deque
from pathlib import Path
import sys

from PIL import Image


def color_distance(pixel: tuple[int, int, int], key: tuple[int, int, int]) -> int:
    return sum(abs(component - base) for component, base in zip(pixel, key))


def remove_background(image: Image.Image, threshold: int = 24) -> Image.Image:
    rgba = image.convert("RGBA")
    width, height = rgba.size
    pixels = rgba.load()
    visited: set[tuple[int, int]] = set()
    queue: deque[tuple[int, int]] = deque()

    corners = [(0, 0), (width - 1, 0), (0, height - 1), (width - 1, height - 1)]
    key = tuple(sum(pixels[x, y][i] for x, y in corners) // len(corners) for i in range(3))

    for corner in corners:
        queue.append(corner)
        visited.add(corner)

    while queue:
        x, y = queue.popleft()
        r, g, b, _ = pixels[x, y]
        if color_distance((r, g, b), key) > threshold:
            continue

        pixels[x, y] = (r, g, b, 0)

        for nx, ny in ((x - 1, y), (x + 1, y), (x, y - 1), (x, y + 1)):
            if 0 <= nx < width and 0 <= ny < height and (nx, ny) not in visited:
                visited.add((nx, ny))
                queue.append((nx, ny))

    return rgba


def main() -> int:
    if len(sys.argv) != 4:
        print("usage: make_app_icon.py <source-png> <clean-png> <icns-out>", file=sys.stderr)
        return 1

    source_png = Path(sys.argv[1]).resolve()
    clean_png = Path(sys.argv[2]).resolve()
    icns_out = Path(sys.argv[3]).resolve()

    clean_png.parent.mkdir(parents=True, exist_ok=True)
    icns_out.parent.mkdir(parents=True, exist_ok=True)

    cleaned = remove_background(Image.open(source_png))
    cleaned = cleaned.resize((1024, 1024), Image.Resampling.LANCZOS)
    cleaned.save(clean_png)
    cleaned.save(icns_out)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
