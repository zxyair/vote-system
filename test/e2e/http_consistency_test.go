package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type poll struct {
	ID      string           `json:"id"`
	Options []string         `json:"options"`
	Votes   map[string]int64 `json:"votes"`
}

type myVotesResponse struct {
	Votes []struct {
		PollID string `json:"poll_id"`
		Option string `json:"option"`
	} `json:"votes"`
}

type apiError struct {
	Error string `json:"error"`
}

type testClient struct {
	baseURL string
	http    *http.Client
}

func e2eBaseURLs(t *testing.T) []string {
	t.Helper()

	raw := os.Getenv("E2E_BASE_URLS")
	if raw == "" {
		raw = os.Getenv("E2E_BASE_URL")
	}
	if raw == "" {
		t.Skip("E2E_BASE_URL or E2E_BASE_URLS is not set; skipping HTTP E2E consistency tests")
	}

	parts := strings.Split(raw, ",")
	baseURLs := make([]string, 0, len(parts))
	for _, part := range parts {
		baseURL := strings.TrimRight(strings.TrimSpace(part), "/")
		if baseURL != "" {
			baseURLs = append(baseURLs, baseURL)
		}
	}
	if len(baseURLs) == 0 {
		t.Fatalf("E2E_BASE_URLS=%q does not contain a valid URL", raw)
	}
	return baseURLs
}

func newTestClient(baseURL string) testClient {
	return testClient{
		baseURL: baseURL,
		http: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (c testClient) createPoll(t *testing.T, userID, question string) poll {
	t.Helper()

	body := map[string]any{
		"question":   question,
		"options":    []string{"Go", "Redis", "Kubernetes"},
		"expires_at": time.Now().UTC().Add(30 * time.Minute).Format(time.RFC3339),
		"is_public":  true,
	}

	var p poll
	status, respBody, err := c.doJSON(context.Background(), http.MethodPost, "/polls/createPoll", userID, "", body, &p)
	if err != nil {
		t.Fatalf("create poll request failed: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("create poll status = %d, body = %s", status, respBody)
	}
	if p.ID == "" {
		t.Fatalf("create poll returned empty id: %s", respBody)
	}
	return p
}

func (c testClient) getPoll(t *testing.T, pollID, userID string) poll {
	t.Helper()

	var p poll
	status, respBody, err := c.doJSON(context.Background(), http.MethodGet, "/polls/"+pollID, userID, "", nil, &p)
	if err != nil {
		t.Fatalf("get poll request failed: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("get poll status = %d, body = %s", status, respBody)
	}
	if p.Votes == nil {
		p.Votes = map[string]int64{}
	}
	return p
}

func (c testClient) getMyVotes(t *testing.T, userID string) myVotesResponse {
	t.Helper()

	var resp myVotesResponse
	status, respBody, err := c.doJSON(context.Background(), http.MethodGet, "/users/me/votes", userID, "", nil, &resp)
	if err != nil {
		t.Fatalf("get my votes request failed: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("get my votes status = %d, body = %s", status, respBody)
	}
	return resp
}

func (c testClient) vote(ctx context.Context, pollID, userID, option, idem string) (int, string, error) {
	var ignored poll
	return c.doJSON(ctx, http.MethodPost, "/votes/"+pollID+"/vote", userID, idem, map[string]string{"option": option}, &ignored)
}

func (c testClient) doJSON(ctx context.Context, method, path, userID, idem string, body any, out any) (int, string, error) {
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, "", err
		}
		r = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, r)
	if err != nil {
		return 0, "", err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-User-Id", userID)
	if idem != "" {
		req.Header.Set("Idempotency-Key", idem)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", err
	}
	if out != nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return resp.StatusCode, string(respBody), err
		}
	}
	return resp.StatusCode, string(respBody), nil
}

func TestHTTPConsistencySingleEntry(t *testing.T) {
	baseURLs := e2eBaseURLs(t)
	client := newTestClient(baseURLs[0])

	t.Run("same user concurrent votes once", func(t *testing.T) {
		p := client.createPoll(t, "e2e_creator_same", uniqueQuestion("same-user"))
		const attempts = 30

		var success int64
		var conflicts int64
		var unexpected int64
		runConcurrent(attempts, func(i int) {
			status, body, err := client.vote(context.Background(), p.ID, "e2e_same_user", "Go", "")
			switch {
			case err != nil:
				t.Logf("vote request %d failed: %v", i, err)
				atomic.AddInt64(&unexpected, 1)
			case status == http.StatusOK:
				atomic.AddInt64(&success, 1)
			case status == http.StatusConflict:
				atomic.AddInt64(&conflicts, 1)
			default:
				t.Logf("vote request %d returned status=%d body=%s", i, status, body)
				atomic.AddInt64(&unexpected, 1)
			}
		})

		if success != 1 {
			t.Fatalf("success count = %d, want 1", success)
		}
		if conflicts != attempts-1 {
			t.Fatalf("conflict count = %d, want %d", conflicts, attempts-1)
		}
		if unexpected != 0 {
			t.Fatalf("unexpected count = %d, want 0", unexpected)
		}

		got := client.getPoll(t, p.ID, "e2e_same_user")
		if got.Votes["Go"] != 1 {
			t.Fatalf("Go votes = %d, want 1; votes=%#v", got.Votes["Go"], got.Votes)
		}
		myVotes := client.getMyVotes(t, "e2e_same_user")
		if countUserPollVotes(myVotes, p.ID) != 1 {
			t.Fatalf("my vote records for poll %s = %d, want 1; response=%#v", p.ID, countUserPollVotes(myVotes, p.ID), myVotes)
		}
	})

	t.Run("different users concurrent votes all counted", func(t *testing.T) {
		p := client.createPoll(t, "e2e_creator_many", uniqueQuestion("many-users"))
		const voters = 90
		options := []string{"Go", "Redis", "Kubernetes"}
		expected := map[string]int64{"Go": 0, "Redis": 0, "Kubernetes": 0}
		for i := 0; i < voters; i++ {
			expected[options[i%len(options)]]++
		}

		var success int64
		var unexpected int64
		runConcurrent(voters, func(i int) {
			userID := fmt.Sprintf("e2e_many_user_%03d_%d", i, time.Now().UnixNano())
			option := options[i%len(options)]
			status, body, err := client.vote(context.Background(), p.ID, userID, option, "")
			if err != nil {
				t.Logf("vote request %d failed: %v", i, err)
				atomic.AddInt64(&unexpected, 1)
				return
			}
			if status != http.StatusOK {
				t.Logf("vote request %d returned status=%d body=%s", i, status, body)
				atomic.AddInt64(&unexpected, 1)
				return
			}
			atomic.AddInt64(&success, 1)
		})

		if success != voters {
			t.Fatalf("success count = %d, want %d", success, voters)
		}
		if unexpected != 0 {
			t.Fatalf("unexpected count = %d, want 0", unexpected)
		}

		got := client.getPoll(t, p.ID, "e2e_many_reader")
		for option, want := range expected {
			if got.Votes[option] != want {
				t.Fatalf("%s votes = %d, want %d; votes=%#v", option, got.Votes[option], want, got.Votes)
			}
		}
		if totalVotes(got.Votes) != voters {
			t.Fatalf("total votes = %d, want %d; votes=%#v", totalVotes(got.Votes), voters, got.Votes)
		}
	})

	t.Run("idempotent retry does not double count", func(t *testing.T) {
		p := client.createPoll(t, "e2e_creator_idem", uniqueQuestion("idempotent"))
		const attempts = 30
		userID := "e2e_idempotent_user"
		idem := "e2e-idem-" + p.ID

		var success int64
		var unexpected int64
		runConcurrent(attempts, func(i int) {
			status, body, err := client.vote(context.Background(), p.ID, userID, "Kubernetes", idem)
			if err != nil {
				t.Logf("idempotent vote request %d failed: %v", i, err)
				atomic.AddInt64(&unexpected, 1)
				return
			}
			if status != http.StatusOK {
				t.Logf("idempotent vote request %d returned status=%d body=%s", i, status, body)
				atomic.AddInt64(&unexpected, 1)
				return
			}
			atomic.AddInt64(&success, 1)
		})

		if success != attempts {
			t.Fatalf("success count = %d, want %d", success, attempts)
		}
		if unexpected != 0 {
			t.Fatalf("unexpected count = %d, want 0", unexpected)
		}

		got := client.getPoll(t, p.ID, userID)
		if got.Votes["Kubernetes"] != 1 {
			t.Fatalf("Kubernetes votes = %d, want 1; votes=%#v", got.Votes["Kubernetes"], got.Votes)
		}
		if totalVotes(got.Votes) != 1 {
			t.Fatalf("total votes = %d, want 1; votes=%#v", totalVotes(got.Votes), got.Votes)
		}
	})
}

func TestHTTPConsistencyMultipleEntries(t *testing.T) {
	baseURLs := e2eBaseURLs(t)
	if len(baseURLs) < 2 {
		t.Skip("E2E_BASE_URLS must contain at least two comma-separated URLs for multi-entry test")
	}

	clients := make([]testClient, 0, len(baseURLs))
	for _, baseURL := range baseURLs {
		clients = append(clients, newTestClient(baseURL))
	}
	creator := clients[0]
	p := creator.createPoll(t, "e2e_creator_multi", uniqueQuestion("multi-entry"))

	const attempts = 30
	var success int64
	var conflicts int64
	var unexpected int64
	runConcurrent(attempts, func(i int) {
		client := clients[i%len(clients)]
		status, body, err := client.vote(context.Background(), p.ID, "e2e_multi_entry_user", "Redis", "")
		switch {
		case err != nil:
			t.Logf("multi-entry vote request %d failed: %v", i, err)
			atomic.AddInt64(&unexpected, 1)
		case status == http.StatusOK:
			atomic.AddInt64(&success, 1)
		case status == http.StatusConflict:
			atomic.AddInt64(&conflicts, 1)
		default:
			t.Logf("multi-entry vote request %d returned status=%d body=%s", i, status, body)
			atomic.AddInt64(&unexpected, 1)
		}
	})

	if success != 1 {
		t.Fatalf("success count = %d, want 1", success)
	}
	if conflicts != attempts-1 {
		t.Fatalf("conflict count = %d, want %d", conflicts, attempts-1)
	}
	if unexpected != 0 {
		t.Fatalf("unexpected count = %d, want 0", unexpected)
	}

	got := creator.getPoll(t, p.ID, "e2e_multi_reader")
	if got.Votes["Redis"] != 1 {
		t.Fatalf("Redis votes = %d, want 1; votes=%#v", got.Votes["Redis"], got.Votes)
	}
	if totalVotes(got.Votes) != 1 {
		t.Fatalf("total votes = %d, want 1; votes=%#v", totalVotes(got.Votes), got.Votes)
	}
}

func TestHTTPConsistencyGrpcServerRestartWithRetry(t *testing.T) {
	if os.Getenv("E2E_ENABLE_K8S_FAILURE") != "true" {
		t.Skip("E2E_ENABLE_K8S_FAILURE=true is not set; skipping grpcserver restart fault-injection test")
	}

	baseURLs := e2eBaseURLs(t)
	client := newTestClient(baseURLs[0])
	p := client.createPoll(t, "e2e_creator_failure", uniqueQuestion("grpc-restart"))

	const voters = 90
	options := []string{"Go", "Redis", "Kubernetes"}
	var success int64
	var failed int64

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < voters; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			userID := fmt.Sprintf("e2e_failure_user_%03d_%d", i, time.Now().UnixNano())
			option := options[i%len(options)]
			idem := fmt.Sprintf("e2e-failure-%s-%03d", p.ID, i)
			if voteWithRetry(client, p.ID, userID, option, idem) {
				atomic.AddInt64(&success, 1)
				return
			}
			atomic.AddInt64(&failed, 1)
		}(i)
	}

	close(start)
	restartGRPCServer(t)
	wg.Wait()
	waitForGRPCServerRollout(t)

	if success == 0 {
		t.Fatalf("successful requests = 0, want at least one successful request during fault injection")
	}

	got := client.getPollEventually(t, p.ID, "e2e_failure_reader")
	if totalVotes(got.Votes) != success {
		t.Fatalf("total votes = %d, want successful requests %d; failed=%d votes=%#v", totalVotes(got.Votes), success, failed, got.Votes)
	}
	if totalVotes(got.Votes) > voters {
		t.Fatalf("total votes = %d, want <= %d; votes=%#v", totalVotes(got.Votes), voters, got.Votes)
	}
}

func (c testClient) getPollEventually(t *testing.T, pollID, userID string) poll {
	t.Helper()

	var lastStatus int
	var lastBody string
	var lastErr error
	for attempt := 0; attempt < 10; attempt++ {
		var p poll
		status, body, err := c.doJSON(context.Background(), http.MethodGet, "/polls/"+pollID, userID, "", nil, &p)
		if err == nil && status == http.StatusOK {
			if p.Votes == nil {
				p.Votes = map[string]int64{}
			}
			return p
		}
		lastStatus = status
		lastBody = body
		lastErr = err
		time.Sleep(time.Duration(200+attempt*200) * time.Millisecond)
	}
	t.Fatalf("get poll did not recover: last status=%d body=%s err=%v", lastStatus, lastBody, lastErr)
	return poll{}
}

func voteWithRetry(client testClient, pollID, userID, option, idem string) bool {
	for attempt := 0; attempt < 8; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		status, _, err := client.vote(ctx, pollID, userID, option, idem)
		cancel()
		if err == nil && status == http.StatusOK {
			return true
		}
		if err == nil && status == http.StatusConflict {
			return false
		}
		time.Sleep(time.Duration(100+attempt*150) * time.Millisecond)
	}
	return false
}

func restartGRPCServer(t *testing.T) {
	t.Helper()
	cmd := exec.Command("kubectl", "rollout", "restart", "deployment/grpcserver", "-n", "vote-system")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("restart grpcserver failed: %v; output=%s", err, out)
	}
}

func waitForGRPCServerRollout(t *testing.T) {
	t.Helper()
	cmd := exec.Command("kubectl", "rollout", "status", "deployment/grpcserver", "-n", "vote-system", "--timeout=240s")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("waiting for grpcserver rollout failed: %v; output=%s", err, out)
	}
}

func runConcurrent(n int, fn func(i int)) {
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			fn(i)
		}(i)
	}
	close(start)
	wg.Wait()
}

func totalVotes(votes map[string]int64) int64 {
	var total int64
	for _, n := range votes {
		total += n
	}
	return total
}

func countUserPollVotes(resp myVotesResponse, pollID string) int {
	count := 0
	for _, vote := range resp.Votes {
		if vote.PollID == pollID {
			count++
		}
	}
	return count
}

func uniqueQuestion(prefix string) string {
	return fmt.Sprintf("E2E consistency %s %d", prefix, time.Now().UnixNano())
}
