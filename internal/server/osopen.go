package server

import "os"

var osOpen = func(p string) (*os.File, error) {
	return os.Open(p)
}
