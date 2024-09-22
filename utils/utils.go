package utils

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"unicode"
)

func ExtractNumber(s string) int {
	n := ""
	for _, c := range s {
		if unicode.IsDigit(c) {
			n += string(c)
		} else {
			break
		}
	}

	if len(n) == 0 {
		return 0
	}

	i, err := strconv.ParseInt(n, 10, 64)
	if err != nil {
		return 0
	}

	return int(i)
}

func CopyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

