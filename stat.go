package mqtt

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/golang-io/requests"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Stat struct {
	Uptime            prometheus.Counter
	ActiveConnections prometheus.Gauge
	PacketReceived    prometheus.Counter
	ByteReceived      prometheus.Counter
	PacketSent        prometheus.Counter
	ByteSent          prometheus.Counter
}

var (
	stat = Stat{
		Uptime:            prometheus.NewCounter(prometheus.CounterOpts{Name: "mqtt_uptime_seconds", Help: "The uptime in seconds"}),
		ActiveConnections: prometheus.NewGauge(prometheus.GaugeOpts{Name: "mqtt_active_client_count", Help: "The active number of MQTT clients"}),
		PacketReceived:    prometheus.NewCounter(prometheus.CounterOpts{Name: "mqtt_received_packets", Help: "The total number of received MQTT packets"}),
		ByteReceived:      prometheus.NewCounter(prometheus.CounterOpts{Name: "mqtt_received_bytes", Help: "The total number of received MQTT bytes"}),
		PacketSent:        prometheus.NewCounter(prometheus.CounterOpts{Name: "mqtt_send_packets", Help: "The total number of send MQTT packets"}),
		ByteSent:          prometheus.NewCounter(prometheus.CounterOpts{Name: "mqtt_send_bytes", Help: "The total number of send MQTT bytes"}),
	}
)

func IN(x string, m ...string) bool {
	for i := range m {
		if strings.HasSuffix(x, m[i]) {
			return true
		}
	}
	return false
}

func ServerLog(ctx context.Context, stat *requests.Stat) {
	if stat.Request.URL == "/" || IN(stat.Request.URL, ".html", ".js", ".css") {
		log.Println(stat.Print())
		return
	}
	log.Printf("%s # body=%s, resp=%s", stat.Print(), stat.RequestBody(), stat.ResponseBody())
}

func Httpd() error {
	stat.Register()
	stat.RefreshUptime()
	mux := requests.NewServeMux(requests.URL(CONFIG.HTTP.URL), requests.Logf(ServerLog))
	mux.GET("/_metrics", promhttp.Handler())
	mux.GET("/_paths", func(w http.ResponseWriter, r *http.Request) {
		mux.Print(w)
	})
	mux.GET("/", http.FileServer(http.Dir("web")))

	mux.Pprof()
	s := requests.NewServer(context.Background(), mux, requests.OnStart(func(s *http.Server) {
		log.Printf("http serve: %s", s.Addr)
	}))
	return s.ListenAndServe()
}

func (s *Stat) RefreshUptime() {
	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for range tick.C {
			s.Uptime.Inc()
		}
	}()
}

func (s *Stat) Register() {
	prometheus.MustRegister(stat.Uptime)
	prometheus.MustRegister(stat.ActiveConnections)
	prometheus.MustRegister(stat.PacketReceived)
	prometheus.MustRegister(stat.ByteReceived)
	prometheus.MustRegister(stat.PacketSent)
	prometheus.MustRegister(stat.ByteSent)
}
