package canvas

func (c *Canvas) Boxes() []Box { return c.boxes }

func (c *Canvas) Texts() []Text { return c.texts }

func (c *Canvas) Connections() []Connection { return c.connections }
