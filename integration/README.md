# Example of deployment with a Splunk instance

Please follow the steps below to run this example.

## Deploy a local Splunk instance

In this folder, run:

```> docker compose --profile splunk up -d```

This will deploy a Splunk instance locally. The instance will start up and be available over localhost:18000 with the credentials `admin` and `changeme`.

## Download and install Splunk Connect for OTLP

Download and install Splunk Connect for OTLP per the steps outlined in https://github.com/splunk/splunk-connect-for-otlp.

Make sure to configure the OTLP endpoint to use 0.0.0.0 so it is exposed by the docker container.

## Download the Splunk addon for Linux

Install the TA for Linux, downloading it from https://splunkbase.splunk.com/app/833

## Install the TA for Linux on the Splunk instance

Go to `Manage Apps`, install it from your download as a tgz file.

## Install The Splunk App for Content Packs (optional)

Download and install this Splunk app from https://splunkbase.splunk.com/app/5391

This app will show the dashboards associated with the data from the TA.

See https://help.splunk.com/en/splunk-it-service-intelligence/content-packs-for-itsi-and-ite/unix-dashboards-and-reports/1.3/install-the-content-pack-for-unix-dashboards-and-reports for more information.

## Set up the linux TA for tarunner

In this folder, run (replace the location as per your download for the Linux TA):

```> tar xzvf ~/Downloads/splunk-add-on-for-unix-and-linux_1020.tgz```

In the `Splunk_TA_nix` folder created, copy the `default` folder as `local`.

```> cd Splunk_TA_nix && cp -r default local```

Open local/inputs.conf and edit each `disabled = 1` line to `disabled = 0`.

## Run tarunner

```> docker compose --profile splunk --profile tarunner up -d --build```
