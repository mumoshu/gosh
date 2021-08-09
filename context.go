package gosh

import (
	"encoding/json"
)

type FunID string

func NewFunID(f Dependency) FunID {
	bs, err := json.Marshal(f)
	if err != nil {
		panic(err)
	}

	return FunID(bs)
}
