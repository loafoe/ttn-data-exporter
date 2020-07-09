package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
)

const (
	baseHost = "data.thethingsnetwork.org"
	getPath = "/api/v2/query"
)

type Measurement struct {
	Battery     int       `json:"battery"`
	DeviceID    string    `json:"device_id"`
	Event       string    `json:"event"`
	Light       int       `json:"light"`
	Raw         string    `json:"raw"`
	Temperature float64   `json:"temperature"`
	Time        time.Time `json:"time"`
}

func measurementGetter(project, token string) func()([]Measurement, error) {
	client := &http.Client{}

	return func() ([]Measurement, error) {
		measurements := make([]Measurement, 0)

		req, err := http.NewRequest("GET",
			fmt.Sprintf("https://%s.%s%s", project, baseHost, getPath), nil)
		if err != nil {
			return measurements, err
		}
		req.Header.Set("Authorization", "key "+token)
		req.Header.Set("Accept", "application/json")
		q := req.URL.Query()
		q.Add("last", "5m")
		req.URL.RawQuery = q.Encode()

		/*
		d, err := httputil.DumpRequest(req, false)
		if err == nil {
			fmt.Printf("Request: %s\n", string(d))
		}
		*/

		resp, err := client.Do(req)
		if err != nil {
			return measurements, err
		}
		defer resp.Body.Close()

		/*
		d, err = httputil.DumpResponse(resp, true)
		if err == nil {
			fmt.Printf("Response: %s\n", string(d))
		}
		*/
		if resp.StatusCode == http.StatusNoContent {
			return measurements, nil
		}

		body, err := ioutil.ReadAll(resp.Body)

		err = json.Unmarshal(body, &measurements)
		if err != nil {
			return measurements, err
		}
		return measurements, nil
	}
}

var (
	ambientTemp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ttn_node_temperature_celsius",
		Help: "Current ambient temperature.",
	})
	batteryLevel = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ttn_node_battery_level",
		Help: "Current battery level.",
	})
	lightLevel = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ttn_node_light_level",
		Help: "Current light level.",
	})
)

func init() {
	// Metrics have to be registered to be exposed:
	prometheus.MustRegister(ambientTemp)
	prometheus.MustRegister(batteryLevel)
	prometheus.MustRegister(lightLevel)
}


func main() {
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ttndata")

	project := viper.GetString("project")
	token := viper.GetString("token")
	if project == "" || token == "" {
		fmt.Printf("missing project and/or token\n")
		return
	}

	getter := measurementGetter(project, token)

	go func() {
		for {
			time.Sleep(60 * time.Second)

			measurements, err := getter()
			if err != nil {
				fmt.Printf("failed to get measurement: %v\n", err)
				continue
			}
			if len(measurements) > 0 {
				// Get last
				last := (measurements)[len(measurements)-1]
				fmt.Printf("refreshing measurement\n")
				batteryLevel.Set(float64(last.Battery))
				ambientTemp.Set(last.Temperature)
				lightLevel.Set(float64(last.Light))
				continue
			}
			// No measurement otherwise
			fmt.Printf("no recent measurement found. zeroing battery\n")
			batteryLevel.Set(0.0)
		}
	}()

	fmt.Printf("server started...\n")
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))

}