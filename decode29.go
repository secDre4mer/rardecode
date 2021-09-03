package rardecode

import (
	"errors"
	"io"
)

const (
	maxCodeSize      = 0x10000
	maxUniqueFilters = 1024
)

var (
	// Errors marking the end of the decoding block and/or file
	endOfFile         = errors.New("rardecode: end of file")
	endOfBlock        = errors.New("rardecode: end of block")
	endOfBlockAndFile = errors.New("rardecode: end of block and file")
)

// decoder29 implements the decoder interface for RAR 3.0 compression (unpack version 29)
// Decode input is broken up into 1 or more blocks. The start of each block specifies
// the decoding algorithm (ppm or lz) and optional data to initialize with.
// Block length is not stored, it is determined only after decoding an end of file and/or
// block marker in the data.
type decoder29 struct {
	br      *rarBitReader
	eof     bool       // at file eof

	// current decode function (lz or ppm).
	// When called it should perform a single decode operation, and either apply the
	// data to the window or return they raw bytes for a filter.
	decode func(w *window) ([]byte, error)

	lz  lz29Decoder  // lz decoder
	ppm ppm29Decoder // ppm decoder
}

// init initializes the decoder for decoding a new file.
func (d *decoder29) init(r io.ByteReader, reset bool) error {
	if d.br == nil {
		d.br = newRarBitReader(r)
	} else {
		d.br.reset(r)
	}
	d.eof = false
	if reset {
		d.lz.reset()
		d.ppm.reset()
		d.decode = nil
	}
	if d.decode == nil {
		return d.readBlockHeader()
	}
	return nil
}

// readBlockHeader determines and initializes the current decoder for a new decode block.
func (d *decoder29) readBlockHeader() error {
	d.br.alignByte()
	n, err := d.br.readBits(1)
	if err == nil {
		if n > 0 {
			d.decode = d.ppm.decode
			err = d.ppm.init(d.br)
		} else {
			d.decode = d.lz.decode
			err = d.lz.init(d.br)
		}
	}
	if err == io.EOF {
		err = errDecoderOutOfData
	}
	return err

}

func (d *decoder29) fill(w *window) ([]*filterBlock, error) {
	if d.eof {
		return nil, io.EOF
	}

	var fl []*filterBlock

	for w.available() > 0 {
		_, err := d.decode(w) // perform a single decode operation

		switch err {
		case nil:
			continue
		case endOfBlock:
			err = d.readBlockHeader()
			if err == nil {
				continue
			}
		case endOfFile:
			d.eof = true
			err = io.EOF
		case endOfBlockAndFile:
			d.eof = true
			d.decode = nil // clear decoder, it will be setup by next init()
			err = io.EOF
		case io.EOF:
			err = errDecoderOutOfData
		}
		return fl, err
	}
	// return filters
	return fl, nil
}
