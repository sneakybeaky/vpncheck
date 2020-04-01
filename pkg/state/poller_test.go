package state

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	"github.com/clearchannelinternational/vpncheck/pkg/actor"
	"github.com/go-kit/kit/log"
	"testing"
	"time"
)

func TestPollingForOneRequest(t *testing.T) {

	status := make(chan []*ec2.VpnConnection)

	// Given a correctly configured ec2 client
	ec2Client := newMockEC2Client()
	expectedGatewayId := "blahblahblah"
	ec2Client.describeVpnConnections = describeVpnConnectionsWith(expectedGatewayId)

	duration := time.Hour
	underTest := pollerActor(log.NewNopLogger(), status, ec2Client, &duration)
	defer underTest.Interrupt(nil)

	// When the actor is run
	go func(a actor.Actor) {
		_ = a.Execute()
	}(underTest)

	// Then the vpn connection status should be sent down the channel
	var update []*ec2.VpnConnection
	select {
	case update = <-status:
		// expected - state has been updated
	case <-time.After(1 * time.Second):
		t.Errorf("No status was sent")
		return
	}

	if len(update) == 0 {
		t.Error("Should have received a populated update")
		return
	}

	if *update[0].VpnGatewayId != expectedGatewayId {
		t.Errorf("VPN Connection Details incorrect. Expected a gateway ID of `%s` but got `%s`", expectedGatewayId, *update[0].VpnGatewayId)
	}

}

func TestPollingErrorHandling(t *testing.T) {

	status := make(chan []*ec2.VpnConnection)

	// Given an incorrectly configured ec2 client
	ec2Client := newMockEC2Client()
	expectedError := errors.New("test error")
	ec2Client.describeVpnConnections = describeVpnConnectionsReturnsErr(expectedError)

	duration := time.Hour
	underTest := pollerActor(log.NewNopLogger(), status, ec2Client, &duration)
	defer underTest.Interrupt(nil)

	// When the actor is run
	foundErrors := make(chan error)
	go func(a actor.Actor) {
		foundErrors <- a.Execute()
	}(underTest)

	// any error is returned
	select {
	case err := <-foundErrors:
		// expected - the error has been propagated
		if err.Error() != expectedError.Error() {
			t.Errorf("Expected an error `%v` but got `%v`", expectedError, err)
		}
	case <-time.After(1 * time.Second):
		t.Errorf("No error was sent")
		return
	}

}

type mockEC2Client struct {
	ec2iface.EC2API
	describeVpnConnections func(*ec2.DescribeVpnConnectionsInput) (*ec2.DescribeVpnConnectionsOutput, error)
}

func (m *mockEC2Client) DescribeVpnConnections(input *ec2.DescribeVpnConnectionsInput) (*ec2.DescribeVpnConnectionsOutput, error) {
	return m.describeVpnConnections(input)
}

func newMockEC2Client() *mockEC2Client {
	return &mockEC2Client{
		describeVpnConnections: func(*ec2.DescribeVpnConnectionsInput) (*ec2.DescribeVpnConnectionsOutput, error) {
			return &ec2.DescribeVpnConnectionsOutput{}, nil
		},
	}
}

func describeVpnConnectionsWith(gatewayId string) func(*ec2.DescribeVpnConnectionsInput) (*ec2.DescribeVpnConnectionsOutput, error) {

	expectedGatewayId := aws.String(gatewayId)
	expectedConnection := &ec2.VpnConnection{VpnGatewayId: expectedGatewayId}
	expectedVpnConnections := []*ec2.VpnConnection{expectedConnection}

	return func(*ec2.DescribeVpnConnectionsInput) (*ec2.DescribeVpnConnectionsOutput, error) {
		return &ec2.DescribeVpnConnectionsOutput{
			VpnConnections: expectedVpnConnections,
		}, nil
	}
}

func describeVpnConnectionsReturnsErr(err error) func(*ec2.DescribeVpnConnectionsInput) (*ec2.DescribeVpnConnectionsOutput, error) {

	return func(*ec2.DescribeVpnConnectionsInput) (*ec2.DescribeVpnConnectionsOutput, error) {
		return nil, err
	}
}
