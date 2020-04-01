# Developing with vpn-check

The layout of this repo more or less follows [this layout](https://peter.bourgon.org/go-best-practices-2016/#repository-structure) and the principles in https://peter.bourgon.org/go-for-industrial-programming/

The entry point for the code is at `cmd/vpnck.go`. Hopefully reading through that should show the general structure.

The core functionality is set up in a [SEDA](https://medium.com/@miko.goldstein/the-seda-architecture-b085310294fb) style, with stages implemented with go routines and the events sent down channels.

Essentially AWS is polled for VPN status, which is passed down to a stage that exposes those metrics for Prometheus to collect. They are then passed down to a stage that makes them available to show in the HTML pages rendered by handlers. 

## Running locally

As the code interacts with AWS API and uses assume role we use [hoverfly](https://hoverfly.io/) to replay captured traffic for local use.

Install hoverfly by `brew install SpectoLabs/tap/hoverfly`

Use the `hoverfly_captures/start.sh` script to start hoverfly and load the happy path API interaction

Make a note of the `HTTP_PROXY` setting the script shows at the end

As we're replaying captured traffic we don't need real AWS credentials, but something that *looks* valid must be present for the code to run. So to run our code we set a fake AWS key and set a fake role to assume and trust any TLS server (to speak with hoverfly)  

```bash
HTTP_PROXY="http://localhost:8500" \
  AWS_REGION="eu-west-1" \
  AWS_ACCESS_KEY_ID="abcdfrehgewwedsa" \
  AWS_SECRET_ACCESS_KEY="9087kfjxhb92387kjfdh21123113" \
   go run cmd/vpnck.go \
    -interval 10s \
    -debug \
    -insecure  
```

Now hitting http://localhost:8080/ should show a happy VPN.

## Running tests

From the root of the project

    go test -timeout 300ms -count=1  -v ./...
    

## Other simulations

Simulations of different AWS responses are in the `hoverfly_captures` directory. See the [README](hoverfly_captures/README.md)

### Using skaffold
You can use skaffold to iterate quickly by detecting code changes and deploying to a k8s cluster.  Be aware this will push docker images to the ECR repo and have other side effects.
If unsure ask for help.

See the [README](skaffold/README.md) for minimal instructions.
