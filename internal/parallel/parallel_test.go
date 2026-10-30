package parallel

import (
	"context"
	"errors"
	"reflect"
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

	want := []Result{
		{Region: "a", Rows: [][]string{{"a"}}, Value: "a"},
		{Region: "b", Rows: [][]string{{"b"}}, Value: "b"},
		{Region: "c", Rows: [][]string{{"c"}}, Value: "c"},
	}
	if !reflect.DeepEqual(results, want) {
		t.Fatalf("unexpected results:\n got: %+v\nwant: %+v", results, want)
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

func TestRegionsCancelsSiblingsOnFirstError(t *testing.T) {
	// The shared ctx is cancelled when the first region fails; siblings waiting
	// on ctx.Done() must observe context.Canceled.
	fn := func(ctx context.Context, region string) ([][]string, interface{}, error) {
		if region == "a" {
			return nil, nil, errors.New("boom")
		}
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}

	results := Regions(context.Background(), []string{"a", "b", "c"}, fn)

	if got := results[0].Err; got == nil || got.Error() != "boom" {
		t.Fatalf("expected region a to report boom, got: %v", got)
	}
	// Siblings (b, c) may or may not have observed the cancellation before the
	// wait returns, but they must never report nil if they did read ctx.Err().
	// At minimum the dedup of errors in the command layer only inspects the
	// first error, so here we only assert the shape is consistent.
	for _, r := range results[1:] {
		if r.Err == nil {
			t.Fatalf("region %s: expected a cancellation error, got nil", r.Region)
		}
	}
}
