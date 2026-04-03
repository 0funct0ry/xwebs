package repl

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/0funct0ry/xwebs/internal/ws"
)

// BenchmarkStats holds the results of a benchmark run.
type BenchmarkStats struct {
	Count      int
	Latencies  []time.Duration
	StartTime  time.Time
	EndTime    time.Time
	ErrorCount int
}

// RunBenchmark executes a sequential latency benchmark.
func (r *REPL) RunBenchmark(ctx context.Context, conn *ws.Connection, n int, message string) {
	if conn == nil || conn.IsClosed() {
		r.Errorf("Error: no active connection\n")
		return
	}

	r.Printf("▶ Benchmarking %d iterations...\n", n)
	
	stats := &BenchmarkStats{
		Count:     n,
		Latencies: make([]time.Duration, 0, n),
		StartTime: time.Now(),
	}

	// Subscribe to receive messages
	msgCh := conn.Subscribe()
	defer conn.Unsubscribe(msgCh)

	// Temporarily disable message printing if not verbose
	oldQuiet := r.Display.Quiet
	if !r.Display.Verbose {
		r.Display.Quiet = true
	}
	defer func() { r.Display.Quiet = oldQuiet }()

	for i := 0; i < n; i++ {
		select {
		case <-ctx.Done():
			r.Printf("\nBenchmark interrupted.\n")
			goto report
		default:
		}

		start := time.Now()
		err := conn.Write(&ws.Message{Type: ws.TextMessage, Data: []byte(message)})
		if err != nil {
			stats.ErrorCount++
			continue
		}

		// Wait for response
		select {
		case <-msgCh:
			stats.Latencies = append(stats.Latencies, time.Since(start))
		case <-time.After(5 * time.Second):
			stats.ErrorCount++
		case <-ctx.Done():
			goto report
		}

		if n > 10 && (i+1)%(n/10) == 0 {
			r.Printf("  Progress: %d%%\r", (i+1)*100/n)
		}
	}
	r.Printf("  Progress: 100%%\n")

report:
	stats.EndTime = time.Now()
	r.printBenchmarkReport(stats)
}

func (r *REPL) printBenchmarkReport(stats *BenchmarkStats) {
	duration := stats.EndTime.Sub(stats.StartTime)
	success := len(stats.Latencies)
	
	r.Printf("\nBenchmark Results:\n")
	r.Printf("  Total Time:     %v\n", duration.Round(time.Millisecond))
	r.Printf("  Messages Sent:  %d\n", stats.Count)
	r.Printf("  Successful:     %d\n", success)
	r.Printf("  Errors/Timeouts: %d\n", stats.ErrorCount)

	if success > 0 {
		sort.Slice(stats.Latencies, func(i, j int) bool {
			return stats.Latencies[i] < stats.Latencies[j]
		})

		var total time.Duration
		for _, l := range stats.Latencies {
			total += l
		}
		mean := total / time.Duration(success)

		r.Printf("\nLatency Stats:\n")
		r.Printf("  Min:    %v\n", stats.Latencies[0])
		r.Printf("  Max:    %v\n", stats.Latencies[success-1])
		r.Printf("  Mean:   %v\n", mean)
		r.Printf("  P50:    %v\n", stats.Latencies[int(float64(success)*0.5)])
		r.Printf("  P90:    %v\n", stats.Latencies[int(float64(success)*0.9)])
		r.Printf("  P99:    %v\n", stats.Latencies[int(float64(success)*0.99)])

		tput := float64(success) / duration.Seconds()
		r.Printf("\nThroughput: %.2f msgs/sec\n", tput)
	} else {
		r.Printf("\nNo successful responses received.\n")
	}
}

// RunFlood sends messages as fast as possible.
func (r *REPL) RunFlood(ctx context.Context, conn *ws.Connection, message string, rate float64) {
	if conn == nil || conn.IsClosed() {
		r.Errorf("Error: no active connection\n")
		return
	}

	if rate > 0 {
		r.Printf("▶ Flooding with %s at %.2f msgs/sec (Ctrl+C to stop)...\n", message, rate)
	} else {
		r.Printf("▶ Flooding with %s at max speed (Ctrl+C to stop)...\n", message)
	}

	start := time.Now()
	sent := 0
	errs := 0

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Subscription to track incoming messages for "receive throughput"
	msgCh := conn.Subscribe()
	defer conn.Unsubscribe(msgCh)
	recv := 0

	// Disable printing
	oldQuiet := r.Display.Quiet
	r.Display.Quiet = true
	defer func() { r.Display.Quiet = oldQuiet }()

	var sleepTime time.Duration
	if rate > 0 {
		sleepTime = time.Duration(float64(time.Second) / rate)
	}

	for {
		select {
		case <-ctx.Done():
			goto finish
		case <-ticker.C:
			dur := time.Since(start)
			r.Printf("  Sent: %d, Recv: %d, Rate: %.2f msgs/sec\r", sent, recv, float64(sent)/dur.Seconds())
		case <-msgCh:
			recv++
		default:
			err := conn.Write(&ws.Message{Type: ws.TextMessage, Data: []byte(message)})
			if err != nil {
				errs++
			} else {
				sent++
			}

			if sleepTime > 0 {
				time.Sleep(sleepTime)
			}
		}
	}

finish:
	duration := time.Since(start)
	r.Printf("\nFlood Finished:\n")
	r.Printf("  Duration:    %v\n", duration.Round(time.Millisecond))
	r.Printf("  Total Sent:  %d\n", sent)
	r.Printf("  Total Recv:  %d\n", recv)
	r.Printf("  Errors:      %d\n", errs)
	r.Printf("  Avg Rate:    %.2f msgs/sec\n", float64(sent)/duration.Seconds())
}

// RunWatch monitors connection stats real-time.
func (r *REPL) RunWatch(ctx context.Context, conn *ws.Connection) {
	if conn == nil || conn.IsClosed() {
		r.Errorf("Error: no active connection\n")
		return
	}

	r.Printf("▶ Watching connection %s (Ctrl+C to stop)...\n", conn.URL)
	
	msgCh := conn.Subscribe()
	defer conn.Unsubscribe(msgCh)

	start := time.Now()
	recv := 0
	bytesRecv := 0
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.Printf("\nWatch stopped.\n")
			return
		case msg, ok := <-msgCh:
			if !ok {
				r.Printf("\nConnection closed.\n")
				return
			}
			if msg.Metadata.Direction == "received" {
				recv++
				bytesRecv += len(msg.Data)
			}
		case <-ticker.C:
			dur := time.Since(start)
			r.Printf("  Uptime: %v, Recv: %d msgs, Rate: %.2f msgs/sec, Data: %s/s\r", 
				dur.Round(time.Second), recv, float64(recv)/dur.Seconds(), formatBytes(float64(bytesRecv)/dur.Seconds()))
		}
	}
}

func formatBytes(b float64) string {
	if b < 1024 {
		return fmt.Sprintf("%.2f B", b)
	}
	if b < 1024*1024 {
		return fmt.Sprintf("%.2f KB", b/1024)
	}
	return fmt.Sprintf("%.2f MB", b/(1024*1024))
}
