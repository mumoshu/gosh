package gosh

import (
	"encoding/json"
	"io"
	"sync"
)

type Context interface {
	Stdin() io.Reader
	Stdout() io.Writer
	Stderr() io.Writer
	Get(string) interface{}
	Set(string, interface{})
	Err(string)
	GetErr() error
}

type FunID string

func NewFunID(f Dependency) FunID {
	bs, err := json.Marshal(f)
	if err != nil {
		panic(err)
	}

	return FunID(bs)
}

type values struct {
	sync.Mutex

	vars map[string]interface{}
}

func (c *values) Set(k string, v interface{}) {
	c.Lock()
	defer c.Unlock()

	if c.vars == nil {
		c.vars = map[string]interface{}{}
	}

	c.vars[k] = v
}

func (c *values) Get(k string) interface{} {
	return c.vars[k]
}
