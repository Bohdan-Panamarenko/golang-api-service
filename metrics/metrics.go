package metrics

import (
	"api-service/api"
	"api-service/logging"
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type CustomMetrics struct {
	RegisteredUsers prometheus.Gauge
	CakesGiven      prometheus.Counter
	WsConnections   prometheus.Gauge
	ExecutionTime   *prometheus.HistogramVec
}

func NewCustomMetrics() *CustomMetrics {
	cm := &CustomMetrics{
		RegisteredUsers: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "api_service_registered_users",
			Help: "The total number of registered users",
		}),
		CakesGiven: promauto.NewCounter(prometheus.CounterOpts{
			Name: "api_service_cakes_given",
			Help: "The total number of all given cakes since start",
		}),
		WsConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "api_service_websocket_connections",
			Help: "The current number of websocket connections",
		}),
	}

	// prometheus.Register(cm.RegisteredUsers)
	// prometheus.Register(cm.CakesGiven)
	// prometheus.Register()
	return cm
}

func (cm *CustomMetrics) BuildExecutionTime() {
	cm.ExecutionTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "api_service_requests_executions_time",
			Help: "The time of execution of all requests",
		},
		[]string{"path"},
	)

	// prometheus.Register(cm.ExecutionTime)
}

type Response struct {
	Path         string
	Status       int
	Duration     time.Duration
	ResponseBody string
}

type ExecuteMetrics func(resp Response, cm *CustomMetrics)

func AddTime(resp Response, cm *CustomMetrics) {
	cm.ExecutionTime.WithLabelValues(strings.Replace(resp.Path, "/", "_", -1)).Observe(float64(resp.Duration.Microseconds()) / 1000)
}

func IncCakes(resp Response, cm *CustomMetrics) {
	AddTime(resp, cm)
	if resp.Status == 0 {
		cm.CakesGiven.Inc()
	}
}

func IncUsers(resp Response, cm *CustomMetrics) {
	AddTime(resp, cm)
	if resp.Status == http.StatusCreated {
		cm.RegisteredUsers.Inc()
	}
}

func Serve() {
	http.Handle("/metrics", promhttp.Handler())

	log.Println("Metrics server started")
	http.ListenAndServe(":2112", nil)
}

func LogRequestMetrics(cm *CustomMetrics, f ExecuteMetrics, h http.HandlerFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		writer := &logging.LogWriter{
			ResponseWriter: rw,
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("Could not read request body", err)
			api.HandleError(errors.New("Could not read requst"), rw)

			return
		}
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		started := time.Now()
		h(writer, r)
		done := time.Since(started)

		log.Printf(
			"PATH: %s -> %d. Finished in %v.\n\tParams: %s\n\tResponse: %s",
			r.URL.Path,
			writer.StatusCode,
			done,
			string(body),
			writer.Response.String(),
		)

		f(
			Response{
				Path:         r.URL.Path,
				Status:       writer.StatusCode,
				Duration:     done,
				ResponseBody: writer.Response.String(),
			},
			cm,
		)
	}
}

// func WsWrapper(cm *CustomMetrics, h http.HandlerFunc) http.HandlerFunc {
// 	cm.WsConnections.Inc()

// 	return func(w http.ResponseWriter, r *http.Request) {
// 		h(w, r)
// 		cm.WsConnections.Dec()
// 	}
// }
