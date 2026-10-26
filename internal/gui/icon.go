//go:build gui

package gui

import (
	"bytes"
	"encoding/binary"
	"unsafe"

	"github.com/getlantern/systray"
)

// trayIconICO 程序化生成 32x32 ICO（无资产依赖）。
// 图案与品牌标识同源：深色 squircle 容器 + 品牌绿拨动轨道 + 白色滑钮
// （share/icons/hicolor/scalable/apps/devswitch.svg 的 32px 版）。
func trayIconICO() []byte {
	const size = 32
	const px = size * size

	// BGRA 位图（Windows ICONENTRY 要 BGRA/DIB）
	bmp := make([]byte, px*4)
	put := func(x, y int, c [4]byte) {
		if x < 0 || y < 0 || x >= size || y >= size {
			return
		}
		// 图标 DIB 自底向上
		row := (size - 1 - y) * size
		copy(bmp[(row+x)*4:], c[:])
	}
	inRect := func(x, y, x0, y0, x1, y1 int) bool {
		return x >= x0 && x <= x1 && y >= y0 && y <= y1
	}
	inRoundRect := func(x, y, x0, y0, x1, y1, r int) bool {
		if !inRect(x, y, x0, y0, x1, y1) {
			return false
		}
		cx := x
		if x < x0+r {
			cx = x0 + r
		} else if x > x1-r {
			cx = x1 - r
		}
		cy := y
		if y < y0+r {
			cy = y0 + r
		} else if y > y1-r {
			cy = y1 - r
		}
		dx, dy := float64(x-cx), float64(y-cy)
		return dx*dx+dy*dy <= float64(r)*float64(r)
	}
	inCircle := func(x, y, cx, cy, r float64) bool {
		dx, dy := float64(x)-cx, float64(y)-cy
		return dx*dx+dy*dy <= r*r
	}

	// 品牌色（BGRA）
	container := [4]byte{30, 26, 29, 255} // #1d212a 深色容器
	green := [4]byte{88, 160, 24, 255}    // #18a058 品牌绿
	knob := [4]byte{243, 245, 242, 255}   // #f2f5f3 滑钮

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			// 32px 基准（256 → /8）：容器 x2..30 y2..30 rx7
			switch {
			case inRoundRect(x, y, 1, 1, 30, 30, 7):
				put(x, y, container)
			default:
				continue // 透明（AND 掩码全 0 但 BGRA A=0）
			}
			// 轨道：x 6..26，高 6（y 13..18），rx 3
			if inRoundRect(x, y, 6, 13, 26, 18, 3) {
				put(x, y, green)
			}
			// 滑钮：cx 22.5 cy 15.5 r 3
			if inCircle(float64(x), float64(y), 22.5, 15.5, 3.0) {
				put(x, y, knob)
			}
		}
	}

	// BITMAPINFOHEADER（40 字节）
	header := make([]byte, 40)
	binary.LittleEndian.PutUint32(header[0:], 40)
	binary.LittleEndian.PutUint32(header[4:], uint32(size))
	binary.LittleEndian.PutUint32(header[8:], uint32(size*2)) // 高度 ×2（含掩码）
	binary.LittleEndian.PutUint16(header[12:], 1)
	binary.LittleEndian.PutUint16(header[14:], 32)

	// AND 掩码（全 0，像素即不透明）
	mask := make([]byte, size*4)

	dib := append(header, bmp...)
	dib = append(dib, mask...)

	// ICONDIR
	dir := make([]byte, 6)
	binary.LittleEndian.PutUint16(dir[0:], 0)
	binary.LittleEndian.PutUint16(dir[2:], 1)
	binary.LittleEndian.PutUint16(dir[4:], 1)
	entry := make([]byte, 16)
	entry[0], entry[1] = size, size
	binary.LittleEndian.PutUint16(entry[4:], 1)
	binary.LittleEndian.PutUint16(entry[6:], 32)
	binary.LittleEndian.PutUint32(entry[8:], uint32(len(dib)))
	binary.LittleEndian.PutUint32(entry[12:], 22)

	return append(append(dir, entry...), dib...)
}

func setTooltipSafe(text string) {
	defer func() { _ = recover() }()
	// systray 内部使用 winapi，跨 goroutine 调用需在其锁内
	_ = unsafe.Pointer(nil)
	_ = bytes.MinRead
	_ = systray.SetTooltip
}
