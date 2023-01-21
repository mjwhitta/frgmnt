package frgmnt_test

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/mjwhitta/frgmnt"
	assert "github.com/stretchr/testify/require"
)

func TestFileBuilder(t *testing.T) {
	var e error

	if runtime.GOOS == "windows" {
		t.Skip("skipping testing on Windows.")
	}

	_, e = frgmnt.NewFileBuilder("/tmp", 0)
	assert.NotNil(t, e)

	_, e = frgmnt.NewFileBuilder("/noexist/file", 0)
	assert.NotNil(t, e)
}

func TestFileStreamer(t *testing.T) {
	var e error

	if runtime.GOOS == "windows" {
		t.Skip("skipping testing on Windows.")
	}

	_, e = frgmnt.NewFileStreamer("/tmp", 0)
	assert.NotNil(t, e)

	_, e = frgmnt.NewFileStreamer("/noexist/file", 0)
	assert.NotNil(t, e)
}

func testStreamer(
	t *testing.T,
	s *frgmnt.Streamer,
	b1 *frgmnt.Builder,
	b2 *frgmnt.Builder,
	expected string,
) {
	var actual string
	var e error
	var save []byte

	// Validate number of fragments while simulating data transfer
	e = s.Each(
		func(fragNum int, numFrags int, data []byte) error {
			var e error

			assert.Equal(t, 2048, numFrags)

			if fragNum == 32 {
				save = make([]byte, len(data))
				copy(save, data)
			} else {
				e = b2.Add(fragNum, data)
				assert.Nil(t, e)
			}

			return b1.Add(fragNum, data)
		},
	)
	assert.Nil(t, e)

	// Calculate hash via Streamer and compare results
	actual = s.Hash()
	assert.Equal(t, expected, s.Hash())

	// Calculate hash via Builder after transfer
	actual, e = b1.Hash()
	assert.Nil(t, e)
	assert.Equal(t, expected, actual)

	// Attempt to use Builder that is missing fragment
	_, e = b2.Hash()
	assert.NotNil(t, e)

	// Add missing fragment
	e = b2.Add(32, save)
	assert.Nil(t, e)

	// Calculate hash via Builder after transfer
	actual, e = b2.Hash()
	assert.Nil(t, e)
	assert.Equal(t, expected, actual)
}

func TestStreamers(t *testing.T) {
	var b1 *frgmnt.Builder
	var b2 *frgmnt.Builder
	var data []byte
	var dataLen = 2 * 1024 * 1024 // 2MB
	var e error
	var expected string
	var f1 *os.File
	var f2 *os.File
	var f3 *os.File
	var n int
	var r *bytes.Buffer = bytes.NewBuffer([]byte{})
	var s *frgmnt.Streamer

	// Read random data
	data = make([]byte, dataLen)
	n, e = rand.Read(data[:])
	assert.Nil(t, e)
	assert.Equal(t, dataLen, n)

	// Calculate hash
	expected = fmt.Sprintf("%x", sha256.Sum256(data[:n]))

	// Write data to tmp files
	f1, e = os.CreateTemp(t.TempDir(), "frgmnt*")
	assert.Nil(t, e)
	assert.NotNil(t, f1)
	defer f1.Close()

	f2, e = os.CreateTemp(t.TempDir(), "frgmnt*")
	assert.Nil(t, e)
	assert.NotNil(t, f2)
	defer f2.Close()

	f3, e = os.CreateTemp(t.TempDir(), "frgmnt*")
	assert.Nil(t, e)
	assert.NotNil(t, f3)
	defer f3.Close()

	f1.Write(data[:n])

	// Create Streamers and Builders
	s = frgmnt.NewByteStreamer(data[:n], 1024) // 1KB
	b1 = frgmnt.NewBuilder(r, s.NumFrags)
	b2 = frgmnt.NewByteBuilder(s.NumFrags)

	// Test
	testStreamer(t, s, b1, b2, expected)

	assert.Equal(t, dataLen, len(r.Bytes()))
	assert.True(t, b2.Finished())

	data, e = b2.Get()
	assert.Nil(t, e)
	assert.Equal(t, dataLen, len(data))

	// Create Streamers and Builders
	s, e = frgmnt.NewFileStreamer(f1.Name(), 1024)
	assert.Nil(t, e)
	assert.NotNil(t, s)

	b1, e = frgmnt.NewFileBuilder(f2.Name(), s.NumFrags)
	assert.Nil(t, e)
	assert.NotNil(t, b1)

	b2, e = frgmnt.NewFileBuilder(f3.Name(), s.NumFrags)
	assert.Nil(t, e)
	assert.NotNil(t, b2)

	// Test
	testStreamer(t, s, b1, b2, expected)
}
