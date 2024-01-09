package keeper_test

import (
	"testing"
	"time"

	"github.com/tellor-io/layer/x/oracle/keeper"
)

func TestFindTimestampBefore(t *testing.T) {
	testCases := []struct {
		name          string
		timestamps    []time.Time
		target        time.Time
		expectedFound bool
		expectedIndex int
	}{
		{
			name:          "Empty slice",
			timestamps:    []time.Time{},
			target:        time.Unix(100, 0),
			expectedFound: false,
			expectedIndex: 0,
		},
		{
			name:          "Single timestamp before target",
			timestamps:    []time.Time{time.Unix(50, 0)},
			target:        time.Unix(100, 0),
			expectedFound: true,
			expectedIndex: 0,
		},
		{
			name:          "Single timestamp after target",
			timestamps:    []time.Time{time.Unix(150, 0)},
			target:        time.Unix(100, 0),
			expectedFound: false,
			expectedIndex: 0,
		},
		{
			name:          "Multiple timestamps, target present",
			timestamps:    []time.Time{time.Unix(50, 0), time.Unix(100, 0), time.Unix(150, 0)},
			target:        time.Unix(100, 0),
			expectedFound: true,
			expectedIndex: 1,
		},
		{
			name:          "Multiple timestamps, target not present",
			timestamps:    []time.Time{time.Unix(50, 0), time.Unix(70, 0), time.Unix(90, 0), time.Unix(110, 0), time.Unix(130, 0)},
			target:        time.Unix(100, 0),
			expectedFound: true,
			expectedIndex: 2,
		},
		{
			name:          "Multiple timestamps, target before all",
			timestamps:    []time.Time{time.Unix(200, 0), time.Unix(300, 0), time.Unix(400, 0)},
			target:        time.Unix(100, 0),
			expectedFound: false,
			expectedIndex: 0,
		},
		{
			name:          "Multiple timestamps, target after all",
			timestamps:    []time.Time{time.Unix(10, 0), time.Unix(20, 0), time.Unix(40, 0)},
			target:        time.Unix(100, 0),
			expectedFound: true,
			expectedIndex: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			found, index := keeper.FindTimestampBefore(tc.timestamps, tc.target)
			if found != tc.expectedFound || index != tc.expectedIndex {
				t.Errorf("Test '%s' failed: expected (%v, %d), got (%v, %d)",
					tc.name, tc.expectedFound, tc.expectedIndex, found, index)
			}
		})
	}
}
