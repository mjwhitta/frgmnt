package frgmnt

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"

	"github.com/mjwhitta/errors"
	"github.com/mjwhitta/pathname"
)

// FragHandler is a function pointer that will operate on each
// fragment when Each() is called.
type FragHandler func(fragNum int, numFrags int, data []byte) error

// Streamer is a type that can convert a blob of data into a stream of
// fragments.
type Streamer struct {
	FragmentSize int
	NumFrags     int
	sha          hash.Hash
	stream       io.ReadSeeker
}

// NewByteStreamer will return a pointer to a new Streamer instance
// from a []byte.
func NewByteStreamer(data []byte, fragSize int) *Streamer {
	return NewStreamer(bytes.NewReader(data), len(data), fragSize)
}

// NewFileStreamer will return a pointer to a new Streamer instance
// from a file path.
func NewFileStreamer(path string, fragSize int) (*Streamer, error) {
	var e error
	var f *os.File
	var fi os.FileInfo
	var ok bool

	// Check if file exists
	if ok, e = pathname.DoesExist(path); e != nil {
		return nil, errors.Newf("file %s not accessible: %w", path, e)
	} else if !ok {
		return nil, errors.Newf("file %s not found", path)
	}

	// Open file
	if f, e = os.Open(path); e != nil {
		e = errors.Newf("failed to open %s: %w", path, e)
		return nil, e
	}

	// Get file stats
	if fi, e = f.Stat(); e != nil {
		e = errors.Newf("failed to get file info for %s: %w", path, e)
		return nil, e
	}

	// Check if file is directory
	if fi.IsDir() {
		return nil, errors.Newf("%s is a directory", path)
	}

	return NewStreamer(f, int(fi.Size()), fragSize), nil
}

// NewStreamer will return a pointer to a new Streamer instance from a
// ReadSeeker.
func NewStreamer(
	r io.ReadSeeker,
	streamSize int,
	fragSize int,
) *Streamer {
	var frags int

	// Use default
	if fragSize == 0 {
		fragSize = FragmentSize
	}

	// Determine number of fragments
	frags = streamSize / fragSize
	if streamSize%fragSize > 0 {
		frags++
	}

	return &Streamer{
		FragmentSize: fragSize,
		NumFrags:     frags,
		sha:          nil,
		stream:       r,
	}
}

// Each will call the specified FragHandler for each fragment in
// numerical order.
func (s *Streamer) Each(handler FragHandler) error {
	var e error
	var frag []byte = make([]byte, s.FragmentSize)
	var n int
	var offset int64

	// Start at beginning
	if offset, e = s.stream.Seek(0, io.SeekStart); e != nil {
		e = errors.Newf("failed to seek to start: %w", e)
		return e
	} else if offset != 0 {
		return errors.New("failed to seek to start")
	}

	// Loop thru each fragment and call handler
	for i := 1; ; i++ {
		if n, e = s.stream.Read(frag[:]); (n == 0) && (e == io.EOF) {
			return nil
		} else if e != nil {
			return errors.Newf("failed to read: %w", e)
		}

		if e = handler(i, s.NumFrags, frag[:n]); e != nil {
			return errors.Newf("FragHandler returned error: %w", e)
		}
	}
}

// Hash will print a SHA256 sum of all the fragments
func (s *Streamer) Hash() string {
	if s.sha == nil {
		s.sha = sha256.New()

		_ = s.Each(
			func(fragNum int, numFrags int, data []byte) error {
				s.sha.Write(data)
				return nil
			},
		)
	}

	return hex.EncodeToString(s.sha.Sum([]byte{}))
}
