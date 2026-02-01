# esphome2prom

esphome2prom listens to ESPHome MQTT discovery and sensor topics, converts ESPHome sensor payloads into Prometheus-style metrics and forwards them to a Prometheus remote write endpoint (with optional local `/prometheus/metrics` endpoint for scraping) 

## Features

### Supported sensor types

See `queue/sensor.go` and `queue/queue.go` for adding more

-  temperature


## How it works 

- The daemon connects to an MQTT broker and subscribes to Home Assistant/ESPHome discovery topic (by default `homeassistant/#`) and common paths used for sensor data:
  - `+/sensor/#` 
- Sensor messages are converted metric units, enriched with configured labels, and pushed to a write queue that forwards metrics to the configured Prometheus remote write endpoint.


## Build

`make` / `make static`

standard go build method should work but it will not embed the app version var


## Run

`MQTT_ADDR="tcp://mqtt:mqtt@example.com:1883" PROMETHEUS_WRITE_URL=http://example.com:8480/insert/999:0/prometheus/api/v1/write ./esphome2prom`

There is a [SystemD unit](esphome2prom.service) available in the repo

Flags

- --mqtt-addr         MQTT broker address (default `127.0.0.1:1883`)  
  or env var `MQTT_ADDR`  

- --listen-addr       HTTP listen address for the web UI (default `127.0.0.1:3001`)  
  or env var `LISTEN_ADDR`.
  That will start the http listener. 

- --pprof-addr        If set, starts pprof on the provided address (e.g. `127.0.0.1:6060`)

- --extra-labels      Comma-separated `key=value` pairs (or use the equivalent env/flag mapping) that will be added to every metric. By default includes the host name.

- --debug             Enable debug logging

- Environment:
    - PROMETHEUS_WRITE_URL — URL of the Prometheus remote-write endpoint. If not set, writing metrics will be disabled or fail depending on config.

Example:
```
./esphome2prom \
    --mqtt-addr tcp://user:pass@mqtt.example.local:1883 \
    --prometheus-write-url https://prometheus-remote-write.example/api/v1/write \
    --extra-labels location=home,site=livingroom \
    --debug
```
or with environment variables:

export MQTT_ADDR="tcp://user:pass@mqtt.example.local:1883"
export PROMETHEUS_WRITE_URL="https://prometheus-remote-write.example/api/v1/write"
./esphome2prom --listen-addr 0.0.0.0:3001

## Configuration files

The program uses `yamlcfg` to optionally load configuration from a list of files (examples shown in code):

- `$HOME/.config/my/cnf.yaml`
- `./cfg/config.yaml`
- `/etc/my/cnf.yaml`

The repository contains a minimal `config.Config` struct:

```yaml
# example config/config.yaml
address: "0.0.0.0:3001"
```

Note: Most runtime options are controlled via CLI flags and environment variables; the YAML config support is minimal in the current code.

## Metrics

- nodes called `ignoreme` will be ignored. This is so new esphome node can be tested before metrics are being sent 
- Metrics are emitted as simple Prometheus metrics (name, labels, value, timestamp) via the configured remote-write URL.
- Temperature sensors are converted to Celsius if unit indicates Kelvin or Fahrenheit. Metric names and label keys follow simple conventions (device, sensor name, plus any `extra-labels` provided).

## Development / Local web assets

The binary embeds the `static` and `templates` directories. If you have local `./static` and `./templates` directories when starting the binary, the program will prefer local files — useful for developing the web frontend without rebuilding the binary.

## Logging & Debugging

- Uses `zap` for structured logging. Add `--debug` to enable development logging and stack traces.
- Optional pprof: pass `--pprof-addr` to enable the Go pprof HTTP server for profiling.
- The program logs discovery and sensor registration events to help debugging MQTT topics and payloads.

## Example systemd unit

Example unit for running on Linux:

```ini
[Unit]
Description=esphome2prom
After=network.target

[Service]
User=esphome
Group=esphome
Environment="MQTT_ADDR=tcp://user:pass@127.0.0.1:1883"
Environment="PROMETHEUS_WRITE_URL=https://prometheus-remote-write.example/api/v1/write"
ExecStart=/opt/esphome2prom/esphome2prom --listen-addr 0.0.0.0:3001
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

## Troubleshooting

- No metrics appear:
    - Ensure `PROMETHEUS_WRITE_URL` is set and reachable.
    - Check logs for registration of sensors and errors from `promwriter`.
    - Verify MQTT connectivity and credentials; you can test with mosquitto_sub/publish.

- Sensors not detected:
    - The app listens for Home Assistant discovery messages. Confirm ESPHome devices publish discovery to `homeassistant/...`.
    - Check MQTT topics with a subscriber (e.g., `mosquitto_sub -t "#" -v`).

## Contributing

- Bug reports and PRs welcome.
- Unit tests: there is a placeholder test file. Add tests under `..._test.go` files and run `go test ./...`.

## License

(Choose and include a license for the repository — e.g. MIT, Apache-2.0 — before publishing.)

---
I reviewed the code (main, queue, sensor, web, Makefile) to capture runtime flags, environment variables, and build steps and used that to create this README. If you'd like, I can open a branch and create a PR that adds this README (and optionally a LICENSE), or expand the README with example MQTT payloads, example prometheus metric payloads, or a Dockerfile.