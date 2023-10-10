# Auto-Scaler for DigitalOcean Apps using Prometheus Metrics

This program automatically scales DigitalOcean apps based on Prometheus metrics. It monitors a specified metric and scales the application instances up or down based on defined threshold values.

## Features

1. **Configurable Scaling**: Scale your application instances up or down based on metric values surpassing the defined thresholds.
2. **Integration with DigitalOcean Apps**: Seamless integration with DigitalOcean's App Platform.
3. **Uses Prometheus**: Leverages Prometheus for fetching metric data.

## Prerequisites

- [DigitalOcean](https://www.digitalocean.com/) account.
- An instance of [Prometheus](https://prometheus.io/) to monitor metrics.

## Setup & Configuration

1. **Environment Variables**: Set the following environment variables:
   - `PROMETHEUS_HOST`: The URL of your Prometheus instance.
   - `THRESHOLD_UP`: The metric value threshold to trigger scaling up.
   - `MAX_SIZE`: The maximum number of instances for your app.
   - `THRESHOLD_DOWN`: The metric value threshold to trigger scaling down.
   - `DO_API_TOKEN`: Your DigitalOcean API token.
   - `DO_APP_ID`: The ID of your DigitalOcean app.
   - `PROMETHEUS_METRIC`: The Prometheus metric to monitor.

2. **godotenv Support**: The program supports loading the environment variables from a `.env` file.

## How it Works

1. The program continuously checks the specified Prometheus metric.
2. If the metric value surpasses the `THRESHOLD_UP`, the program scales up the application, provided the number of instances hasn't reached `MAX_SIZE`.
3. If the metric value falls below the `THRESHOLD_DOWN`, the program scales down the application, but ensures there's always at least one instance running.

## Limitations

- Currently, the program assumes that your DigitalOcean app has only one service. Multiple services are not supported yet.

## Future Improvements

- Support for scaling multiple services within a DigitalOcean app.
- Customizable sleep duration between metric checks.
- Handle Prometheus and DigitalOcean API connection issues gracefully.

## Usage

Run the program using:

    go run main.go


