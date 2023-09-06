package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Response struct {
	Kitty struct {
		ID                    string `json:"id"`
		OwnerID               string `json:"ownerId"`
		OwnerFirstName        string `json:"ownerFirstName"`
		OwnerLastName         string `json:"ownerLastName"`
		ContributionsCounter  int    `json:"contributionsCounter"`
		TotalCollectedAmount  int    `json:"totalCollectedAmount"`
	} `json:"kitty"`
	Available  int    `json:"available"`
}

var (
	contributionsCounterGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lyf_contributions_counter",
			Help: "Number of contributions on the kitty",
		},
		[]string{"OwnerFirstName", "OwnerLastName", "OwnerID", "ID"},
	)
	totalCollectedAmountGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lyf_total_collected_amount",
			Help: "Total collected amount (cents)",
		},
		[]string{"OwnerFirstName", "OwnerLastName", "OwnerID", "ID"},
	)
)

func main() {
	prometheus.MustRegister(contributionsCounterGauge)
	prometheus.MustRegister(totalCollectedAmountGauge)

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		fmt.Println("Starting HTTP server on :8080")
		http.ListenAndServe(":8080", nil)
	}()

	url := "https://api.lyf.eu/public/api/kitties/" + os.Getenv("LYF_KITTY_UUID")

	for {
		response, err := http.Get(url)
		if err != nil {
			fmt.Printf("Failed to make HTTP request: %v\n", err)
			time.Sleep(time.Second * 5)
			continue
		}
		defer response.Body.Close()

		var responseData Response
		if err := json.NewDecoder(response.Body).Decode(&responseData); err != nil {
			fmt.Printf("Failed to parse JSON response: %v\n", err)
			time.Sleep(time.Second * 5)
			continue
		}

		contributionsCounterGauge.WithLabelValues(responseData.Kitty.OwnerFirstName, responseData.Kitty.OwnerLastName, responseData.Kitty.OwnerID, responseData.Kitty.ID).Set(float64(responseData.Kitty.ContributionsCounter))
		totalCollectedAmountGauge.WithLabelValues(responseData.Kitty.OwnerFirstName, responseData.Kitty.OwnerLastName, responseData.Kitty.OwnerID, responseData.Kitty.ID).Set(float64(responseData.Kitty.TotalCollectedAmount))

		time.Sleep(time.Minute)
	}
}