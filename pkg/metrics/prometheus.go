package metrics

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/oklog/pkg/group"
	"github.com/prometheus/client_golang/prometheus"
	"sort"
	"strings"
)

// tunnelUpGauge wraps a Gauge from the Prometheus client library for lifecycle management.
// This allows us to dynamically add and remove gauges as needed
type tunnelUpGauge struct {
	id     string
	labels prometheus.Labels
	gauge  prometheus.Gauge
	delete func()
}

// newTunnelUpGauge returns a populated gauge with the supplied details
func newTunnelUpGauge(id string, labels prometheus.Labels, gauge prometheus.Gauge, delete func()) *tunnelUpGauge {
	return &tunnelUpGauge{
		id:     id,
		labels: labels,
		gauge:  gauge,
		delete: delete,
	}
}

// updateFrom updates the gauge from the supplied tunnel telemetry data
func (t *tunnelUpGauge) updateFrom(telemetry *ec2.VgwTelemetry) *tunnelUpGauge {

	status := 1.0
	if *telemetry.Status != ec2.TelemetryStatusUp {
		status = 0
	}

	t.gauge.Set(status)

	return t
}

type Updater interface {
	Update(connections []*ec2.VpnConnection)
}

// vpnCollector manages prometheus metrics for VPNs we care about.
// As VPN components can come and go we have to add a layer of management on top of the standard Prometheus functionality
type vpnCollector struct {
	tunnelUpGaugeVec *prometheus.GaugeVec
	gauges           map[string]*tunnelUpGauge
	collect          chan *collectAndDone
	update           chan []*ec2.VpnConnection
	cancel           chan struct{}
	logger           log.Logger
}

// NewVpnStatusCollector returns an instance ready to use. The Execute() method should be called from a go routine to process updates and publish metrics, with the Interrupt() method being called to signal that process should stop.
func NewVpnStatusCollector(registerer prometheus.Registerer, logger log.Logger) *vpnCollector {

	c := vpnCollector{
		tunnelUpGaugeVec: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "cc",
				Subsystem: "vpn",
				Name:      "tunnel_up",
				Help:      "If the site to site VPN tunnel status is up, partitioned by VPN Connection ID and Outside IP.",
			},
			[]string{
				// Which VPN ?
				"vpn_id",
				// and what's the Outside IP ?
				"outside_ip",
			},
		),
		gauges:  make(map[string]*tunnelUpGauge),
		collect: make(chan *collectAndDone),
		cancel:  make(chan struct{}),
		update:  make(chan []*ec2.VpnConnection),
		logger:  log.With(logger, "actor", "vpncollector"),
	}

	registerer.MustRegister(c.tunnelUpGaugeVec)

	return &c
}

// AddAsStage adds as a stage to the supplied run group
func (c *vpnCollector) AddAsStage(group *group.Group) {
	group.Add(c.Execute, c.Interrupt)
}

// Execute is the heart of this unit. All logic that deals with state happens here.
// This should be called from a go routine, with the Interrupt() method used to signal this body should be exited.
func (c *vpnCollector) Execute() error {

	_ = level.Debug(c.logger).Log("msg", "started execute loop")

	for {
		select {
		case collect := <-c.collect:
			_ = level.Debug(c.logger).Log("msg", "sending metrics to collector")
			c.tunnelUpGaugeVec.Collect(collect.ch)
			collect.finished()
		case connections := <-c.update:
			_ = level.Debug(c.logger).Log("msg", "received new VPN status")
			c.updateWith(connections)
		case <-c.cancel:
			_ = level.Info(c.logger).Log("msg", "received cancellation - exiting loop")
			return nil
		}
	}

}

// Interrupt signals that the processing of updates & publishing metrics should finish.
// Once called instances cannot be re-used.
func (c *vpnCollector) Interrupt(err error) {
	_ = level.Info(c.logger).Log("interrupted", fmt.Sprintf("interrupted with %v", err))

	close(c.cancel)
}

// Describe returns all descriptions of the managedGauge.
func (c *vpnCollector) Describe(ch chan<- *prometheus.Desc) {
	c.tunnelUpGaugeVec.Describe(ch)
}

// Collect returns the current state of all metrics of the managedGauge.
func (c *vpnCollector) Collect(ch chan<- prometheus.Metric) {

	cd := &collectAndDone{
		ch:   ch,
		done: make(chan interface{}),
	}
	c.collect <- cd

	cd.wait()

}

// Update refreshes metrics with the tunnel connection data
func (c *vpnCollector) Update(connections []*ec2.VpnConnection) {
	c.update <- connections
}

// Update updates the metric gauges with the current state of the VPNs.
// Collectors for tunnels that have been removed are deleted, and new ones are created.
func (c *vpnCollector) updateWith(connections []*ec2.VpnConnection) {

	// Gauges we want to keep
	currentGauges := make(map[string]*tunnelUpGauge)

	for _, conn := range connections {

		for _, tunnel := range conn.VgwTelemetry {

			labels := labelsForTunnelGauge(*conn.VpnGatewayId, *tunnel.OutsideIpAddress)
			id := idForTunnelGauge(labels)

			if existingGauge, ok := c.gauges[id]; ok {

				_ = level.Debug(c.logger).Log("msg", fmt.Sprintf("Updating existing gauge: %v", labels))

				existingGauge.updateFrom(tunnel)
				currentGauges[id] = existingGauge
				delete(c.gauges, id)

			} else {

				_ = level.Debug(c.logger).Log("msg", fmt.Sprintf("Adding gauge for new tunnel instance: %v", labels))

				newGauge := newTunnelUpGauge(id,
					labels,
					c.tunnelUpGaugeVec.With(labels),
					func() {
						c.tunnelUpGaugeVec.Delete(labels)
					},
				).updateFrom(tunnel)

				currentGauges[newGauge.id] = newGauge

			}

		}

	}

	for _, redundantCollector := range c.gauges {
		_ = level.Debug(c.logger).Log("msg", fmt.Sprintf("Removing redundant gauge: %v", redundantCollector.id))
		delete(c.gauges, redundantCollector.id)
		redundantCollector.delete()
	}

	c.gauges = currentGauges

}

// collectAndDone is used to perform the Prometheus Collect operation from a go routine.
type collectAndDone struct {
	// The channel to be used for the Collect operation
	ch chan<- prometheus.Metric

	// signals when the collect operation is complete by closing this channel
	done chan interface{}
}

// wait blocks until the collect is completed
func (c collectAndDone) wait() {
	<-c.done
}

// finished signals the collect is completed
func (c collectAndDone) finished() {
	close(c.done)
}

// buildCollectorID returns an id that distinguishes a gauge from any other
func buildCollectorID(metricName string, labels prometheus.Labels) string {
	var labelNamesValues []string
	for name, value := range labels {
		labelNamesValues = append(labelNamesValues, name, value)
	}
	sort.Strings(labelNamesValues)
	return metricName + ":" + strings.Join(labelNamesValues, "|")
}

func labelsForTunnelGauge(gatewayId string, outsideIP string) prometheus.Labels {
	return prometheus.Labels{
		"vpn_id":     gatewayId,
		"outside_ip": outsideIP,
	}
}

func idForTunnelGauge(labels prometheus.Labels) string {
	return buildCollectorID("tunnel_up", labels)
}
