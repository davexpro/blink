package util

import (
	"log"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCurrentTimeCache(t *testing.T) {
	randomMs := rand.Intn(1000)
	time.Sleep(time.Millisecond * time.Duration(randomMs))

	periodBuffer := 5 * time.Millisecond

	realNow := time.Now()
	cacheNow := GetCurrentTimeCache()
	assert.True(t, realNow.Before(cacheNow.Add(refreshPeriod+periodBuffer)))
	diff := realNow.UnixNano() - cacheNow.UnixNano()
	// logs.Debug("[TestCurrentTimeCache] realNow:%d, cacheNow:%d, diff:%dms, period:%dms", realNow.UnixNano(), cacheNow.UnixNano(), diff/1e6, refreshPeriod.Milliseconds())

	if diff > refreshPeriod.Nanoseconds() {
		log.Printf("[TestCurrentTimeCache] realNow:%d, cacheNow:%d, diff:%dms, period:%dms\n", realNow.UnixNano(), cacheNow.UnixNano(), diff/1e6, refreshPeriod.Milliseconds())
	}

	time.Sleep(1 * time.Second)
}

func TestConcurrentCurrentTimeCache(t *testing.T) {
	randomMs := rand.Intn(1000)
	time.Sleep(time.Millisecond * time.Duration(randomMs))

	periodBuffer := 5 * time.Millisecond

	for i := 0; i < 1000; i++ {
		go func() {
			realNow := time.Now()
			cacheNow := GetCurrentTimeCache()
			assert.True(t, realNow.Before(cacheNow.Add(refreshPeriod+periodBuffer)))
			diff := realNow.UnixNano() - cacheNow.UnixNano()
			// logs.Debug("[TestConcurrentCurrentTimeCache] realNow:%d, cacheNow:%d, diff:%dms, period:%dms", realNow.UnixNano(), cacheNow.UnixNano(), diff/1e6, refreshPeriod.Milliseconds())

			if diff > refreshPeriod.Nanoseconds() {
				log.Printf("[TestConcurrentCurrentTimeCache] realNow:%d, cacheNow:%d, diff:%dms, period:%dms\n", realNow.UnixNano(), cacheNow.UnixNano(), diff/1e6, refreshPeriod.Milliseconds())
			}
		}()
	}

	time.Sleep(2 * time.Second)
}

func TestFreeObject(t *testing.T) {
	go func() {
		for i := 0; i < 1000; i++ {
			refreshCurrentTime(time.Now())
			runtime.GC()
		}
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			_ = GetCurrentTimeCache()
			runtime.GC()
		}
	}()
	time.Sleep(3 * time.Second)
}
