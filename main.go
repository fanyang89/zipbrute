package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"

	"github.com/schollz/progressbar/v3"
	"github.com/yeka/zip"
)

var charTable = "0123456789" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ" // + "abcdefghijklmnopqrstuvwxyz"

func isEnd(s []int) bool {
	c := 0
	for _, v := range s {
		c += v
	}
	return (c / (len(charTable) - 1)) == len(s)
}

func makeString(s []int) string {
	x := make([]uint8, len(s))
	for i, v := range s {
		x[i] = charTable[v]
	}
	return string(x)
}

func startsWithDigit(s string) bool {
	if len(s) == 0 {
		return false
	}
	return '0' <= s[0] && s[0] <= '9'
}

func iterAll(size int) func(yield func(string) bool) {
	indices := make([]int, size)

	return func(yield func(string) bool) {
		for !isEnd(indices) {
			if !yield(makeString(indices)) {
				return
			}

			indices[len(indices)-1]++

			for i := len(indices) - 1; i >= 1; i-- {
				if indices[i] >= len(charTable) {
					indices[i] = 0
					indices[i-1]++
				}
			}
		}

		yield(makeString(indices))
	}
}

func verifyDecrypt(password string) (ok bool, err error) {
	ok = true

	r, err := zip.OpenReader(input)
	if err != nil {
		return
	}
	defer func() { _ = r.Close() }()

	for _, f := range r.File {
		if f.IsEncrypted() {
			f.SetPassword(password)
		}

		var reader io.ReadCloser
		reader, err = f.Open()
		if err != nil {
			ok = false
			return
		}

		_, err = io.ReadAll(reader)
		_ = reader.Close()
		if err != nil {
			ok = false
			return
		}
	}

	return
}

func tryDecryptWorker(ctx context.Context, passwordC <-chan string, bar *progressbar.ProgressBar) {
	r, err := zip.OpenReader(input)
	if err != nil {
		panic(err)
	}
	defer func() { _ = r.Close() }()

	if len(r.File) == 0 {
		log.Println("Decrypt worker exited")
		return
	}

	f := r.File[0]

	for {
		select {
		case <-ctx.Done():
			return

		case password := <-passwordC:
			_ = bar.Add(1)
			if startsWithDigit(password) {
				continue
			}

			if f.IsEncrypted() {
				f.SetPassword(password)
			}

			var reader io.ReadCloser
			reader, err = f.Open()
			if err != nil {
				continue
			}

			_, err = io.ReadAll(reader)
			_ = reader.Close()
			if err != nil {
				continue
			}

			var ok bool
			ok, err = verifyDecrypt(password)
			if ok {
				fmt.Printf("Password: %s\n", password)
			}
		}
	}
}

var length int
var input string
var worker int

func intPow(n, m int64) int64 {
	if m == 0 {
		return 1
	}
	if m == 1 {
		return n
	}
	result := n
	for i := int64(2); i <= m; i++ {
		result *= n
	}
	return result
}

func main() {
	flag.IntVar(&length, "length", 1, "length of each element")
	flag.StringVar(&input, "input", "", "input filename")
	flag.IntVar(&worker, "worker", 8, "worker num")
	flag.Parse()

	if len(input) == 0 {
		fmt.Printf("input file is empty\n")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	bar := progressbar.Default(intPow(int64(len(charTable)), int64(length)))
	passwordC := make(chan string, 10000)

	for i := 0; i < worker; i++ {
		go tryDecryptWorker(ctx, passwordC, bar)
	}

	for p := range iterAll(length) {
		passwordC <- p
	}

	_ = bar.Finish()
	cancel()
}
