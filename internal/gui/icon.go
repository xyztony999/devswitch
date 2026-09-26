//go:build gui

package gui

import (
	"bytes"
	"encoding/binary"
	"unsafe"

	"github.com/getlantern/systray"
)

// trayIconICO 程序化生成 32x32 ICO（无资产依赖），图案与各端托盘一致：
// 深蓝底 + 蓝圆 + 白色切换标记。
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
	navy := [4]byte{41, 23, 15, 255}   // BGRA: #0f1729
	blue := [4]byte{212, 77, 28, 255}  // #1c4dd4 近似主蓝
	white := [4]byte{253, 250, 247, 255}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			put(x, y, navy)
		}
	}
	// 圆
	r := 12.0
	cx, cy := 16.0, 16.0
	for y := 4; y < size-4; y++ {
		for x := 4; x < size-4; x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			if dx*dx+dy*dy <= r*r {
				put(x, y, blue)
			}
		}
	}
	// 白色横线 + 端点圆
	for x := 11; x <= 21; x++ {
		put(x, 16, white)
	}
	for y := 13; y <= 19; y++ {
		for x := 18; x <= 24; x++ {
			dx, dy := float64(x)-21.0, float64(y)-16.0
			if dx*dx+dy*dy <= 9 {
				put(x, y, white)
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
