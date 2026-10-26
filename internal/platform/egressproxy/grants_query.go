package egressproxy

// Has reports whether a grant id is held.
func (g *Grants) Has(id string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.byID[id] != nil
}

// IDs lists the grant ids held.
func (g *Grants) IDs() []string {
	g.mu.Lock()
	defer g.mu.Unlock()

	ids := make([]string, 0, len(g.byID))
	for id := range g.byID {
		ids = append(ids, id)
	}

	return ids
}

// ClearTask revokes every grant of the task, in whatever project, and closes its tunnels. Task ids are unique, so
// this also works when only the id is known, such as for a task that has just been deleted.
func (g *Grants) ClearTask(taskID string) {
	g.mu.Lock()

	var leases []*Lease

	for _, entry := range g.byID {
		if entry.scope.Task == taskID {
			leases = append(leases, g.dropLocked(entry)...)
		}
	}

	g.mu.Unlock()

	closeAll(leases)
}
