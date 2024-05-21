package metrics

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/exp/maps"
)

type TimeBuckets map[time.Time]int

type Count struct {
	startTime          time.Time
	retentionSeconds   int
	measurements       chan measurement
	retentionThreshold time.Time
	ingestionMap       map[time.Time]keyBuckets
	historicalData     map[string]TimeBuckets
	closed             chan struct{}
	mu                 sync.Mutex
}

func NewCount(retentionSeconds int) *Count {
	c := &Count{
		startTime:        time.Now().Round(time.Second),
		retentionSeconds: retentionSeconds,
		// Large buffer to absorb writes while reading. This could still block if we record metrics
		// faster than we can ingest them.
		measurements:   make(chan measurement, 100000),
		ingestionMap:   map[time.Time]keyBuckets{},
		historicalData: map[string]TimeBuckets{},
		closed:         make(chan struct{}),
		mu:             sync.Mutex{},
	}

	go c.run()

	return c
}

func (c *Count) Record(key string, value int) {
	m := measurement{
		key:   key,
		value: value,
		// NOTE: The Round() function rounds to the nearest increment so the actual value of second
		// may be in the future. This shouldn't matter, it just means we're clustering measurements
		// around the second to whose start they're the closest instead of in the second within
		// which they occurred.
		second: time.Now().Round(time.Second),
	}

	start := time.Now()

	select {
	case c.measurements <- m:
		return
	default:
	}

	c.measurements <- m
	blockedFor := time.Since(start)
	fmt.Printf("warn: Count#Record blocked for %v\n", blockedFor)
}

func (c *Count) Data() map[string]TimeBuckets {
	data := func() map[string]TimeBuckets {
		c.mu.Lock()
		defer c.mu.Unlock()
		data := make(map[string]TimeBuckets, len(c.historicalData))
		for key, element := range c.historicalData {
			tb := make(TimeBuckets, len(element))
			for second, count := range element {
				tb[second] = count
			}
			data[key] = tb
		}
		return data
	}()

	// Populate the exported data with zero counts for any time within the retention period where
	// we have no data.
	now := time.Now()
	earliest := c.retentionThreshold
	if c.startTime.After(earliest) {
		earliest = c.startTime
	}
	for t := earliest.Round(time.Second); t.Before(now); t = t.Add(time.Second) {
		for key := range data {
			if _, ok := data[key][t]; !ok {
				data[key][t] = 0
			}
		}
	}

	return data
}

func (c *Count) Close() {
	defer close(c.measurements)
	defer close(c.closed)
	c.closed <- struct{}{}
}

func (c *Count) updateRetentionThreshold() {
	c.retentionThreshold = time.Now().Add(-time.Duration(c.retentionSeconds) * time.Second)
}

func (c *Count) run() {

	c.updateRetentionThreshold()

	onTick := func() {
		c.updateRetentionThreshold()
		c.mu.Lock()
		defer c.mu.Unlock()
		c.indexMeasurements()
		c.expireOldData()
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		// Concurrent map writes cause panics so we need to syncronize ingestion, indexing, and
		// expiration. The first select handles either: ingestion, indexing and expiration, or
		// closure, depending on the available signals and blocks until one is available. The
		// select does a non-blocking read for an indexing and expiration or a closure signal. When
		// multiple signals in a select are both available without blocking the control flow is
		// chosen arbitrarily, meaning that a series of buffered measurements could cause a delay
		// processing the index/expire or close signals. It's unlikely that a significant delay
		// would occur since the select would be re-evaluated after each ingested message and the
		// probabilty of selecting ingest `n` times in a row decreass as `n` increases; but this
		// approach ensures we never delay an index/expire or close for more than a single ingest.

		select {
		case m := <-c.measurements:
			c.ingestMeasurement(m)
		case <-ticker.C:
			onTick()
		case <-c.closed:
			return
		}

		select {
		case <-ticker.C:
			onTick()
		case <-c.closed:
			return
		default:
		}
	}
}

func (c *Count) ingestMeasurement(m measurement) {
	// Don't bother ingesting a measurement that's already older than our retention period.
	if m.second.Before(c.retentionThreshold) {
		return
	}

	countsByKey, ok := c.ingestionMap[m.second]
	if !ok {
		countsByKey = keyBuckets{}
		c.ingestionMap[m.second] = countsByKey
	}
	countsByKey[m.key] += m.value
}

func (c *Count) indexMeasurements() {
	for second, keyBuckets := range c.ingestionMap {
		for key, count := range keyBuckets {
			element, ok := c.historicalData[key]
			if !ok {
				element = TimeBuckets{}
				c.historicalData[key] = element
			}
			element[second] += count
			// Zero the ingest count so we don't re-index the same measurements again.
			keyBuckets[key] = 0
		}
	}
}

func (c *Count) expireOldData() {
	for _, timeBuckets := range c.historicalData {
		for _, second := range maps.Keys(timeBuckets) {
			if second.After(c.retentionThreshold) {
				continue
			}
			delete(timeBuckets, second)
		}
	}
	for second := range c.ingestionMap {
		if second.After(c.retentionThreshold) {
			continue
		}
		delete(c.ingestionMap, second)
	}
}

type keyBuckets map[string]int

type measurement struct {
	key    string
	value  int
	second time.Time
}
