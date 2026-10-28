package apptest

import "time"

const (
	waitTimeout          = 10 * time.Second
	requestPollInterval  = 25 * time.Millisecond
	commandTimeout       = 5 * time.Minute
	testDirectoryMode    = 0o750
	testFileMode         = 0o600
	maxSandboxCount      = 4
	sitesPerUser         = 2
	maxSitesTotal        = 6
	completionTokenCount = 5
	promptTokenCount     = 10
	splitInHalf          = 2
	randomIDBytes        = 16
)
