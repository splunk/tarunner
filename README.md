# Technical Addon (TA) Runner

The TA runner is a wrapper for a Splunk Technical Addon to run without the presence of a Splunk instance.

The TA runner can interpret `inputs.conf`, `transforms.conf` and `props.conf` files runs technical addons according to these settings.

It exports all data over the OpenTelemetry Protocol (OTLP).

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

# License

The TA Runner is licensed under Apache Software License 2.0. See [LICENSE](./LICENSE).