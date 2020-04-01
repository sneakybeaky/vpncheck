# VPN check service

This service polls any VPN via the AWS [DescribeVpnConnections](https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeVpnConnections.html) and displays a human friendly result page.


## Usage manual

```console
USAGE
  vpnck [flags]

FLAGS
  -debug false       More verbose logging
  -debug-addr :8081  Debug and metrics listen address
  -http-addr :8080   HTTP listen address
  -insecure false    Ignore invalid server TLS certificates
  -interval 5m0s     Time between polling the VPN status
```

### Optional flags

##### `-debug-addr` 

The address the debug & metrics endpoint will listen to

##### `-http-addr` 

The main HTTP listen address

##### `-interval` 

Time between checking the VPN status, in the format that [time.ParseDuration](https://golang.org/pkg/time/#ParseDuration) accepts.
A duration string is a possibly signed sequence of decimal numbers, each with optional fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".

##### `-insecure` 

Accept any TLS certificate presented by the server and any host name in that certificate. In this mode, TLS is susceptible to man-in-the-middle attacks.
This should be used only for testing.

##### `-debug` 

Show more detailed logs


## Other configuration

Configuration for [using the AWS API](https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html) must be set up. When running in a k8s setup typically the only thing you will need to configure is the AWS Region to use - e.g. `AWS_REGION=eu-west-1` 
