# Technical Addon (TA) Runner

Run technical addons with this simple utility program.

The TA runner can interpret `inputs.conf`, `transforms.conf` and `props.conf` files and runs technical addons according to these settings.

This project is under active development. You can consult the [roadmap](https://github.com/splunk/tarunner/issues) to learn more. 

This program exports all data over the OpenTelemetry Protocol (OTLP). It can be used with [Splunk Connect for OTLP](https://github.com/splunk/splunk-connect-for-otlp) to send data to a Splunk instance.

# Getting started

* Download the binary from the [latest release](https://github.com/splunk/tarunner/releases)
* Run the binary with the following arguments:
  
  `> tarunner <basedir> <otlp-endpoint>`
  
  `basedir`: the location of the technical addon, uncompressed.
  
  `otlp-endpoint`: the OTLP gRPC endpoint to target with the runner. Example: `http://localhost:4317`

## Using Docker

Build the Docker image:
* `> docker build -t tarunner .`

Run the image:
* `> docker run --rm -v $(pwd)/ta:/ta /ta http://endpoint:4317`

See also under the `integration` folder a `docker-compose.yml` example.

Run the example with: `docker compose up`

# Modes

## UF mode (raw data)

In this mode, the TA runner will run the scripts, modinputs, monitors, capturing their output.
It will tag them with host, source and sourcetype fields.

UF mode is the default mode.

## HF mode (cooked data)

In this mode, the TA runner performs the steps of the UF mode and additional performs index time actions:
* Indexed extractions
* Ingest eval
* Rulesets
* Transforms

HF mode is experimental and incomplete. This [issue](https://github.com/splunk/tarunner/issues/9) tracks the work.

The mode can be enabled by running the runner with `--feature-flags +cook`.

# License

The TA Runner is licensed under Apache Software License 2.0. See [LICENSE](./LICENSE).