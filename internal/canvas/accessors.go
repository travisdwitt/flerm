package canvas

func (c *Canvas) Boxes() []Box { return c.boxes }

func (c *Canvas) Texts() []Text { return c.texts }

func (c *Canvas) Connections() []Connection { return c.connections }

func (c *Canvas) Reset() {
	c.boxes = c.boxes[:0]
	c.texts = c.texts[:0]
	c.connections = c.connections[:0]
	c.highlights = make(map[string]int)
}
