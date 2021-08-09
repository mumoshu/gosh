package context

import "sync"

type Variables struct {
	sync.Mutex

	vars map[string]interface{}
}

func (c *Variables) Set(k string, v interface{}) {
	c.Lock()
	defer c.Unlock()

	if c.vars == nil {
		c.vars = map[string]interface{}{}
	}

	c.vars[k] = v
}

func (c *Variables) Get(k string) interface{} {
	return c.vars[k]
}
