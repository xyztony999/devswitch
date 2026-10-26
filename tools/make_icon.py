# -*- coding: utf-8 -*-
"""Generate share/icons/devswitch.ico (multi-size) with Pillow.

Artwork: 拨动开关（品牌标识）——深色 squircle 容器 + 品牌绿激活轨道 + 白色滑钮。
16px 特例为无容器简化形（绿胶囊 + 白圆点），保证托盘尺寸可辨。
与 share/icons/hicolor/scalable/apps/devswitch.svg 同源。
Run:  python tools/make_icon.py   (requires Pillow)
"""
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

# 品牌色（与 GUI accent 一致）
BG_TOP = (38, 42, 51, 255)      # #262a33
BG_BOTTOM = (21, 24, 30, 255)   # #15181e
GREEN = (24, 160, 88, 255)      # #18a058
KNOB = (242, 245, 243, 255)     # #f2f5f3


def _squircle_bg(size):
    """深色渐变圆角容器（squircle 近似：rounded_rectangle）。"""
    from PIL import Image

    top = Image.new("RGBA", (size, size), BG_TOP)
    bottom = Image.new("RGBA", (size, size), BG_BOTTOM)
    mask = Image.new("L", (size, size), 0)
    from PIL import ImageDraw

    ImageDraw.Draw(mask).rounded_rectangle(
        [size * 8 // 256, size * 8 // 256, size * 248 // 256, size * 248 // 256],
        radius=size * 56 // 256, fill=255)
    return Image.composite(top, bottom, mask)


def draw(size):
    from PIL import Image, ImageDraw

    img = _squircle_bg(size)
    d = ImageDraw.Draw(img)

    if size <= 16:
        # 特例：无容器，纯胶囊 + 白点（16px 托盘可辨性优先）
        track_h = size * 7 // 16
        y0 = (size - track_h) // 2
        d.rounded_rectangle([0, y0, size - 1, y0 + track_h],
                            radius=track_h // 2, fill=GREEN)
        r = size * 5 // 16
        cx = size * 11 // 16
        cy = size // 2
        d.ellipse([cx - r, cy - r, cx + r, cy + r], fill=KNOB)
        return img

    # 轨道：x 52..204（256 基准），高 48，rx 24
    x0 = size * 52 // 256
    x1 = size * 204 // 256
    th = size * 48 // 256
    y0 = (size - th) // 2
    d.rounded_rectangle([x0, y0, x1, y0 + th], radius=th // 2, fill=GREEN)
    # 滑钮：cx 180，r 17
    r = size * 17 // 256
    cx = size * 180 // 256
    cy = size // 2
    d.ellipse([cx - r, cy - r, cx + r, cy + r], fill=KNOB)
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
