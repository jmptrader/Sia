package renter

import (
	"bytes"
	"crypto/rand"
	"testing"
	"time"
)

func (h *testHost) addPiece(p uploadPiece) error {
	h.pieceMap[p.chunkIndex] = append(h.pieceMap[p.chunkIndex], pieceData{
		p.chunkIndex,
		p.pieceIndex,
		uint64(len(h.data)),
	})
	h.data = append(h.data, p.data...)
	// simulate I/O delay
	time.Sleep(h.delay)
	return nil
}

func (h *testHost) fileContract() fileContract { return fileContract{} }

// TestErasureUpload tests parallel uploading of erasure-coded data.
func TestErasureUpload(t *testing.T) {
	// generate data
	const dataSize = 777
	data := make([]byte, dataSize)
	rand.Read(data)

	// create Reed-Solomon encoder
	rsc, err := NewRSCode(2, 10)
	if err != nil {
		t.Fatal(err)
	}

	// create hosts
	hosts := make([]uploader, rsc.NumPieces())
	for i := range hosts {
		hosts[i] = &testHost{
			pieceMap: make(map[uint64][]pieceData),
			delay:    time.Duration(i) * time.Millisecond,
		}
	}
	// make one host really slow
	hosts[0].(*testHost).delay = 100 * time.Millisecond

	// upload data to hosts
	const pieceSize = 10
	f := newFile("foo", rsc, pieceSize, dataSize)
	err = f.upload(bytes.NewReader(data), hosts)
	if err != nil {
		t.Fatal(err)
	}

	// download data
	buf := new(bytes.Buffer)
	chunk := make([][]byte, rsc.NumPieces())
	for i := uint64(0); i < f.numChunks(); i++ {
		for _, h := range hosts {
			host := h.(*testHost)
			for _, p := range host.pieceMap[i] {
				chunk[p.Piece] = host.data[p.Offset : p.Offset+pieceSize]
			}
		}
		err = rsc.Recover(chunk, f.chunkSize(), buf)
		if err != nil {
			t.Fatal(err)
		}
	}
	buf.Truncate(dataSize)

	if !bytes.Equal(buf.Bytes(), data) {
		t.Fatal("recovered data does not match original")
	}

	/*
		for i, h := range hosts {
			host := h.(*testHost)
			pieces := 0
			for _, p := range host.pieceMap {
				pieces += len(p)
			}
			t.Logf("Host #: %d\tDelay: %v\t# Pieces: %v\t# Chunks: %d", i, host.delay, pieces, len(host.pieceMap))
		}
	*/
}