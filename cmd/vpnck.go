package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	vpnhttp "github.com/clearchannelinternational/vpncheck/pkg/http"
	"github.com/clearchannelinternational/vpncheck/pkg/metrics"
	"github.com/clearchannelinternational/vpncheck/pkg/runtime"
	"github.com/clearchannelinternational/vpncheck/pkg/state"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"net"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/oklog/oklog/pkg/group"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	_ "net/http/pprof"
)

func main() {
	// Define our flags.
	fs := flag.NewFlagSet("vpnck", flag.ExitOnError)
	var (
		debugAddr = fs.String("debug-addr", ":8081", "Debug and metrics listen address")
		httpAddr  = fs.String("http-addr", ":8080", "HTTP listen address")
		insecure  = fs.Bool("insecure", false, "Ignore invalid server TLS certificates")
		debug     = fs.Bool("debug", false, "More verbose logging")
		interval  = fs.Duration("interval", 5*time.Minute, "Time between polling the VPN status")
	)

	fs.Usage = usageFor(fs, os.Args[0]+" [flags]")
	fs.Parse(os.Args[1:])

	if *insecure {
		disableTlsVerify()
	}
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := ec2.New(sess)

	// Create a single logger, which we'll use and give to other components.
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)

		if *debug {
			logger = level.NewFilter(logger, level.AllowAll())
		} else {
			logger = level.NewFilter(logger, level.AllowInfo())
		}

	}

	var currentState state.State
	var handlers = &vpnhttp.StateHandlers{State: &currentState}

	http.DefaultServeMux.Handle("/metrics", promhttp.Handler())

	// Now we're to the part of the func main where we want to start actually
	// running things, like servers bound to listeners to receive Connections.
	//
	// The method is the same for each component: add a new actor to the group
	// struct, which is a combination of 2 anonymous functions: the first
	// function actually runs the component, and the second function should
	// interrupt the first function and cause it to return. It's in these
	// functions that we actually bind the Go kit server/handler structs to the
	// concrete transports and run them.
	//
	// Putting each component into its own block is mostly for aesthetics: it
	// clearly demarcates the scope in which each listener/socket may be used.
	var g group.Group
	{
		// The debug listener mounts the http.DefaultServeMux, and serves up
		// stuff like the Prometheus metrics route, the Go debug and profiling
		// routes, and so on.
		debugListener, err := net.Listen("tcp", *debugAddr)
		if err != nil {
			_ = logger.Log("transport", "debug/HTTP", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			_ = logger.Log("transport", "debug/HTTP", "addr", *debugAddr)
			return http.Serve(debugListener, http.DefaultServeMux)
		}, func(error) {
			_ = debugListener.Close()
		})
	}
	{
		// The HTTP listener mounts the Go kit HTTP handler we created.
		httpListener, err := net.Listen("tcp", *httpAddr)
		if err != nil {
			_ = logger.Log("transport", "HTTP", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			_ = logger.Log("transport", "HTTP", "addr", *httpAddr)
			return http.Serve(httpListener, handlers.Handler())
		}, func(error) {
			_ = httpListener.Close()
		})
	}

	// Assemble the stages for the pipeline that polls for updates, publishes metrics and updates
	// state for surfacing from our HTTP handlers.
	// This is a SEDA (https://stackoverflow.com/questions/3570610/what-is-seda-staged-event-driven-architecture) style approach
	{

		// Add the stage that exposes the state for HTML pages to render. This stage is a sink
		status := make(chan []*ec2.VpnConnection)
		state.AddMonitorStage(&g, logger, status, state.NewUTCClock(), &currentState)

		// Add the stage that exposes the metrics for Prometheus to collect. This stage is a sink.
		collector := metrics.NewVpnStatusCollector(prometheus.DefaultRegisterer, logger)
		collector.AddAsStage(&g)

		// Add the stage that updates the metrics every time new VPN telemetry data is received, and sends to next stage
		vpnUpdates := make(chan []*ec2.VpnConnection)
		metrics.AddUpdaterStage(&g, logger, collector, vpnUpdates, status)

		// Add the stage that periodically fetches VPN telemetry data and sends to the next stage. This stage is a generator.
		state.AddPollerStage(&g, logger, vpnUpdates, svc, interval)
	}

	// Finally add a shutdown hook to the run group
	runtime.Shutdown(&g, logger)

	_ = logger.Log("exit", g.Run())
}

// disableTlsVerify turns of verification of any TLS certificates
func disableTlsVerify() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
}

func usageFor(fs *flag.FlagSet, short string) func() {
	return func() {
		_, _ = fmt.Fprintf(os.Stderr, "USAGE\n")
		_, _ = fmt.Fprintf(os.Stderr, "  %s\n", short)
		_, _ = fmt.Fprintf(os.Stderr, "\n")
		_, _ = fmt.Fprintf(os.Stderr, "FLAGS\n")
		w := tabwriter.NewWriter(os.Stderr, 0, 2, 2, ' ', 0)
		fs.VisitAll(func(f *flag.Flag) {
			_, _ = fmt.Fprintf(w, "\t-%s %s\t%s\n", f.Name, f.DefValue, f.Usage)
		})
		_ = w.Flush()
		_, _ = fmt.Fprintf(os.Stderr, "\n")
	}
}
