package tray

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"image"
	"image/draw"
	"image/png"
)

//go:embed appicon.png
var iconData []byte

// Icon returns the application icon as a 32×32 ICO file.
// fyne.io/systray on Windows uses LoadImage(LR_LOADFROMFILE, IMAGE_ICON)
// which only accepts ICO format — PNG bytes are silently ignored.
func Icon() []byte {
	src, err := png.Decode(bytes.NewReader(iconData))
	if err != nil {
		return iconData
	}

	// Resize to 32×32 using nearest-neighbour (stdlib only)
	const size = 32
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	srcB := src.Bounds()
	srcW := srcB.Max.X - srcB.Min.X
	srcH := srcB.Max.Y - srcB.Min.Y
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			sx := srcB.Min.X + x*srcW/size
			sy := srcB.Min.Y + y*srcH/size
			draw.Draw(dst, image.Rect(x, y, x+1, y+1), src, image.Point{sx, sy}, draw.Src)
		}
	}

	return encodeICO(dst)
}

// encodeICO encodes an NRGBA image as a 32-bit ICO file (BMP-inside-ICO format).
// Structure: ICONDIR + ICONDIRENTRY + BITMAPINFOHEADER + XOR pixels + AND mask.
func encodeICO(img *image.NRGBA) []byte {
	const (
		size       = 32
		bmpHdrSize = 40
		xorSize    = size * size * 4         // 32bpp BGRA, bottom-up
		andSize    = (size * ((size + 31) / 32) * 4) // 1bpp AND mask, DWORD-aligned rows
		imageSize  = bmpHdrSize + xorSize + andSize
		icoHdrSize = 6
		dirSize    = 16
		dataOffset = icoHdrSize + dirSize
	)

	buf := new(bytes.Buffer)
	le := binary.LittleEndian

	// ICONDIR header
	buf.Write([]byte{0, 0}) // reserved
	binary.Write(buf, le, uint16(1)) // type: icon
	binary.Write(buf, le, uint16(1)) // image count

	// ICONDIRENTRY
	buf.WriteByte(size)   // width
	buf.WriteByte(size)   // height
	buf.WriteByte(0)      // color count (0 = true colour)
	buf.WriteByte(0)      // reserved
	binary.Write(buf, le, uint16(1))  // planes
	binary.Write(buf, le, uint16(32)) // bit count
	binary.Write(buf, le, uint32(imageSize))
	binary.Write(buf, le, uint32(dataOffset))

	// BITMAPINFOHEADER
	binary.Write(buf, le, uint32(bmpHdrSize)) // biSize
	binary.Write(buf, le, int32(size))         // biWidth
	binary.Write(buf, le, int32(size*2))       // biHeight (double: XOR+AND)
	binary.Write(buf, le, uint16(1))           // biPlanes
	binary.Write(buf, le, uint16(32))          // biBitCount
	binary.Write(buf, le, uint32(0))           // biCompression (BI_RGB)
	binary.Write(buf, le, uint32(0))           // biSizeImage
	binary.Write(buf, le, int32(0))            // biXPelsPerMeter
	binary.Write(buf, le, int32(0))            // biYPelsPerMeter
	binary.Write(buf, le, uint32(0))           // biClrUsed
	binary.Write(buf, le, uint32(0))           // biClrImportant

	// XOR mask: 32bpp BGRA, rows bottom-to-top
	for y := size - 1; y >= 0; y-- {
		for x := 0; x < size; x++ {
			c := img.NRGBAAt(x, y)
			buf.WriteByte(c.B)
			buf.WriteByte(c.G)
			buf.WriteByte(c.R)
			buf.WriteByte(c.A)
		}
	}

	// AND mask: 1bpp, rows bottom-to-top, DWORD-aligned
	// All zeros = fully opaque (alpha handled by XOR mask above)
	rowBytes := (size + 31) / 32 * 4
	andRow := make([]byte, rowBytes)
	for y := 0; y < size; y++ {
		buf.Write(andRow)
	}

	return buf.Bytes()
}
