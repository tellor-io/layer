package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tellor-io/layer/scripts/chain-monitor/internal/api"
)

func TestReportsByAggregate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tellor-io/layer/oracle/get_reports_by_aggregate/qid123/99" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		if r.URL.Query().Get("pagination.limit") != "1000" {
			t.Fatalf("pagination=%v", r.URL.Query())
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"microReports": []map[string]string{
				{"reporter": "tellor1a"},
				{"reporter": "tellor1b"},
				{"reporter": ""},
			},
		})
	}))
	defer srv.Close()

	c, err := api.NewClient(srv.URL, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.ReportsByAggregate(context.Background(), "qid123", 99)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "tellor1a" || got[1] != "tellor1b" {
		t.Fatalf("got=%v", got)
	}
}

func TestLooksLikeTendermint(t *testing.T) {
	if !api.LooksLikeTendermint("http://127.0.0.1:26657") {
		t.Fatal("expected tendermint")
	}
	if api.LooksLikeTendermint("http://127.0.0.1:1317") {
		t.Fatal("lcd should not look like tendermint")
	}
}

func TestNewClientEmpty(t *testing.T) {
	c, err := api.NewClient("", time.Second)
	if err != nil || c != nil {
		t.Fatalf("empty url: c=%v err=%v", c, err)
	}
}
