package cbytecache

// GetWithUnmarshaller gets entry bytes and apply unmarshal logic.
func (c *Cache) GetWithUnmarshaller(key string, u Unmarshaller) error {
	if u == nil {
		return ErrNoUnmarshaller
	}
	b, err := c.Get(key)
	if err != nil {
		return err
	}
	return u.Unmarshal(b)
}

// GetToWithUnmarshaller gets entry bytes to dst and apply unmarshal logic.
func (c *Cache) GetToWithUnmarshaller(dst []byte, key string, u Unmarshaller) (err error) {
	if u == nil {
		return ErrNoUnmarshaller
	}
	if dst, err = c.GetTo(dst, key); err != nil {
		return
	}
	return u.Unmarshal(dst)
}

// ExtractWithUnmarshaller extracts entry bytes and apply unmarshal logic.
func (c *Cache) ExtractWithUnmarshaller(key string, u Unmarshaller) error {
	if u == nil {
		return ErrNoUnmarshaller
	}
	b, err := c.Extract(key)
	if err != nil {
		return err
	}
	return u.Unmarshal(b)
}

// ExtractToWithUnmarshaller extracts entry bytes and apply unmarshal logic.
func (c *Cache) ExtractToWithUnmarshaller(dst []byte, key string, u Unmarshaller) (err error) {
	if u == nil {
		return ErrNoUnmarshaller
	}
	if dst, err = c.ExtractTo(dst, key); err != nil {
		return
	}
	return u.Unmarshal(dst)
}
