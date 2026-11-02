package application

// NotifyExternalFileChanged publishes the existing Files-tab refresh signal
// after an authorized Orchestrator file save. No Coding behavior is modified.
func (s *Service) NotifyExternalFileChanged(projectID string) {
	if s.deps.Notify != nil {
		s.deps.Notify.FilesChanged(projectID)
	}
}
