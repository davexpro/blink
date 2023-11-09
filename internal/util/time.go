package util

import (
	"sync/atomic"
	"time"
	"unsafe"
)

var (
	currentTimeCache *time.Time
	refreshPeriod    = time.Millisecond * 10
	ticker           = time.NewTicker(refreshPeriod)
)

// SetRefreshPeriod 设置时间缓存的更新周期
func SetRefreshPeriod(t time.Duration) {
	atomic.StoreInt64((*int64)(&refreshPeriod), int64(t))
}

// GetCurrentTimeCache 获取当前时间的缓存
func GetCurrentTimeCache() time.Time {
	return *(*time.Time)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&currentTimeCache))))
}

func refreshTask() {
	localC := atomic.LoadInt64((*int64)(&refreshPeriod))
	for {
		cur := <-ticker.C
		refreshCurrentTime(cur)
		clock := atomic.LoadInt64((*int64)(&refreshPeriod))
		if clock != localC {
			ticker.Stop()
			ticker = time.NewTicker(time.Duration(clock))
		}
	}
}

func refreshCurrentTime(cur time.Time) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&currentTimeCache)), unsafe.Pointer(&cur))
}

func init() {
	refreshCurrentTime(time.Now())
	go refreshTask()
}
