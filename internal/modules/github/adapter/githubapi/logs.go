package githubapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

const (
	maxLog       = 8 << 20
	maxFailed    = 3
	downloadWait = 30 * time.Second
)

// logDownloads is the client for the address GitHub redirects a log to. That address is on a storage host that
// changes, so it is never allowlisted by name: it may only reach the public internet, and it carries no token.
func logDownloads() *http.Client {
	return &http.Client{
		Timeout:       downloadWait,
		Transport:     egress.Transport(egress.Policy{Rule: egress.PublicOnly}),
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

type jobList struct {
	Jobs []struct {
		Name       string `json:"name"`
		Conclusion string `json:"conclusion"`
		ID         int64  `json:"id"`
	} `json:"jobs"`
}

// FailedJobLogs returns the logs of the jobs of the latest attempt of a run that failed or timed out.
func (c *Client) FailedJobLogs(ctx context.Context, access, repo string, runID int64) ([]domain.JobLog, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/actions/runs/%d/jobs?filter=latest&per_page=100", c.api, repo, runID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+access)

	var list jobList

	status, err := c.do(req, &list)
	if err != nil {
		return nil, err
	}

	if err = refused(status); err != nil {
		return nil, err
	}

	var logs []domain.JobLog

	for _, job := range list.Jobs {
		if job.Conclusion != "failure" && job.Conclusion != "timed_out" {
			continue
		}

		if len(logs) == maxFailed {
			break
		}

		log, err := c.JobLog(ctx, access, repo, job.ID)
		if err != nil {
			return nil, err
		}

		log.Name, log.Conclusion = job.Name, job.Conclusion
		logs = append(logs, log)
	}

	return logs, nil
}

// JobLog returns the log of one job. GitHub answers with a redirect to a short-lived address; the token is sent to
// GitHub's API only, never to that address.
func (c *Client) JobLog(ctx context.Context, access, repo string, jobID int64) (domain.JobLog, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/actions/jobs/%d/logs", c.api, repo, jobID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return domain.JobLog{}, fmt.Errorf("build request: %w", err)
	}

	setHeaders(req)
	req.Header.Set("Authorization", "Bearer "+access)

	res, err := c.http.Do(req)
	if err != nil {
		return domain.JobLog{}, fmt.Errorf("call github: %w", err)
	}

	_ = res.Body.Close()

	if res.StatusCode != http.StatusFound {
		return domain.JobLog{}, refused(res.StatusCode)
	}

	text, err := c.fetch(ctx, res.Header.Get("Location"))

	return domain.JobLog{Text: text}, err
}

func (c *Client) fetch(ctx context.Context, location string) (string, error) {
	target, err := url.Parse(location)
	if err != nil || target.Scheme != "https" || target.Host == "" {
		return "", errUnavailable
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), http.NoBody)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	res, err := c.logs.Do(req)
	if err != nil {
		return "", fmt.Errorf("download log: %w", err)
	}

	defer func() { _ = res.Body.Close() }()

	if res.StatusCode != http.StatusOK {
		return "", domain.ErrLogsGone
	}

	raw, err := io.ReadAll(io.LimitReader(res.Body, maxLog))
	if err != nil {
		return "", fmt.Errorf("read log: %w", err)
	}

	return string(raw), nil
}

// refused turns the status of a logs call into what the person is told.
func refused(status int) error {
	switch status {
	case http.StatusOK, http.StatusFound:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return domain.ErrLogsDenied
	case http.StatusNotFound, http.StatusGone:
		return domain.ErrLogsGone
	}

	return errUnavailable
}
