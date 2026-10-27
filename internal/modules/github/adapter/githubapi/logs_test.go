package githubapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/github/domain"
)

// logRig is a GitHub API that answers logs with a redirect to a separate storage server.
type logRig struct {
	client   *Client
	storage  *httptest.Server
	storeSaw string
}

func newLogRig(t *testing.T, jobs string) *logRig {
	t.Helper()

	rig := &logRig{}
	rig.storage = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rig.storeSaw = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("step 1\nboom: exit code 1\n"))
	}))

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r/actions/runs/7/jobs":
			_, _ = w.Write([]byte(jobs))
		case "/repos/o/r/actions/jobs/11/logs":
			http.Redirect(w, r, rig.storage.URL+"/blob", http.StatusFound)
		case "/repos/o/r/actions/jobs/12/logs":
			http.Error(w, "no", http.StatusForbidden)
		default:
			http.NotFound(w, r)
		}
	}))

	t.Cleanup(rig.storage.Close)
	t.Cleanup(api.Close)

	rig.client = New("id", "secret", "http://cb", time.Now)
	rig.client.api = api.URL
	rig.client.logs = rig.storage.Client()

	return rig
}

func TestJobLogFollowsTheRedirectWithoutSendingTheToken(t *testing.T) {
	rig := newLogRig(t, "")

	log, err := rig.client.JobLog(context.Background(), "ghu_secret", "o/r", 11)
	if err != nil {
		t.Fatalf("JobLog: %v", err)
	}

	if log.Text != "step 1\nboom: exit code 1\n" {
		t.Errorf("text = %q", log.Text)
	}

	if rig.storeSaw != "" {
		t.Errorf("the storage host was sent %q; it must get no token", rig.storeSaw)
	}
}

func TestJobLogTellsDeniedFromMissing(t *testing.T) {
	rig := newLogRig(t, "")

	if _, err := rig.client.JobLog(context.Background(), "t", "o/r", 12); !errors.Is(err, domain.ErrLogsDenied) {
		t.Errorf("403 gave %v, want ErrLogsDenied", err)
	}

	if _, err := rig.client.JobLog(context.Background(), "t", "o/r", 99); !errors.Is(err, domain.ErrLogsGone) {
		t.Errorf("404 gave %v, want ErrLogsGone", err)
	}
}

func TestFailedJobLogsReadsOnlyTheFailedJobs(t *testing.T) {
	jobs := `{"jobs":[{"id":10,"name":"build","conclusion":"success"},{"id":11,"name":"Deploy","conclusion":"failure"}]}`
	rig := newLogRig(t, jobs)

	logs, err := rig.client.FailedJobLogs(context.Background(), "t", "o/r", 7)
	if err != nil {
		t.Fatalf("FailedJobLogs: %v", err)
	}

	if len(logs) != 1 || logs[0].Name != "Deploy" || logs[0].Conclusion != "failure" {
		t.Errorf("logs = %+v, want only the failed Deploy job", logs)
	}
}
