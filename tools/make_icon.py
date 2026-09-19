# -*- coding: utf-8 -*-
"""Generate share/icons/devswitch.ico (multi-size) with Pillow.

Same artwork as the tray icon: dark navy square, blue circle, white swap mark.
Run:  python tools/make_icon.py   (requires Pillow)
"""
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))


def draw(size):
    from PIL import Image, ImageDraw

    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    # 深蓝底
    draw.rectangle([0, 0, size - 1, size - 1], fill=(15, 23, 41, 255))
    # 蓝色圆
    r = int(size * 0.62)
    off = (size - r) // 2
    draw.ellipse([off, off, off + r, off + r], fill=(28, 77, 212, 255))
    # 白色切换标记：横线 + 端点圆
    line_w = max(2, int(size * 0.11))
    y = size // 2
    x1 = int(size * 0.30)
    x2 = int(size * 0.62)
    draw.line([x1, y, x2, y], fill=(247, 250, 253, 255), width=line_w)
    d = max(3, int(size * 0.20))
    cx = int(size * 0.68)
    draw.ellipse([cx - d // 2, y - d // 2, cx + d // 2, y + d // 2], fill=(247, 250, 253, 255))
    return img


def main():
    try:
        from PIL import Image
    except ImportError:
        print("需要 Pillow：python -m pip install --user Pillow", file=sys.stderr)
        return 1
    dest = ROOT / "share" / "icons" / "devswitch.ico"
    dest.parent.mkdir(parents=True, exist_ok=True)
    base = draw(256)
    base.save(
        dest,
        format="ICO",
        sizes=[(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)],
    )
    print("wrote", dest)
    return 0


if __name__ == "__main__":
    sys.exit(main())
