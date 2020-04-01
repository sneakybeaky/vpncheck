package metrics

import (
	"fmt"
	"github.com/Pallinder/go-randomdata"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"strings"
	"testing"
)

var tunneltests = []struct {
	name string
	test *telemetryAndTruth
}{
	{name: "Empty telemetry", test: &telemetryAndTruth{}},
	{name: "One tunnel up", test: testCaseFor(1, 0)},
	{name: "One tunnel down", test: testCaseFor(0, 1)},
	{name: "One tunnel up, one down", test: testCaseFor(1, 1)},
}

func TestTunnelUp(t *testing.T) {

	for _, tt := range tunneltests {
		t.Run(tt.name, func(t *testing.T) {
			underTest := NewVpnStatusCollector(prometheus.NewRegistry(), log.NewNopLogger())
			defer underTest.Interrupt(nil)

			// When the actor is run
			go func(c *vpnCollector) {
				_ = c.Execute()
			}(underTest)

			underTest.Update(tt.test.telemetry)

			if err := testutil.CollectAndCompare(underTest, strings.NewReader(tt.test.truth)); err != nil {
				t.Errorf("unexpected collecting result:\n%s", err)
			}
		})
	}
}

var oneUpOneDown = testCaseFor(1, 1)

type updatetest struct {
	name         string
	firstupdate  *telemetryAndTruth
	secondupdate *telemetryAndTruth
}

var updatedtests = []updatetest{
	{name: "No common connections between updates", firstupdate: testCaseFor(1, 0), secondupdate: testCaseFor(0, 1)},
	{name: "Identical data between updates", firstupdate: oneUpOneDown, secondupdate: oneUpOneDown},
}

func TestMetricsUpdated(t *testing.T) {

	updatedtests = append(updatedtests, someCommonDataBetweenUpdates())
	updatedtests = append(updatedtests, oneTunnelChangingState())

	for _, tt := range updatedtests {
		t.Run(tt.name, func(t *testing.T) {

			underTest := NewVpnStatusCollector(prometheus.NewRegistry(), log.NewNopLogger())
			defer underTest.Interrupt(nil)

			// When the actor is run
			go func(c *vpnCollector) {
				_ = c.Execute()
			}(underTest)

			underTest.Update(tt.firstupdate.telemetry)

			if err := testutil.CollectAndCompare(underTest, strings.NewReader(tt.firstupdate.truth)); err != nil {
				t.Errorf("First update failed:\n%s", err)
				return
			}

			underTest.Update(tt.secondupdate.telemetry)

			if err := testutil.CollectAndCompare(underTest, strings.NewReader(tt.secondupdate.truth)); err != nil {
				t.Errorf("Second update failed:\n%s", err)
				return
			}

		})
	}
}

// Holds vpn connection data and the corresponding metric output for it
type telemetryAndTruth struct {
	telemetry []*ec2.VpnConnection
	truth     string
}

// testCaseFor creates test input and expected string representation
func testCaseFor(up int, down int) *telemetryAndTruth {

	gwid := randomdata.RandStringRunes(10)
	telemetry := append(genTelemetry(down, aws.String(ec2.TelemetryStatusDown)), genTelemetry(up, aws.String(ec2.TelemetryStatusUp))...)

	return &telemetryAndTruth{
		telemetry: []*ec2.VpnConnection{
			{VpnGatewayId: aws.String(gwid),
				VgwTelemetry: telemetry},
		},
		truth: expectedOutputFor(gwid, telemetry),
	}

}

func expectedOutputFor(gwid string, telemetry []*ec2.VgwTelemetry) string {

	const metadata = `
		# HELP cc_vpn_tunnel_up If the site to site VPN tunnel status is up, partitioned by VPN Connection ID and Outside IP.
		# TYPE cc_vpn_tunnel_up gauge
	`
	var str strings.Builder
	str.WriteString(metadata)

	str.WriteString(toExpectedMetricString(gwid, telemetry))

	return str.String()
}

// Set up test case where one tunnel changes state between updates
func oneTunnelChangingState() updatetest {

	gwid := randomdata.RandStringRunes(10)

	// First update has the tunnel UP
	first := genTelemetry(1, aws.String(ec2.TelemetryStatusUp))

	firstUpdate := &telemetryAndTruth{
		telemetry: []*ec2.VpnConnection{
			{VpnGatewayId: aws.String(gwid),
				VgwTelemetry: first},
		},
		truth: expectedOutputFor(gwid, first),
	}

	// Second update has the same tunnel but DOWN
	second := []*ec2.VgwTelemetry{
		{
			Status:           aws.String(ec2.TelemetryStatusDown),
			OutsideIpAddress: first[0].OutsideIpAddress,
		},
	}

	secondUpdate := &telemetryAndTruth{
		telemetry: []*ec2.VpnConnection{
			{VpnGatewayId: aws.String(gwid),
				VgwTelemetry: second},
		},
		truth: expectedOutputFor(gwid, second),
	}

	return updatetest{name: "One tunnel with status changing", firstupdate: firstUpdate, secondupdate: secondUpdate}

}

// Set up test case where some tunnels are common between updates
func someCommonDataBetweenUpdates() updatetest {

	gwid := randomdata.RandStringRunes(10)
	shared := append(genTelemetry(2, aws.String(ec2.TelemetryStatusDown)), genTelemetry(2, aws.String(ec2.TelemetryStatusUp))...)

	var firstSharedUpdate, secondSharedUpdate *telemetryAndTruth

	{
		firstupdate := append(append(genTelemetry(1, aws.String(ec2.TelemetryStatusDown)), genTelemetry(1, aws.String(ec2.TelemetryStatusUp))...), shared...)

		firstSharedUpdate = &telemetryAndTruth{
			telemetry: []*ec2.VpnConnection{
				{VpnGatewayId: aws.String(gwid),
					VgwTelemetry: firstupdate},
			},
			truth: expectedOutputFor(gwid, firstupdate),
		}
	}
	{
		secondupdate := append(append(genTelemetry(1, aws.String(ec2.TelemetryStatusDown)), genTelemetry(1, aws.String(ec2.TelemetryStatusUp))...), shared...)

		secondSharedUpdate = &telemetryAndTruth{
			telemetry: []*ec2.VpnConnection{
				{VpnGatewayId: aws.String(gwid),
					VgwTelemetry: secondupdate},
			},
			truth: expectedOutputFor(gwid, secondupdate),
		}
	}

	return updatetest{name: "Some shared data between updates", firstupdate: firstSharedUpdate, secondupdate: secondSharedUpdate}

}

// toExpectedMetricString generates the expected metrics as a string for the supplied tunnel data
func toExpectedMetricString(gwid string, tunnels []*ec2.VgwTelemetry) string {
	var str strings.Builder

	for _, tunnel := range tunnels {

		status := 0

		if *tunnel.Status == ec2.TelemetryStatusUp {
			status = 1
		}

		str.WriteString(fmt.Sprintf("cc_vpn_tunnel_up{outside_ip=\"%s\",vpn_id=\"%s\"} %d\n", *tunnel.OutsideIpAddress, gwid, status))
	}

	return str.String()

}

// Builds a test case including the supplied data
func genTelemetry(required int, status *string) []*ec2.VgwTelemetry {

	telemetry := make([]*ec2.VgwTelemetry, 0)

	for i := 0; i < required; i++ {
		ip := randomdata.IpV4Address()
		telemetry = append(telemetry, &ec2.VgwTelemetry{Status: status, OutsideIpAddress: aws.String(ip)})
	}

	return telemetry

}
