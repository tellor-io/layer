package notify

import (
	"strings"
	"testing"
	"time"
)

func TestFormatMessage(t *testing.T) {
	msg := Message{
		Content: "🚨 **New Dispute Detected!**",
		Embeds: []Embed{{
			Title: "⚖️ New Dispute",
			Fields: []Field{
				{Name: "📍 Dispute ID", Value: "`42`"},
				{Name: "🧱 Block Height", Value: "`100`"},
			},
			Footer:    &Footer{Text: "local-monitor"},
			Timestamp: "2026-08-06T12:00:00Z",
		}},
	}
	got := FormatMessage(msg)
	for _, want := range []string{
		"New Dispute Detected",
		"New Dispute",
		"Dispute ID: `42`",
		"Block Height: `100`",
		"local-monitor",
		"timestamp: 2026-08-06T12:00:00Z",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("FormatMessage missing %q\n got:\n%s", want, got)
		}
	}
}

func TestRateLimiterWindowAndCooldown(t *testing.T) {
	rl := NewRateLimiter()
	rl.Configure("r1", 2, time.Minute, time.Hour)
	now := time.Now()

	d1 := rl.Check("r1", now)
	if !d1.Allow || d1.EnterCooldown {
		t.Fatalf("first: %+v", d1)
	}
	rl.Record("r1", now)

	d2 := rl.Check("r1", now.Add(time.Second))
	if !d2.Allow || d2.EnterCooldown {
		t.Fatalf("second: %+v", d2)
	}
	rl.Record("r1", now.Add(time.Second))

	// Third hits max → allow this one but enter cooldown
	d3 := rl.Check("r1", now.Add(2*time.Second))
	if !d3.Allow || !d3.EnterCooldown {
		t.Fatalf("third: %+v", d3)
	}

	d4 := rl.Check("r1", now.Add(3*time.Second))
	if d4.Allow || !d4.InCooldown {
		t.Fatalf("fourth should be in cooldown: %+v", d4)
	}
}

func TestDeduper(t *testing.T) {
	d := NewDeduper(time.Hour)
	now := time.Now()
	if d.SeenOrAdd("a", now) {
		t.Fatal("first should not be seen")
	}
	if !d.SeenOrAdd("a", now) {
		t.Fatal("second should be duplicate")
	}
	if d.SeenOrAdd("b", now) {
		t.Fatal("different key")
	}
}

func TestAllowWindow(t *testing.T) {
	rl := NewRateLimiter()
	rl.Configure("log", 2, time.Minute, 0)
	now := time.Now()

	if !rl.AllowWindow("log", now) {
		t.Fatal("first should allow")
	}
	if !rl.AllowWindow("log", now.Add(time.Second)) {
		t.Fatal("second should allow")
	}
	if rl.AllowWindow("log", now.Add(2*time.Second)) {
		t.Fatal("third should be rejected within window")
	}
	// After window, allow again
	if !rl.AllowWindow("log", now.Add(time.Minute+time.Second)) {
		t.Fatal("after window should allow")
	}

	unlimited := NewRateLimiter()
	unlimited.Configure("u", 0, time.Minute, 0)
	for i := 0; i < 50; i++ {
		if !unlimited.AllowWindow("u", now.Add(time.Duration(i)*time.Millisecond)) {
			t.Fatalf("unlimited should always allow (i=%d)", i)
		}
	}
}
