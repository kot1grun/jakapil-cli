package parallel

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRegionsPreservesOrderAndCollects(t *testing.T) {
	regions := []string{"a", "b", "c"}
	results := Regions(context.Background(), regions, func(_ context.Context, region string) ([][]string, interface{}, error) {
		return [][]string{{region}}, region, nil
	})

	want := []struct {
		region string
		rows   [][]string
		value  string
	}{
		{region: "a", rows: [][]string{{"a"}}, value: "a"},
		{region: "b", rows: [][]string{{"b"}}, value: "b"},
		{region: "c", rows: [][]string{{"c"}}, value: "c"},
	}

	if len(results) != len(want) {
		t.Fatalf("expected %d results, got %d", len(want), len(results))
	}
	for i, r := range results {
		w := want[i]
		if r.Region != w.region {
			t.Errorf("results[%d].Region = %q, want %q", i, r.Region, w.region)
		}
		if r.Err != nil {
			t.Errorf("results[%d].Err = %v, want nil", i, r.Err)
		}
		if len(r.Rows) != len(w.rows) {
			t.Errorf("results[%d].Rows length = %d, want %d", i, len(r.Rows), len(w.rows))
			continue
		}
		for j := range w.rows {
			if len(r.Rows[j]) != len(w.rows[j]) {
				t.Errorf("results[%d].Rows[%d] length = %d, want %d", i, j, len(r.Rows[j]), len(w.rows[j]))
				continue
			}
			for k := range w.rows[j] {
				if r.Rows[j][k] != w.rows[j][k] {
					t.Errorf("results[%d].Rows[%d][%d] = %q, want %q", i, j, k, r.Rows[j][k], w.rows[j][k])
				}
			}
		}
		if got, ok := r.Value.(string); !ok || got != w.value {
			t.Errorf("results[%d].Value = %#v, want %q", i, r.Value, w.value)
		}
	}
}

func TestRegionsRunsConcurrently(t *testing.T) {
	block := make(chan struct{})
	var started atomic.Int32
	var wg sync.WaitGroup

	regions := []string{"a", "b", "c"}
	wg.Add(len(regions))
	go func() {
		Regions(context.Background(), regions, func(_ context.Context, _ string) ([][]string, interface{}, error) {
			started.Add(1)
			defer wg.Done()
			<-block // hold until released
			return nil, nil, nil
		})
	}()

	// Give goroutines a moment to spin up; if Regions were sequential all of
	// them could not be blocked on a single channel simultaneously.
	deadline := time.Now().Add(2 * time.Second)
	for started.Load() < int32(len(regions)) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if started.Load() != int32(len(regions)) {
		t.Fatalf("expected all %d fns to start concurrently, got %d", len(regions), started.Load())
	}
	close(block)
	wg.Wait()
}

func TestRegionsBestEffortStillQueriesSucceededRegions(t *testing.T) {
	// A failure in one region must not prevent the others from being queried:
	// best-effort fans out to every region and reports each outcome in its own
	// Result.
	fn := func(_ context.Context, region string) ([][]string, interface{}, error) {
		if region == "a" {
			return nil, nil, errors.New("boom")
		}
		return [][]string{{region + "-row"}}, region, nil
	}

	results := Regions(context.Background(), []string{"a", "b", "c"}, fn)

	if got := results[0].Err; got == nil || got.Error() != "boom" {
		t.Fatalf("expected region a to report boom, got: %v", got)
	}
	if len(results[0].Rows) != 0 {
		t.Fatalf("region a: expected no rows on error, got %v", results[0].Rows)
	}

	for _, idx := range []int{1, 2} {
		if results[idx].Err != nil {
			t.Fatalf("region %s: unexpected error %v", results[idx].Region, results[idx].Err)
		}
		if len(results[idx].Rows) != 1 || results[idx].Rows[0][0] != results[idx].Region+"-row" {
			t.Fatalf("region %s: unexpected rows %v", results[idx].Region, results[idx].Rows)
		}
		if v, ok := results[idx].Value.(string); !ok || v != results[idx].Region {
			t.Fatalf("region %s: unexpected value %#v", results[idx].Region, results[idx].Value)
		}
	}
}
