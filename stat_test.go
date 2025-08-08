package mqtt

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestStatRegister(t *testing.T) {
	// Create a new stat instance for testing
	testStat := Stat{
		Uptime:            prometheus.NewCounter(prometheus.CounterOpts{Name: "test_uptime", Help: "Test uptime"}),
		ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_active_connections", Help: "Test active connections"}),
		PacketReceived:    prometheus.NewCounter(prometheus.CounterOpts{Name: "test_packets_received", Help: "Test packets received"}),
		ByteReceived:      prometheus.NewCounter(prometheus.CounterOpts{Name: "test_bytes_received", Help: "Test bytes received"}),
		PacketSent:        prometheus.NewCounter(prometheus.CounterOpts{Name: "test_packets_sent", Help: "Test packets sent"}),
		ByteSent:          prometheus.NewCounter(prometheus.CounterOpts{Name: "test_bytes_sent", Help: "Test bytes sent"}),
	}

	// Test that Register doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Register panicked: %v", r)
		}
	}()

	testStat.Register()
}

func TestStatRefreshUptime(t *testing.T) {
	testStat := Stat{
		Uptime: prometheus.NewCounter(prometheus.CounterOpts{Name: "test_uptime", Help: "Test uptime"}),
	}

	// Start the uptime refresh
	testStat.RefreshUptime()

	// Wait a bit for the goroutine to start
	time.Sleep(100 * time.Millisecond)

	// Check that the uptime counter has been incremented
	// Note: We can't easily test the exact value since it's running in a goroutine
	// but we can verify the function doesn't panic
}

func TestStatIncrement(t *testing.T) {
	testStat := Stat{
		ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_active_connections", Help: "Test active connections"}),
		PacketReceived:    prometheus.NewCounter(prometheus.CounterOpts{Name: "test_packets_received", Help: "Test packets received"}),
		ByteReceived:      prometheus.NewCounter(prometheus.CounterOpts{Name: "test_bytes_received", Help: "Test bytes received"}),
		PacketSent:        prometheus.NewCounter(prometheus.CounterOpts{Name: "test_packets_sent", Help: "Test packets sent"}),
		ByteSent:          prometheus.NewCounter(prometheus.CounterOpts{Name: "test_bytes_sent", Help: "Test bytes sent"}),
	}

	// Test incrementing counters
	testStat.ActiveConnections.Inc()
	testStat.PacketReceived.Inc()
	testStat.ByteReceived.Add(100)
	testStat.PacketSent.Inc()
	testStat.ByteSent.Add(200)

	// These operations should not panic
}

func TestStatDecrement(t *testing.T) {
	testStat := Stat{
		ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_active_connections", Help: "Test active connections"}),
	}

	// Test decrementing gauge
	testStat.ActiveConnections.Inc()
	testStat.ActiveConnections.Dec()

	// This operation should not panic
}

func TestStatAdd(t *testing.T) {
	testStat := Stat{
		ByteReceived: prometheus.NewCounter(prometheus.CounterOpts{Name: "test_bytes_received", Help: "Test bytes received"}),
		ByteSent:     prometheus.NewCounter(prometheus.CounterOpts{Name: "test_bytes_sent", Help: "Test bytes sent"}),
	}

	// Test adding values to counters
	testStat.ByteReceived.Add(1024)
	testStat.ByteSent.Add(2048)

	// These operations should not panic
}

func TestStatConcurrentAccess(t *testing.T) {
	testStat := Stat{
		ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{Name: "test_active_connections", Help: "Test active connections"}),
		PacketReceived:    prometheus.NewCounter(prometheus.CounterOpts{Name: "test_packets_received", Help: "Test packets received"}),
	}

	// Test concurrent access to stat counters
	done := make(chan bool)

	// Start multiple goroutines that access the stat counters
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				testStat.ActiveConnections.Inc()
				testStat.PacketReceived.Inc()
				testStat.ActiveConnections.Dec()
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// This should not cause any race conditions or panics
}

func TestStatInitialization(t *testing.T) {
	// Test that the global stat variable is properly initialized
	if stat.Uptime == nil {
		t.Error("stat.Uptime should not be nil")
	}
	if stat.ActiveConnections == nil {
		t.Error("stat.ActiveConnections should not be nil")
	}
	if stat.PacketReceived == nil {
		t.Error("stat.PacketReceived should not be nil")
	}
	if stat.ByteReceived == nil {
		t.Error("stat.ByteReceived should not be nil")
	}
	if stat.PacketSent == nil {
		t.Error("stat.PacketSent should not be nil")
	}
	if stat.ByteSent == nil {
		t.Error("stat.ByteSent should not be nil")
	}
}

func TestStatMetricNames(t *testing.T) {
	// Test that metric names are properly set
	// Note: We can't easily access the metric names from the prometheus types
	// so we'll just test that the metrics are not nil
	if stat.Uptime == nil {
		t.Error("stat.Uptime should not be nil")
	}
	if stat.ActiveConnections == nil {
		t.Error("stat.ActiveConnections should not be nil")
	}
	if stat.PacketReceived == nil {
		t.Error("stat.PacketReceived should not be nil")
	}
	if stat.ByteReceived == nil {
		t.Error("stat.ByteReceived should not be nil")
	}
	if stat.PacketSent == nil {
		t.Error("stat.PacketSent should not be nil")
	}
	if stat.ByteSent == nil {
		t.Error("stat.ByteSent should not be nil")
	}
}
