package raster

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"hash/crc32"
)

// encodePNG encodes premultiplied RGBA pixels as a non-interlaced 8-bit RGBA
// PNG with no ancillary chunks, so equal pixels always give equal bytes.
//
// image/png can't be used as is: it writes opaque images as RGB.
func encodePNG(w, h int, premul []byte) ([]byte, error) {
	if w <= 0 || h <= 0 || len(premul) != w*h*4 {
		return nil, errors.New("encoding PNG: bad pixel buffer")
	}
	var buf bytes.Buffer
	buf.WriteString("\x89PNG\r\n\x1a\n")

	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], uint32(w))
	binary.BigEndian.PutUint32(ihdr[4:], uint32(h))
	ihdr[8] = 8 // bit depth
	ihdr[9] = 6 // color type: RGBA
	writeChunk(&buf, "IHDR", ihdr)

	var idat bytes.Buffer
	zw, err := zlib.NewWriterLevel(&idat, zlib.DefaultCompression)
	if err != nil {
		return nil, err
	}
	stride := w * 4
	prev := make([]byte, stride)
	cur := make([]byte, stride)
	// filtered[f] holds the row with filter type f, after its type byte.
	var filtered [5][]byte
	for i := range filtered {
		filtered[i] = make([]byte, stride+1)
		filtered[i][0] = byte(i)
	}
	for y := 0; y < h; y++ {
		row := premul[y*stride : (y+1)*stride]
		for i := 0; i < stride; i += 4 {
			r, g, b, a := row[i], row[i+1], row[i+2], row[i+3]
			if a != 0 && a != 255 {
				r = demul(r, a)
				g = demul(g, a)
				b = demul(b, a)
			}
			cur[i], cur[i+1], cur[i+2], cur[i+3] = r, g, b, a
		}
		if _, err := zw.Write(filterRow(cur, prev, &filtered)); err != nil {
			return nil, err
		}
		prev, cur = cur, prev
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	writeChunk(&buf, "IDAT", idat.Bytes())
	writeChunk(&buf, "IEND", nil)
	return buf.Bytes(), nil
}

func demul(c, a byte) byte {
	v := (uint32(c)*255 + uint32(a)/2) / uint32(a)
	if v > 255 {
		v = 255
	}
	return byte(v)
}

func writeChunk(buf *bytes.Buffer, typ string, data []byte) {
	var n [4]byte
	binary.BigEndian.PutUint32(n[:], uint32(len(data)))
	buf.Write(n[:])
	crc := crc32.NewIEEE()
	crc.Write([]byte(typ))
	crc.Write(data)
	buf.WriteString(typ)
	buf.Write(data)
	binary.BigEndian.PutUint32(n[:], crc.Sum32())
	buf.Write(n[:])
}

// filterRow picks the filter with the smallest sum of absolute differences,
// the heuristic the PNG spec suggests and image/png uses.
func filterRow(cur, prev []byte, out *[5][]byte) []byte {
	const bpp = 4
	n := len(cur)
	none, sub, up, avg, paeth := out[0][1:], out[1][1:], out[2][1:], out[3][1:], out[4][1:]
	copy(none, cur)
	for i := 0; i < n; i++ {
		var left, upLeft byte
		if i >= bpp {
			left = cur[i-bpp]
			upLeft = prev[i-bpp]
		}
		above := prev[i]
		sub[i] = cur[i] - left
		up[i] = cur[i] - above
		avg[i] = cur[i] - byte((int(left)+int(above))/2)
		paeth[i] = cur[i] - paethPredictor(left, above, upLeft)
	}
	best, bestSum := 0, -1
	for f := 0; f < 5; f++ {
		sum := 0
		for _, b := range out[f][1:] {
			sum += absByte(b)
		}
		if bestSum < 0 || sum < bestSum {
			best, bestSum = f, sum
		}
	}
	return out[best]
}

func absByte(b byte) int {
	if b < 128 {
		return int(b)
	}
	return 256 - int(b)
}

func paethPredictor(a, b, c byte) byte {
	p := int(a) + int(b) - int(c)
	pa, pb, pc := abs(p-int(a)), abs(p-int(b)), abs(p-int(c))
	if pa <= pb && pa <= pc {
		return a
	}
	if pb <= pc {
		return b
	}
	return c
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
