//go:build integration

package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"worldcup/internal/testsupport"
)

const testPw = "s3cret"

func newTestServer(t *testing.T) (string, *pgxpool.Pool) {
	pool := testsupport.NewPostgres(t)
	h := New(pool, nil, testPw)
	srv := httptest.NewServer(h.Router())
	t.Cleanup(srv.Close)
	return srv.URL, pool
}

func do(t *testing.T, method, url, pw string, body any) (int, map[string]any) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	rq, err := http.NewRequest(method, url, r)
	if err != nil {
		t.Fatal(err)
	}
	rq.Header.Set("Content-Type", "application/json")
	if pw != "" {
		rq.Header.Set("X-Settings-Password", pw)
	}
	resp, err := http.DefaultClient.Do(rq)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if len(b) > 0 {
		_ = json.Unmarshal(b, &out)
	}
	return resp.StatusCode, out
}

func num(v any) int { return int(v.(float64)) }

// TestParticipantsAndAssignAPI drives the real router (with the settings-password
// middleware) over HTTP: public reads stay open, host actions require the
// password, assigning is idempotent, and delete/reset work.
func TestParticipantsAndAssignAPI(t *testing.T) {
	base, pool := newTestServer(t)
	testsupport.SeedTeams(t, pool, 48)

	// --- password gate on writes ---
	if code, _ := do(t, "POST", base+"/api/participants", "", map[string]any{"name": "Alice"}); code != http.StatusUnauthorized {
		t.Errorf("add without password = %d, want 401", code)
	}
	if code, _ := do(t, "POST", base+"/api/participants", "wrong", map[string]any{"name": "Alice"}); code != http.StatusUnauthorized {
		t.Errorf("add with wrong password = %d, want 401", code)
	}

	// --- add participants ---
	code, body := do(t, "POST", base+"/api/participants", testPw, map[string]any{"name": "Alice"})
	if code != http.StatusCreated {
		t.Fatalf("add Alice = %d, want 201", code)
	}
	aliceID := int64(num(body["id"]))
	if code, _ := do(t, "POST", base+"/api/participants", testPw, map[string]any{"name": "Alice"}); code != http.StatusConflict {
		t.Errorf("duplicate name = %d, want 409", code)
	}
	do(t, "POST", base+"/api/participants", testPw, map[string]any{"name": "Bob"})

	// --- public list ---
	code, body = do(t, "GET", base+"/api/participants", "", nil)
	if code != http.StatusOK {
		t.Fatalf("list = %d, want 200", code)
	}
	if num(body["total"]) != 2 || num(body["assigned"]) != 0 {
		t.Errorf("list total=%v assigned=%v, want 2/0", body["total"], body["assigned"])
	}

	// --- assign gated + works + idempotent ---
	if code, _ := do(t, "POST", base+"/api/assign", "", map[string]any{"participant_id": aliceID}); code != http.StatusUnauthorized {
		t.Errorf("assign without password = %d, want 401", code)
	}
	code, body = do(t, "POST", base+"/api/assign", testPw, map[string]any{"participant_id": aliceID})
	if code != http.StatusOK {
		t.Fatalf("assign = %d, want 200", code)
	}
	asg := body["assignment"].(map[string]any)
	team1, _ := asg["team_name"].(string)
	if team1 == "" {
		t.Fatal("assign returned no team")
	}
	_, body = do(t, "POST", base+"/api/assign", testPw, map[string]any{"participant_id": aliceID})
	if again := body["assignment"].(map[string]any)["team_name"].(string); again != team1 {
		t.Errorf("assign not idempotent: %q then %q", team1, again)
	}
	if _, body = do(t, "GET", base+"/api/participants", "", nil); num(body["assigned"]) != 1 {
		t.Errorf("after assign, assigned=%v, want 1", body["assigned"])
	}

	// --- verify endpoint ---
	if code, _ := do(t, "POST", base+"/api/settings/verify", testPw, nil); code != http.StatusOK {
		t.Errorf("verify with password = %d, want 200", code)
	}
	if code, _ := do(t, "POST", base+"/api/settings/verify", "", nil); code != http.StatusUnauthorized {
		t.Errorf("verify without password = %d, want 401", code)
	}

	// --- delete + reset ---
	if code, _ := do(t, "DELETE", base+"/api/participants/"+strconv.FormatInt(aliceID, 10), testPw, nil); code != http.StatusOK {
		t.Errorf("delete = %d, want 200", code)
	}
	if _, body = do(t, "GET", base+"/api/participants", "", nil); num(body["total"]) != 1 {
		t.Errorf("after delete, total=%v, want 1", body["total"])
	}
	do(t, "POST", base+"/api/reset", testPw, nil)
	if _, body = do(t, "GET", base+"/api/participants", "", nil); num(body["assigned"]) != 0 {
		t.Errorf("after reset, assigned=%v, want 0", body["assigned"])
	}
}
