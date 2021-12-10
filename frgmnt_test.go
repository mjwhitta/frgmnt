package frgmnt_test

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"testing"

	"gitlab.com/mjwhitta/frgmnt"
)

func TestFileBuilder(t *testing.T) {
	var e error
	var err string

	if runtime.GOOS != "windows" {
		err = strings.Join(
			[]string{
				"frgmnt: failed to create file /tmp",
				"open /tmp: is a directory",
			},
			": ",
		)
		if _, e = frgmnt.NewFileBuilder("/tmp", 0); e == nil {
			t.Fatalf("\ngot: nil\nwant: %s", err)
		} else if e.Error() != err {
			t.Fatalf("\ngot: %s\nwant: %s", e.Error(), err)
		}

		err = strings.Join(
			[]string{
				"frgmnt: failed to create file /noexist",
				"open /noexist: permission denied",
			},
			": ",
		)
		if _, e = frgmnt.NewFileBuilder("/noexist", 0); e == nil {
			t.Fatalf("\ngot: nil\nwant: %s", err)
		} else if e.Error() != err {
			t.Fatalf("\ngot: %s\nwant: %s", e.Error(), err)
		}
	}
}

func TestFileStreamer(t *testing.T) {
	var e error
	var err string

	if runtime.GOOS != "windows" {
		err = "frgmnt: path is a directory: /tmp"
		if _, e = frgmnt.NewFileStreamer("/tmp", 0); e == nil {
			t.Fatalf("\ngot: nil\nwant: %s", err)
		} else if e.Error() != err {
			t.Fatalf("\ngot: %s\nwant: %s", e.Error(), err)
		}

		err = "frgmnt: file not found: /noexist"
		if _, e = frgmnt.NewFileStreamer("/noexist", 0); e == nil {
			t.Fatalf("\ngot: nil\nwant: %s", err)
		} else if e.Error() != err {
			t.Fatalf("\ngot: %s\nwant: %s", e.Error(), err)
		}
	}
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
	var tmp string

	// Validate number of fragments while simulating data transfer
	e = s.Each(
		func(fragNum int, numFrags int, data []byte) error {
			var err error

			if numFrags != 2048 {
				t.Fatalf("\ngot: %d\nwant: 2048", numFrags)
			}

			if fragNum == 32 {
				save = make([]byte, len(data))
				copy(save, data)
			} else {
				if err = b2.Add(fragNum, data); err != nil {
					t.Fatalf("\ngot: %s\nwant: nil", err.Error())
				}
			}

			return b1.Add(fragNum, data)
		},
	)
	if e != nil {
		t.Fatalf("\ngot: %s\nwant: nil", e.Error())
	}

	// Calculate hash via Streamer
	if actual, e = s.Hash(); e != nil {
		t.Fatalf("\ngot: %s\nwant: nil", e.Error())
	}

	// Compare results
	if actual != expected {
		t.Fatalf("\ngot: %s\nwant: %s", actual, expected)
	}

	// Calculate hash via Builder after transfer
	if actual, e = b1.Hash(); e != nil {
		t.Fatalf("\ngot: %s\nwant: nil", e.Error())
	}

	// Compare results
	if actual != expected {
		t.Fatalf("\ngot: %s\nwant: %s", actual, expected)
	}

	// Attempt to use Builder that is missing fragment
	actual = "nil"
	if _, e = b2.Hash(); e != nil {
		actual = e.Error()
	}

	tmp = "frgmnt: missing 1 fragments"
	if actual != tmp {
		t.Fatalf("\ngot: %s\nwant: %s", actual, tmp)
	}

	// Add missing fragment
	if e = b2.Add(32, save); e != nil {
		t.Fatalf("\ngot: %s\nwant: nil", e.Error())
	}

	// Calculate hash via Builder after transfer
	if actual, e = b2.Hash(); e != nil {
		t.Fatalf("\ngot: %s\nwant: nil", e.Error())
	}

	// Compare results
	if actual != expected {
		t.Fatalf("\ngot: %s\nwant: %s", actual, expected)
	}
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
	if n, e = rand.Read(data[:]); e != nil {
		t.Fatalf("\ngot: %s\nwant: nil", e.Error())
	} else if n == 0 {
		t.Fatalf("\ngot: 0MB\nwant: 2MB")
	}

	// Calculate hash
	expected = fmt.Sprintf("%x", sha256.Sum256(data[:n]))

	// Write data to tmp files
	if f1, e = ioutil.TempFile(os.TempDir(), "frgmnt*"); e != nil {
		t.Fatalf("\ngot: %s\nwant: nil", e.Error())
	}
	defer f1.Close()
	defer os.Remove(f1.Name())

	if f2, e = ioutil.TempFile(os.TempDir(), "frgmnt*"); e != nil {
		t.Fatalf("\ngot: %s\nwant: nil", e.Error())
	}
	defer f2.Close()
	defer os.Remove(f2.Name())

	if f3, e = ioutil.TempFile(os.TempDir(), "frgmnt*"); e != nil {
		t.Fatalf("\ngot: %s\nwant: nil", e.Error())
	}
	defer f3.Close()
	defer os.Remove(f3.Name())

	f1.Write(data[:n])

	// Create Streamers and Builders
	s = frgmnt.NewByteStreamer(data[:n], 1024) // 1KB
	b1 = frgmnt.NewBuilder(r, s.NumFrags)
	b2 = frgmnt.NewByteBuilder(s.NumFrags)

	// Test
	testStreamer(t, s, b1, b2, expected)

	if len(r.Bytes()) != dataLen {
		t.Fatalf("\ngot: %d\nwant: %d", len(r.Bytes()), dataLen)
	}

	if !b2.Finished() {
		t.Fatalf("\ngot: false\nwant: true")
	}

	if data, e = b2.Get(); e != nil {
		t.Fatalf("\ngot: %s\nwant: nil", e.Error())
	}

	if len(data) != dataLen {
		t.Fatalf("\ngot: %d\nwant: %d", len(data), dataLen)
	}

	// Create Streamers and Builders
	if s, e = frgmnt.NewFileStreamer(f1.Name(), 1024); e != nil {
		t.Fatalf("\ngot: %s\nwant: nil", e.Error())
	}

	b1, e = frgmnt.NewFileBuilder(f2.Name(), s.NumFrags)
	if e != nil {
		t.Fatalf("\ngot: %s\nwant: nil", e.Error())
	}

	b2, e = frgmnt.NewFileBuilder(f3.Name(), s.NumFrags)
	if e != nil {
		t.Fatalf("\ngot: %s\nwant: nil", e.Error())
	}

	// Test
	testStreamer(t, s, b1, b2, expected)
}
