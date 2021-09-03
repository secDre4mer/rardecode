package rardecode

import (
	"encoding/binary"
)

const (
	fileSize = 0x1000000
)

func filterE8(c byte, v5 bool, buf []byte, offset int64) ([]byte, error) {
	off := int32(offset)
	for b := buf; len(b) >= 5; {
		ch := b[0]
		b = b[1:]
		off++
		if ch != 0xe8 && ch != c {
			continue
		}
		if v5 {
			off %= fileSize
		}
		addr := int32(binary.LittleEndian.Uint32(b))
		if addr < 0 {
			if addr+off >= 0 {
				binary.LittleEndian.PutUint32(b, uint32(addr+fileSize))
			}
		} else if addr < fileSize {
			binary.LittleEndian.PutUint32(b, uint32(addr-off))
		}
		off += 4
		b = b[4:]
	}
	return buf, nil
}

func filterDelta(n int, buf []byte) ([]byte, error) {
	var res []byte
	l := len(buf)
	if cap(buf) >= 2*l {
		res = buf[l : 2*l] // use unused capacity
	} else {
		res = make([]byte, l, 2*l)
	}

	i := 0
	for j := 0; j < n; j++ {
		var c byte
		for k := j; k < len(res); k += n {
			c -= buf[i]
			i++
			res[k] = c
		}
	}
	return res, nil
}

func filterArm(buf []byte, offset int64) ([]byte, error) {
	for i := 0; len(buf)-i > 3; i += 4 {
		if buf[i+3] == 0xeb {
			n := uint(buf[i])
			n += uint(buf[i+1]) * 0x100
			n += uint(buf[i+2]) * 0x10000
			n -= (uint(offset) + uint(i)) / 4
			buf[i] = byte(n)
			buf[i+1] = byte(n >> 8)
			buf[i+2] = byte(n >> 16)
		}
	}
	return buf, nil
}