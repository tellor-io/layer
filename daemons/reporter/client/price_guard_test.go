package client

import (
	"testing"
	"time"

	"cosmossdk.io/log"
)

func TestPriceGuard_FirstSubmission(t *testing.T) {
	logger := log.NewNopLogger()
	pg := NewPriceGuard(0.5, 30*time.Minute, true, false, logger)

	queryData := []byte("test_query_data")

	shouldSubmit, reason := pg.ShouldSubmit(queryData, 3000.0)

	if !shouldSubmit {
		t.Errorf("First submission should be allowed, got blocked with reason: %s", reason)
	}
}

func TestPriceGuard_WithinThreshold(t *testing.T) {
	logger := log.NewNopLogger()
	pg := NewPriceGuard(0.5, 30*time.Minute, true, false, logger)

	queryData := []byte("test_query_data")

	// First submission
	pg.UpdateLastPrice(queryData, 3000.0)

	// Second submission within threshold (1% change)
	shouldSubmit, reason := pg.ShouldSubmit(queryData, 3030.0)

	if !shouldSubmit {
		t.Errorf("Submission within threshold should be allowed, got blocked with reason: %s", reason)
	}
}

func TestPriceGuard_ExceedsThreshold(t *testing.T) {
	logger := log.NewNopLogger()
	pg := NewPriceGuard(0.5, 30*time.Minute, true, false, logger)

	queryData := []byte("test_query_data")

	// First submission
	pg.UpdateLastPrice(queryData, 3000.0)

	// Second submission exceeds threshold (96.7% change)
	shouldSubmit, reason := pg.ShouldSubmit(queryData, 100.0)

	if shouldSubmit {
		t.Errorf("Submission exceeding threshold should be blocked")
	}

	if reason == "" {
		t.Errorf("Should have a reason for blocking")
	}
}

func TestPriceGuard_ExpiredMaxAge(t *testing.T) {
	logger := log.NewNopLogger()
	pg := NewPriceGuard(0.5, 100*time.Millisecond, true, false, logger) // Very short max age for testing

	queryData := []byte("test_query_data")

	// First submission
	pg.UpdateLastPrice(queryData, 3000.0)

	// Wait for max age to expire
	time.Sleep(150 * time.Millisecond)

	// Submission with large change should be allowed because price expired
	shouldSubmit, reason := pg.ShouldSubmit(queryData, 100.0)

	if !shouldSubmit {
		t.Errorf("Submission should be allowed when price expired, got blocked with reason: %s", reason)
	}
}

func TestPriceGuard_Disabled(t *testing.T) {
	logger := log.NewNopLogger()
	pg := NewPriceGuard(0.5, 30*time.Minute, false, false, logger) // Disabled

	queryData := []byte("test_query_data")

	// Update with a price
	pg.UpdateLastPrice(queryData, 3000.0)

	// Try to submit with huge change - should be allowed because disabled
	shouldSubmit, reason := pg.ShouldSubmit(queryData, 1.0)

	if !shouldSubmit {
		t.Errorf("When disabled, all submissions should be allowed, got blocked with reason: %s", reason)
	}
}

func TestPriceGuard_ZeroLastPrice(t *testing.T) {
	logger := log.NewNopLogger()
	pg := NewPriceGuard(0.5, 30*time.Minute, true, false, logger)

	queryData := []byte("test_query_data")

	// First submission with zero (edge case)
	pg.UpdateLastPrice(queryData, 0.0)

	// Should allow submission to avoid division by zero
	shouldSubmit, reason := pg.ShouldSubmit(queryData, 3000.0)

	if !shouldSubmit {
		t.Errorf("Submission should be allowed when last price is zero, got blocked with reason: %s", reason)
	}
}

func TestPriceGuard_OverOneHundredPercentDeviation(t *testing.T) {
	logger := log.NewNopLogger()
	pg := NewPriceGuard(2.5, 30*time.Minute, true, false, logger) // 250% threshold

	queryData := []byte("test_query_data")

	// First submission
	pg.UpdateLastPrice(queryData, 100.0)

	// 200% increase - within 250% threshold
	shouldSubmit, reason := pg.ShouldSubmit(queryData, 300.0)
	if !shouldSubmit {
		t.Errorf("200%% change should be allowed with 250%% threshold, got blocked with reason: %s", reason)
	}

	// 300% increase - exceeds 250% threshold
	shouldSubmit, reason = pg.ShouldSubmit(queryData, 400.0)
	if shouldSubmit {
		t.Errorf("300%% change should be blocked with 250%% threshold")
	}
}

func TestPriceGuard_ExactThresholdBoundary(t *testing.T) {
	logger := log.NewNopLogger()
	pg := NewPriceGuard(0.5, 30*time.Minute, true, false, logger)

	queryData := []byte("test_query_data")

	// First submission
	pg.UpdateLastPrice(queryData, 1000.0)

	// Exactly 50% change - should be ALLOWED (change > threshold, 0.5 is not > 0.5)
	shouldSubmit, reason := pg.ShouldSubmit(queryData, 1500.0)
	if !shouldSubmit {
		t.Errorf("Exactly 50%% change should be allowed with 50%% threshold (uses > not >=), got blocked with reason: %s", reason)
	}

	// Just over 50% change - should be blocked
	shouldSubmit, _ = pg.ShouldSubmit(queryData, 1501.0)
	if shouldSubmit {
		t.Errorf("50.1%% change should be blocked with 50%% threshold")
	}
}

func TestPriceGuard_MultipleQueriesIsolation(t *testing.T) {
	logger := log.NewNopLogger()
	pg := NewPriceGuard(0.5, 30*time.Minute, true, false, logger)

	queryData1 := []byte("btc_usd")
	queryData2 := []byte("eth_usd")

	// Set different baseline prices for each query
	pg.UpdateLastPrice(queryData1, 50000.0)
	pg.UpdateLastPrice(queryData2, 3000.0)

	// Large change for queryData1 should be blocked
	shouldSubmit, _ := pg.ShouldSubmit(queryData1, 100000.0) // 100% increase
	if shouldSubmit {
		t.Errorf("BTC query should be blocked on 100%% change")
	}

	// Small change for queryData2 should be allowed
	shouldSubmit, reason := pg.ShouldSubmit(queryData2, 3100.0) // 3.3% increase
	if !shouldSubmit {
		t.Errorf("ETH query should be allowed on 3.3%% change, got blocked with reason: %s", reason)
	}

	// Verify queryData1's blocked submission didn't affect queryData2
	shouldSubmit, reason = pg.ShouldSubmit(queryData2, 3050.0)
	if !shouldSubmit {
		t.Errorf("ETH query should still work independently, got blocked with reason: %s", reason)
	}
}

func TestPriceGuard_PriceIncreaseAndDecrease(t *testing.T) {
	logger := log.NewNopLogger()
	pg := NewPriceGuard(0.5, 30*time.Minute, true, false, logger)

	queryData := []byte("test_query_data")

	// Test price increase that exceeds threshold
	pg.UpdateLastPrice(queryData, 100.0)
	shouldSubmit, _ := pg.ShouldSubmit(queryData, 200.0) // 100% increase
	if shouldSubmit {
		t.Errorf("100%% price increase should be blocked with 50%% threshold")
	}

	// Test price decrease that exceeds threshold
	pg.UpdateLastPrice(queryData, 200.0)
	shouldSubmit, _ = pg.ShouldSubmit(queryData, 50.0) // 75% decrease (|50-200|/200 = 0.75)
	if shouldSubmit {
		t.Errorf("75%% price decrease should be blocked with 50%% threshold")
	}

	// Test smaller increase that should pass
	pg.UpdateLastPrice(queryData, 100.0)
	shouldSubmit, reason := pg.ShouldSubmit(queryData, 140.0) // 40% increase
	if !shouldSubmit {
		t.Errorf("40%% price increase should be allowed with 50%% threshold, got blocked with reason: %s", reason)
	}

	// Test smaller decrease that should pass
	pg.UpdateLastPrice(queryData, 200.0)
	shouldSubmit, reason = pg.ShouldSubmit(queryData, 130.0) // 35% decrease
	if !shouldSubmit {
		t.Errorf("35%% price decrease should be allowed with 50%% threshold, got blocked with reason: %s", reason)
	}
}

func TestPriceGuard_Volatile(t *testing.T) {
	logger := log.NewNopLogger()
	pg := NewPriceGuard(0.3, 30*time.Minute, true, false, logger) // 30% threshold

	queryData := []byte("btc_usd")

	// Simulate price jump then continue up
	prices := []float64{50000, 80000, 85000, 90000, 95000, 10000, 100000}

	// First price establishes baseline
	pg.UpdateLastPrice(queryData, prices[0])

	// 50k -> 80k = 60% increase, should block
	shouldSubmit, _ := pg.ShouldSubmit(queryData, prices[1])
	if shouldSubmit {
		t.Errorf("60%% jump should be blocked with 30%% threshold")
	}
	// But baseline updates to 80k
	pg.UpdateLastPrice(queryData, prices[1])

	// 80k -> 85k = 6.25%, should allow
	shouldSubmit, reason := pg.ShouldSubmit(queryData, prices[2])
	if !shouldSubmit {
		t.Errorf("6.25%% change should be allowed, got blocked with reason: %s", reason)
	}
	pg.UpdateLastPrice(queryData, prices[2])

	// 85k -> 90k = 5.88%, should allow
	shouldSubmit, reason = pg.ShouldSubmit(queryData, prices[3])
	if !shouldSubmit {
		t.Errorf("5.88%% change should be allowed, got blocked with reason: %s", reason)
	}
	pg.UpdateLastPrice(queryData, prices[3])

	// 90k -> 95k = 5.55%, should allow
	shouldSubmit, reason = pg.ShouldSubmit(queryData, prices[4])
	if !shouldSubmit {
		t.Errorf("5.55%% change should be allowed, got blocked with reason: %s", reason)
	}
	pg.UpdateLastPrice(queryData, prices[4])

	// 95k to 10k
	shouldSubmit, reason = pg.ShouldSubmit(queryData, prices[5])
	if shouldSubmit {
		t.Errorf("huge move downshould be blocked, got allowed with reason: %s", reason)
	}
	pg.UpdateLastPrice(queryData, prices[5])

	// 10k to 100k
	shouldSubmit, reason = pg.ShouldSubmit(queryData, prices[6])
	if shouldSubmit {
		t.Errorf("huge move up should be blocked, got allowed with reason: %s", reason)
	}
	pg.UpdateLastPrice(queryData, prices[6])
}
