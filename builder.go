package frgmnt

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
)

// Builder is a type that can rebuild a stream of fragments back into
// a blob of data.
type Builder struct {
	NumFrags   int
	queue      map[int][]byte
	sha        hash.Hash
	stream     io.ReadWriter
	TotalFrags int
}

// NewBuilder will return a pointer to a new Builder instance.
func NewBuilder(r io.ReadWriter, numFrags int) *Builder {
	return &Builder{
		NumFrags:   0,
		queue:      map[int][]byte{},
		sha:        sha256.New(),
		stream:     r,
		TotalFrags: numFrags,
	}
}

// NewByteBuilder will return a pointer to a new Builder instance that
// writes to a []byte. Use Get() to get the data when finished.
func NewByteBuilder(numFrags int) *Builder {
	return NewBuilder(bytes.NewBuffer([]byte{}), numFrags)
}

// NewFileBuilder will return a pointer to a new Builder instance that
// writes to a file.
func NewFileBuilder(path string, numFrags int) (*Builder, error) {
	var e error
	var f *os.File

	if f, e = os.Create(path); e != nil {
		return nil, e
	}

	return NewBuilder(f, numFrags), nil
}

// Add will
func (b *Builder) Add(fragNum int, data []byte) error {
	var ok bool

	// Validate fragNum
	if fragNum <= 0 {
		return fmt.Errorf("Fragment ID should be greater than 0")
	} else if fragNum > b.TotalFrags {
		return fmt.Errorf("Fragment ID %d is out of bounds", fragNum)
	} else if len(data) == 0 {
		return fmt.Errorf("Fragment ID %d is empty", fragNum)
	}

	if fragNum <= b.NumFrags {
		// Throw away repeat fragments
		return nil
	} else if fragNum == (b.NumFrags + 1) {
		// Add fragment
		b.sha.Write(data)
		b.stream.Write(data)
		b.NumFrags++

		for {
			// Add fragments waiting in queue
			if data, ok = b.queue[b.NumFrags+1]; !ok {
				break
			}

			// Add fragment
			b.sha.Write(data)
			b.stream.Write(data)
			b.NumFrags++

			// Delete queued fragment
			delete(b.queue, b.NumFrags)
		}
	} else {
		// Queue fragment for later
		b.queue[fragNum] = make([]byte, len(data))
		copy(b.queue[fragNum], data)
	}

	return nil
}

// Get will return a []byte with the Builder, if the Builder was
// created with NewByteBuilder, otherwise an empty []byte.
func (b *Builder) Get() []byte {
	switch b.stream.(type) {
	case (*bytes.Buffer):
		return b.stream.(*bytes.Buffer).Bytes()
	default:
		return []byte{}
	}
}

// Hash will print a SHA256 sum of all the fragments
func (b *Builder) Hash() (string, error) {
	var missing int = b.TotalFrags - (b.NumFrags + len(b.queue))

	// Check for missing fragments
	if missing > 0 {
		return "", fmt.Errorf("Missing %d fragments", missing)
	}

	return hex.EncodeToString(b.sha.Sum([]byte{})), nil
}
