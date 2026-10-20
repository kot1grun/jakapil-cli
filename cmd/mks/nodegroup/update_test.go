package nodegroup

import (
	"reflect"
	"testing"

	mksnodegroup "github.com/selectel/mks-go/pkg/v1/nodegroup"
)

func taint(key, value string, effect mksnodegroup.TaintEffect) mksnodegroup.Taint {
	return mksnodegroup.Taint{Key: key, Value: value, Effect: effect}
}

func TestApplyLabels(t *testing.T) {
	testCases := []struct {
		name      string
		current   map[string]string
		adds      map[string]string
		removals  []string
		expectNil bool
		expected  map[string]string
	}{
		{
			name:     "adds and removes preserve untouched labels",
			current:  map[string]string{"env": "prod", "team": "platform"},
			adds:     map[string]string{"env": "staging", "region": "ru-1"},
			removals: []string{"team"},
			expected: map[string]string{"env": "staging", "region": "ru-1"},
		},
		{
			name:     "empty inputs keep current set untouched",
			current:  map[string]string{"a": "1"},
			expected: map[string]string{"a": "1"},
		},
		{
			name:      "clear semantics (empty current and no changes)",
			expected:  map[string]string{},
			expectNil: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := applyLabels(tc.current, tc.adds, tc.removals)
			if tc.expectNil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("applyLabels() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestApplyTaints(t *testing.T) {
	current := []mksnodegroup.Taint{
		taint("dedicated", "gpu", "NoSchedule"),
		taint("legacy", "", "NoExecute"),
	}
	adds := []mksnodegroup.Taint{taint("spot", "", "PreferNoSchedule")}
	removals := []string{"legacy"}

	// Non-clear merge: keeps dedicated, drops legacy, adds spot.
	got := applyTaints(current, adds, removals, false)
	assertTaintsEqual(t, got, []mksnodegroup.Taint{
		taint("dedicated", "gpu", "NoSchedule"),
		taint("spot", "", "PreferNoSchedule"),
	})

	// Clear: discards current, keeps only additions.
	gotClear := applyTaints(current, adds, nil, true)
	assertTaintsEqual(t, gotClear, []mksnodegroup.Taint{
		taint("spot", "", "PreferNoSchedule"),
	})

	// Clear with no additions emits empty non-nil slice.
	gotEmpty := applyTaints(current, nil, nil, true)
	if gotEmpty == nil {
		t.Fatal("expected non-nil empty slice for clear")
	}
	if len(gotEmpty) != 0 {
		t.Fatalf("expected empty, got %v", gotEmpty)
	}
}

func assertTaintsEqual(t *testing.T, got, want []mksnodegroup.Taint) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("applyTaints() len = %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for _, w := range want {
		found := false
		for _, g := range got {
			if g.Key == w.Key && g.Value == w.Value && g.Effect == w.Effect {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("applyTaints() missing %v in %v", w, got)
		}
	}
}

func TestParseLabelKeysAndTaintKeys(t *testing.T) {
	labelKeys, err := parseLabelKeys([]string{"a", "b"})
	if err != nil || !reflect.DeepEqual(labelKeys, []string{"a", "b"}) {
		t.Fatalf("parseLabelKeys = %v, %v", labelKeys, err)
	}
	if _, err := parseLabelKeys([]string{""}); err == nil {
		t.Fatal("parseLabelKeys should reject empty key")
	}

	taintKeys, err := parseTaintKeys([]string{"gpu", "legacy:NoExecute", "dedicated=gpu:NoSchedule"})
	if err != nil || !reflect.DeepEqual(taintKeys, []string{"gpu", "legacy", "dedicated"}) {
		t.Fatalf("parseTaintKeys = %v, %v", taintKeys, err)
	}
	if _, err := parseTaintKeys([]string{":NoSchedule"}); err == nil {
		t.Fatal("parseTaintKeys should reject empty key")
	}
}
