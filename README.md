# Frgmnt

<a href="https://www.buymeacoffee.com/mjwhitta">🍪 Buy me a cookie</a>

[![Go Report Card](https://goreportcard.com/badge/github.com/mjwhitta/frgmnt)](https://goreportcard.com/report/github.com/mjwhitta/frgmnt)
![Workflow](https://github.com/mjwhitta/frgmnt/actions/workflows/ci.yaml/badge.svg?event=push)

## What is this?

A simple Go library to ease fragmenting data and putting it back
together again.

## How to install

Open a terminal and run the following:

```
$ go get --ldflags="-s -w" --trimpath -u github.com/mjwhitta/frgmnt
```

## Usage

```
package main

import (
    "crypto/rand"
    "fmt"

    "github.com/mjwhitta/frgmnt"
)

func main() {
    var b *frgmnt.Builder
    var data [2 * 1024 * 1024]byte // 2MB
    var e error
    var hash string
    var n int
    var s *frgmnt.Streamer

    // Read random data
    if n, e = rand.Read(data[:]); e != nil {
        panic(e)
    }

    // Create streamer and builder
    s = frgmnt.NewByteStreamer(data[:n], 1024) // 1KB
    b = frgmnt.NewByteBuilder(s.NumFrags)

    // Simulate data transfer
    e = s.Each(
        func(fragNum int, numFrags int, data []byte) error {
            return b.Add(fragNum, data)
        },
    )
    if e != nil {
        panic(e)
    }

    // Calculate hash
    if hash, e = s.Hash(); e != nil {
        panic(e)
    }

    fmt.Println(hash)

    // Calculate hash via Builder after transfer
    if hash, e = b.Hash(); e != nil {
        panic(e)
    }

    fmt.Println(hash)
}
```

## Links

- [Source](https://github.com/mjwhitta/frgmnt)
