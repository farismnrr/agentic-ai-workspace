package egressproxy

func (g *Grants) openCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.total
}
