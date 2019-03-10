package hashingwriter

import (
	"hash"
	"io"
)

type HashingWriter struct {
	w         io.Writer
	newHash   func() hash.Hash
	iterStops func() int
	onSum     func(offset int, sum []byte) error

	nextStop int
	nWritten int
	nSum     int
	h        hash.Hash
}

var _ io.WriteCloser = new(HashingWriter)

func NewHashingWriter(
	w io.Writer,
	newHash func() hash.Hash,
	iterStops func() int,
	onSum func(int, []byte) error,
) *HashingWriter {
	return &HashingWriter{
		w:         w,
		newHash:   newHash,
		iterStops: iterStops,
		onSum:     onSum,
		nextStop:  iterStops(),
		h:         newHash(),
	}
}

func (h *HashingWriter) Write(bs []byte) (int, error) {
	var totalN int
	for len(bs) > 0 {
		l := h.nextStop - h.nWritten
		if l > len(bs) {
			l = len(bs)
		}
		n, err := h.w.Write(bs[:l])
		if err != nil {
			return totalN, err
		}
		_, err = h.h.Write(bs[:l])
		if err != nil {
			return totalN, err
		}
		h.nWritten += n
		if h.nWritten == h.nextStop {
			if err := h.onSum(h.nWritten, h.h.Sum(nil)); err != nil {
				return totalN, err
			}
			h.nextStop = h.iterStops()
			h.nSum += n
			h.h = h.newHash()
		}
		bs = bs[n:]
		totalN += n
	}
	return totalN, nil
}

func (h *HashingWriter) Close() error {
	if h.nSum != h.nWritten {
		sum := h.h.Sum(nil)
		if err := h.onSum(h.nWritten, sum); err != nil {
			return err
		}
	}
	return nil
}
