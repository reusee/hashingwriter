package hashingwriter

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"testing"
)

func every(n int) func() int {
	c := 0
	return func() int {
		c += n
		return c
	}
}

var pt = fmt.Printf

func TestWriter(t *testing.T) {

	type Sum struct {
		Offset int
		Sum    []byte
	}

	bs := bytes.Repeat([]byte("hello, world!"), 65536)
	l := len("hello, world!") * 65536

	// get sums
	var sums []Sum
	w := NewHashingWriter(
		ioutil.Discard,
		sha256.New,
		every(7),
		func(offset int, sum []byte) error {
			sums = append(sums, Sum{
				Offset: offset,
				Sum:    sum,
			})
			return nil
		},
	)
	if n, err := w.Write(bs); err != nil {
		t.Fatal(err)
	} else if n != l {
		t.Fatal()
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if expected := 1 + l/7; len(sums) != expected {
		t.Fatalf("expected %d, got %d", expected, len(sums))
	}

	var lastOffset int
	for i, sum := range sums {
		if i == len(sums)-1 {
			if sum.Offset != l {
				t.Fatalf("bad offset: %v", sum)
			}
		} else {
			if sum.Offset != (i+1)*7 {
				t.Fatalf("bad offset: %v", sum)
			}
		}
		h := sha256.New()
		h.Write(bs[lastOffset:sum.Offset])
		if !bytes.Equal(h.Sum(nil), sum.Sum) {
			t.Fatalf("bad sum: %v", sum)
		}
		lastOffset = sum.Offset
	}

	sumMap := make(map[int][]byte)
	for _, sum := range sums {
		sumMap[sum.Offset] = sum.Sum
	}

	// verify writer
	w = NewHashingWriter(
		ioutil.Discard,
		sha256.New,
		every(7),
		func(offset int, sum []byte) error {
			if !bytes.Equal(sumMap[offset], sum) {
				return fmt.Errorf("bad sum at %d", offset)
			}
			return nil
		},
	)
	for _, b := range bs {
		if _, err := w.Write([]byte{b}); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// verify reader
	w = NewHashingWriter(
		ioutil.Discard,
		sha256.New,
		every(7),
		func(offset int, sum []byte) error {
			if !bytes.Equal(sumMap[offset], sum) {
				return fmt.Errorf("bad sum at %d", offset)
			}
			return nil
		},
	)
	r := io.TeeReader(bytes.NewReader(bs), w)
	if _, err := io.Copy(ioutil.Discard, r); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// bad input
	badBs := make([]byte, len(bs))
	copy(badBs, bs)
	badBs[0] = ^badBs[0]
	w = NewHashingWriter(
		ioutil.Discard,
		sha256.New,
		every(7),
		func(offset int, sum []byte) error {
			if !bytes.Equal(sumMap[offset], sum) {
				return fmt.Errorf("bad sum at %d", offset)
			}
			return nil
		},
	)
	r = io.TeeReader(bytes.NewReader(badBs), w)
	if _, err := io.Copy(ioutil.Discard, r); err == nil {
		t.Fatalf("should fail")
	} else {
		if err.Error() != "bad sum at 7" {
			t.Fatal()
		}
	}

	// bad input at the end
	badBs = make([]byte, len(bs))
	copy(badBs, bs)
	badBs[len(badBs)-1] = ^badBs[len(badBs)-1]
	w = NewHashingWriter(
		ioutil.Discard,
		sha256.New,
		every(7),
		func(offset int, sum []byte) error {
			if !bytes.Equal(sumMap[offset], sum) {
				return fmt.Errorf("bad sum at %d", offset)
			}
			return nil
		},
	)
	r = io.TeeReader(bytes.NewReader(badBs), w)
	if _, err := io.Copy(ioutil.Discard, r); err != nil {
		t.Fatal()
	}
	if err := w.Close(); err == nil {
		t.Fatalf("should fail")
	} else {
		if err.Error() != "bad sum at 851968" {
			t.Fatal()
		}
	}

}
