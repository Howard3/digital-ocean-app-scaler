package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/digitalocean/godo"
	"github.com/joho/godotenv"
	promAPI "github.com/prometheus/client_golang/api"
	promV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type SimpleWebServer struct {
	lastInstanceSize int       `json:"last_instance_size"`
	lastCheck        time.Time `json:"last_check"`
}

func (sws *SimpleWebServer) start() {
	log.Printf("Starting web server\n")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		json, _ := json.Marshal(sws)

		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
	})

	go func() {
		bindPort := "8080"
		if os.Getenv("BIND_PORT") != "" {
			bindPort = os.Getenv("BIND_PORT")
		}

		if err := http.ListenAndServe(":"+bindPort, nil); err != nil {
			panic(fmt.Sprintf("Error starting simple webserver: %v", err))
		}
	}()
}

type Config struct {
	PrometheusHost   string
	ThresholdUp      float64
	MaxSize          int
	ThresholdDown    float64
	DOAPIToken       string
	DOAppID          string
	PrometheusMetric string
	prometheusV1API  *promV1.API
	godoClient       *godo.Client
	simpleWebServer  *SimpleWebServer
}

func LoadConfig() (Config, error) {
	_ = godotenv.Load()

	var config Config
	var ok bool

	if config.PrometheusHost, ok = os.LookupEnv("PROMETHEUS_HOST"); !ok {
		return config, errors.New("PROMETHEUS_HOST is required")
	}

	if thresholdUp, ok := os.LookupEnv("THRESHOLD_UP"); !ok {
		return config, errors.New("THRESHOLD_UP is required")
	} else {
		var err error
		config.ThresholdUp, err = strconv.ParseFloat(thresholdUp, 64)
		if err != nil {
			return config, err
		}
	}

	if maxSize, ok := os.LookupEnv("MAX_SIZE"); !ok {
		return config, errors.New("MAX_SIZE is required")
	} else {
		var err error
		config.MaxSize, err = strconv.Atoi(maxSize)
		if err != nil {
			return config, err
		}
	}

	if thresholdDown, ok := os.LookupEnv("THRESHOLD_DOWN"); !ok {
		return config, errors.New("THRESHOLD_DOWN is required")
	} else {
		var err error
		config.ThresholdDown, err = strconv.ParseFloat(thresholdDown, 64)
		if err != nil {
			return config, err
		}
	}

	if config.DOAPIToken, ok = os.LookupEnv("DO_API_TOKEN"); !ok {
		return config, errors.New("DO_API_TOKEN is required")
	}

	if config.DOAppID, ok = os.LookupEnv("DO_APP_ID"); !ok {
		return config, errors.New("DO_APP_ID is required")
	}

	if config.PrometheusMetric, ok = os.LookupEnv("PROMETHEUS_METRIC"); !ok {
		return config, errors.New("PROMETHEUS_METRIC is required")
	}

	// todo: make optional
	SimpleWebServer := SimpleWebServer{}
	SimpleWebServer.start()

	config.simpleWebServer = &SimpleWebServer

	return config, nil
}

func main() {
	config, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	for {
		currentValue := config.getMetric()

		log.Printf("Current value: %f\n", currentValue)

		if currentValue > config.ThresholdUp {
			config.scaleUp()
		} else if currentValue < config.ThresholdDown {
			config.scaleDown()
		}

		log.Printf("Sleeping for 1 minute\n")

		time.Sleep(1 * time.Minute)
	}
}

func (c *Config) getPrometheusAPIClient() *promV1.API {
	if c.prometheusV1API != nil {
		return c.prometheusV1API
	}

	cli, err := promAPI.NewClient(promAPI.Config{Address: c.PrometheusHost})
	if err != nil {
		log.Fatal(err)
	}

	v1api := promV1.NewAPI(cli)
	c.prometheusV1API = &v1api

	return &v1api
}

// Get the current value of the metric from Prometheus
func (c *Config) getMetric() float64 {
	cli := *c.getPrometheusAPIClient()
	metric := fmt.Sprintf(`scalar(%s)`, c.PrometheusMetric)
	res, warnings, err := cli.Query(context.Background(), metric, time.Now())
	if err != nil {
		log.Fatal(err)
	}

	if len(warnings) > 0 {
		log.Fatal(warnings)
	}

	if res.Type() != model.ValScalar {
		log.Fatal("Result is not a scalar value")
	}

	// read the scalar value
	scalar, ok := res.(*model.Scalar)
	if !ok {
		log.Fatal("Result is not a scalar value")
	}

	return float64(scalar.Value)
}

func (c *Config) scaleUp() {
	ctx := context.Background()
	count := c.getCurrentAppSize(ctx)
	if count >= c.MaxSize {
		log.Printf("Already at maximum size\n")
		return
	}

	log.Printf("Scaling up\n")
	c.setAppSize(ctx, int64(count+1))
}

func (c *Config) scaleDown() {
	ctx := context.Background()
	count := c.getCurrentAppSize(ctx)
	if count <= 1 {
		log.Printf("Already at minimum size\n")
		return
	}

	log.Printf("Scaling down\n")
	c.setAppSize(ctx, int64(count-1))
}

func (c *Config) getCurrentAppSize(ctx context.Context) int {
	log.Printf("Getting current app size\n")

	cli := c.getDOAPIClient()

	app, _, err := cli.Apps.Get(ctx, c.DOAppID)
	if err != nil {
		log.Fatal(fmt.Errorf("Error getting app: %s", err))
	}

	services := app.Spec.GetServices()
	if len(services) == 0 {
		log.Fatal("No services found")
	}

	service := services[0] // todo: support multiple services
	size := int(service.GetInstanceCount())

	log.Printf("Current app size: %d\n", size)

	c.simpleWebServer.lastInstanceSize = size
	c.simpleWebServer.lastCheck = time.Now()

	return size
}

func (c *Config) getDOAPIClient() *godo.Client {
	if c.godoClient == nil {
		c.godoClient = godo.NewFromToken(c.DOAPIToken)
	}

	return c.godoClient
}

func (c *Config) setAppSize(ctx context.Context, size int64) {
	log.Printf("Setting app size to %d\n", size)

	cli := c.getDOAPIClient()

	app, _, err := cli.Apps.Get(ctx, c.DOAppID)
	if err != nil {
		log.Fatal(fmt.Errorf("Error getting app: %s", err))
	}

	services := app.Spec.GetServices()
	if len(services) == 0 {
		log.Fatal("No services found")
	}

	newAppSpec := &godo.AppUpdateRequest{
		Spec: app.Spec,
	}

	newAppSpec.Spec.Services[0].InstanceCount = size // todo: support multiple services

	_, _, err = cli.Apps.Update(ctx, c.DOAppID, newAppSpec)
	if err != nil {
		log.Fatal(fmt.Errorf("Error updating app: %s", err))
	}
}
