// Package parallel provides small concurrency helpers for read-only fan-out
// workloads, such as querying the same API resource in every region at once.
package parallel

import (
	"context"
	"sync"
)

// Result holds the outcome of one region fetch performed by Regions. Rows are
// the pre-rendered table rows for the region (without the region column); Value
// is the typed result used for json/yaml grouped output.
type Result struct {
	Region string
	Rows   [][]string
	Value  interface{}
	Err    error
}

// Regions runs fn for each region concurrently and returns one Result per
// region, in the same order as the input slice. It is meant for idempotent
// GET-style calls. A failure in one region does not affect the others: every
// region is still queried and its outcome is reported in its own Result, so
// callers can produce best-effort output and warn about individual failures.
//
// The returned slice is fully populated before Regions returns; callers read it
// after the (single) WaitGroup wait, so there are no concurrent reads.
func Regions(ctx context.Context, regions []string, fn func(ctx context.Context, region string) ([][]string, interface{}, error)) []Result {
	results := make([]Result, len(regions))

	var wg sync.WaitGroup
	for i, region := range regions {
		wg.Add(1)
		go func(i int, region string) {
			defer wg.Done()
			// Each goroutine writes to its own slice element, so there is no
			// data race between them.
			results[i].Region = region
			results[i].Rows, results[i].Value, results[i].Err = fn(ctx, region)
		}(i, region)
	}

	wg.Wait()
	return results
}
