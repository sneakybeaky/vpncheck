package http

import (
	"bytes"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"html/template"
	"testing"
)

var nametests = []struct {
	name  string
	tags  []*ec2.Tag
	truth string
}{
	{name: "Pascal case", tags: []*ec2.Tag{asTag("Name", "blah")}, truth: "blah"},
	{name: "No tags", tags: []*ec2.Tag{}, truth: ""},
	{name: "No name tag", tags: []*ec2.Tag{asTag("foo", "bar")}, truth: ""},
	{name: "Lower case", tags: []*ec2.Tag{asTag("name", "bar")}, truth: "bar"},
}

func asTag(k string, v string) *ec2.Tag {
	return &ec2.Tag{Key: aws.String(k), Value: aws.String(v)}
}

func TestGetConnectionName(t *testing.T) {
	testTemplate, err := template.New("test").Funcs(templateFuncs).Parse("{{connectionName .Connection}}")

	if err != nil {
		t.Errorf("Unable to create test testTemplate: %v", err)
		return
	}

	b := new(bytes.Buffer)

	for _, tt := range nametests {
		t.Run(tt.name, func(t *testing.T) {

			var data = struct {
				Connection *ec2.VpnConnection
			}{&ec2.VpnConnection{Tags: tt.tags}}

			b.Reset()

			if err := testTemplate.Execute(b, &data); err != nil {
				t.Errorf("errored incorrectly : %v", err)
				return
			}

			if b.String() != tt.truth {
				t.Errorf("want %s; got %s", tt.truth, b.String())
			}

		})
	}
}
