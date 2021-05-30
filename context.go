package gosh

import (
	"encoding/json"
	"errors"
	"fmt"
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

func WrapContext(ctx Context) *context {
	return &context{ctx: ctx}
}

type context struct {
	ctx Context

	stdin          io.Reader
	stdout, stderr io.Writer

	sync.Mutex

	vars map[string]interface{}

	err string
}

func (c *context) Stdin() io.Reader {
	if c.ctx != nil {
		return c.ctx.Stdin()
	}

	return c.stdin
}

func (c *context) Stdout() io.Writer {
	if c.ctx != nil {
		return c.ctx.Stdout()
	}

	return c.stdout
}

func (c *context) Stderr() io.Writer {
	if c.ctx != nil {
		return c.ctx.Stderr()
	}

	return c.stderr
}

func (c *context) Set(k string, v interface{}) {
	if c.ctx != nil {
		c.ctx.Set(k, v)
		return
	}

	c.Lock()
	defer c.Unlock()

	if c.vars == nil {
		c.vars = map[string]interface{}{}
	}

	c.vars[k] = v
}

func (c *context) Get(k string) interface{} {
	if c.ctx != nil {
		return c.ctx.Get(k)
	}

	return c.vars[k]
}

func (c *context) Err(msg string) {
	if c.err != "" {
		panic(fmt.Sprintf("err %q is already set", c.err))
	}

	c.err = msg
}

func (c *context) GetErr() error {
	if c.err != "" {
		return errors.New(c.err)
	}

	return nil
}
