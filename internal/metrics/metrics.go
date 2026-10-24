// Package metrics 轻量 Prometheus 文本格式指标（无第三方依赖）。
// 指标：请求计数（按 method/route/code）、请求耗时 summary、进程运行时信息。
package metrics

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

var buckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

var (
	mu       sync.Mutex
	counts   = map[string]float64{}             // key: method|route|code
	sums     = map[string]float64{}             // key: method|route
	cnts     = map[string]float64{}             // key: method|route
	bucketed = map[string]map[float64]float64{} // key: method|route → le → 累计
	started  = time.Now()
)

// Observe 记录一次请求。
func Observe(method, route string, code int, seconds float64) {
	ckey := fmt.Sprintf("%s|%s|code:%d", method, route, code)
	hkey := method + "|" + route

	mu.Lock()
	defer mu.Unlock()
	counts[ckey]++
	sums[hkey] += seconds
	cnts[hkey]++
	b := bucketed[hkey]
	if b == nil {
		b = make(map[float64]float64, len(buckets))
		bucketed[hkey] = b
	}
	for _, up := range buckets {
		if seconds <= up {
			b[up]++
		}
	}
}

func sortedKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func fmtLe(v float64) string {
	return fmt.Sprintf("%g", v)
}

// Render 输出 Prometheus 文本格式（version 0.0.4）。
func Render() string {
	var b strings.Builder

	b.WriteString("# HELP goask_uptime_seconds 进程运行时长\n")
	b.WriteString("# TYPE goask_uptime_seconds gauge\n")
	b.WriteString(fmt.Sprintf("goask_uptime_seconds %d\n", int(time.Since(started).Seconds())))

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	b.WriteString("# HELP goask_memory_alloc_bytes 堆内存占用\n")
	b.WriteString("# TYPE goask_memory_alloc_bytes gauge\n")
	b.WriteString(fmt.Sprintf("goask_memory_alloc_bytes %d\n", m.Alloc))
	b.WriteString("# HELP goask_goroutines goroutine 数量\n")
	b.WriteString("# TYPE goask_goroutines gauge\n")
	b.WriteString(fmt.Sprintf("goask_goroutines %d\n", runtime.NumGoroutine()))

	b.WriteString("# HELP goask_requests_total 请求总数\n")
	b.WriteString("# TYPE goask_requests_total counter\n")
	for _, k := range sortedKeys(counts) {
		parts := strings.SplitN(k, "|", 3)
		b.WriteString(fmt.Sprintf("goask_requests_total{method=%q,route=%q,code=%q} %g\n",
			parts[0], parts[1], strings.TrimPrefix(parts[2], "code:"), counts[k]))
	}

	b.WriteString("# HELP goask_request_duration_seconds 请求耗时\n")
	b.WriteString("# TYPE goask_request_duration_seconds summary\n")
	for _, k := range sortedKeys(sums) {
		parts := strings.SplitN(k, "|", 2)
		for _, up := range buckets {
			b.WriteString(fmt.Sprintf("goask_request_duration_seconds_bucket{method=%q,route=%q,le=%q} %g\n",
				parts[0], parts[1], fmtLe(up), bucketed[k][up]))
		}
		b.WriteString(fmt.Sprintf("goask_request_duration_seconds_sum{method=%q,route=%q} %g\n", parts[0], parts[1], sums[k]))
		b.WriteString(fmt.Sprintf("goask_request_duration_seconds_count{method=%q,route=%q} %g\n", parts[0], parts[1], cnts[k]))
	}
	return b.String()
}
