# Technical Addon (TA) Runner

Run technical addons with this simple utility program.

The TA runner can interpret `inputs.conf`, `transforms.conf` and `props.conf` files and runs technical addons according to these settings.

This project is under active development. You can consult the [roadmap](https://github.com/splunk/tarunner/issues) to learn more. 

This program exports all data over the OpenTelemetry Protocol (OTLP). It can be used with [Splunk Connect for OTLP](https://github.com/splunk/splunk-connect-for-otlp) to send data to a Splunk instance.

# Getting started

* Download the binary from the [latest release](https://github.com/splunk/tarunner/releases)
* Run the binary with the following arguments:
  
  `> tarunner <basedir>`
  
  `basedir`: the location of the technical addon, uncompressed.
  
  The tarunner expects a tarunner.yaml file located at the root of the TA folder.

  The tarunner.yaml file consists of 3 fields:
  * `type`: the type of exporter to use. `otlp_http` will use the OTLP HTTP exporter (default value). Any other value is interpreted as sending over Splunk HEC.
  * `endpoint`: the endpoint to which to send the data. `http://localhost:4318` is the default value.
  * `token`: the token to set if sending over HEC.
  
## Using Docker

Build the Docker image:
* `> docker build -t tarunner .`

Run the image:
* `> docker run --rm -v $(pwd)/ta:/ta /ta`

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