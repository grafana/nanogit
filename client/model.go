package client

import "errors"

var (
	ErrRefNotFound = errors.New("ref not found")
)

type Ref struct {
	Name string
	Hash string
}
