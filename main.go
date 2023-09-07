package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"time"

	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Kitty contains contrib data for project (?)
type Kitty struct {
	ID                   string `json:"id"`
	OwnerID              string `json:"ownerId"`
	OwnerFirstName       string `json:"ownerFirstName"`
	OwnerLastName        string `json:"ownerLastName"`
	ContributionsCounter int    `json:"contributionsCounter"`
	TotalCollectedAmount int    `json:"totalCollectedAmount"`
}

// Response contains data from Lyf API
type Response struct {
	Kitty     `json:"kitty"`
	Available int `json:"available"`
}

// config contains data grabber config
type config struct {
	url   string
	delay time.Duration
	port  int
}

// collectors contains metrics
type collectors struct {
	contributionsCounterGauge *prometheus.GaugeVec
	totalCollectedAmountGauge *prometheus.GaugeVec
}

// parseConfig builds url and delay according to env variables
func parseConfig() (*config, error) {
	var err error

	url := "https://api.lyf.eu/public/api/kitties/" + os.Getenv("LYF_KITTY_UUID")
	delay := 60 * time.Second
	port := 8080

	if os.Getenv("LYF_DELAY") != "" {
		delay, err = time.ParseDuration(os.Getenv("LYF_DELAY"))
		if err != nil {
			return nil, fmt.Errorf("error parsing LYF_DELAY %q: %v", os.Getenv("LYF_DELAY"), err)
		}
	}

	if os.Getenv("LYF_PORT") != "" {
		port, err = strconv.Atoi(os.Getenv("LYF_PORT"))
		if err != nil {
			return nil, fmt.Errorf("error parsing LYF_PORT %q: %v", os.Getenv("LYF_PORT"), err)
		}
	}

	return &config{
		url:   url,
		delay: delay,
		port:  port,
	}, nil
}

// registerPrometheusMetrics creates & registers metrics
func registerPrometheusMetrics() (collectors, error) {
	var c collectors

	c.contributionsCounterGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lyf_contributions_counter",
			Help: "Number of contributions on the kitty",
		},
		[]string{"OwnerFirstName", "OwnerLastName", "OwnerID", "ID"},
	)

	c.totalCollectedAmountGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lyf_total_collected_amount",
			Help: "Total collected amount (cents)",
		},
		[]string{"OwnerFirstName", "OwnerLastName", "OwnerID", "ID"},
	)

	if err := prometheus.Register(c.contributionsCounterGauge); err != nil {
		return c, fmt.Errorf("contributionsCounterGauge not registered: %v", err)
	}

	if err := prometheus.Register(c.totalCollectedAmountGauge); err != nil {
		return c, fmt.Errorf("totalCollectedAmountGauge not registered: %v", err)
	}

	return c, nil
}

func getData(ctx context.Context, url string) (*Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP response %d: %s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	var responseData Response
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	return &responseData, nil
}

func dataGrabber(ctx context.Context, url string, delay time.Duration, c collectors) {
	// we want to start immediately at first run
	next := time.Duration(0)

	for {
		time.Sleep(next * time.Second)
		next = delay

		responseData, err := getData(ctx, url)
		if err != nil {
			slog.Error("failed to get data: %v", err)
		}

		c.contributionsCounterGauge.WithLabelValues(responseData.Kitty.OwnerFirstName, responseData.Kitty.OwnerLastName, responseData.Kitty.OwnerID, responseData.Kitty.ID).Set(float64(responseData.Kitty.ContributionsCounter))
		c.totalCollectedAmountGauge.WithLabelValues(responseData.Kitty.OwnerFirstName, responseData.Kitty.OwnerLastName, responseData.Kitty.OwnerID, responseData.Kitty.ID).Set(float64(responseData.Kitty.TotalCollectedAmount))
	}
}

func main() {
	config, err := parseConfig()
	if err != nil {
		slog.Error("unable to parse environment variables: %v", err)
		os.Exit(1)
	}

	metrics, err := registerPrometheusMetrics()
	if err != nil {
		slog.Error("unable to register metrics: %v", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.port),
		Handler: mux,
	}

	g := run.Group{}
	ctx, cancel := context.WithCancel(context.Background())

	// probably overkill since nobody will ever return
	// however, this is a good starting point if other services are added
	g.Add(func() error {
		slog.Info("server listening", "port", config.port)
		return server.ListenAndServe()
	}, func(error) {
		server.Shutdown(ctx)
	})

	g.Add(func() error {
		slog.Info("starting data grabber", "delay", config.delay, "url", config.url)
		dataGrabber(ctx, config.url, config.delay, metrics)
		return nil
	}, func(error) {
		cancel()
	})

	g.Run()
}
