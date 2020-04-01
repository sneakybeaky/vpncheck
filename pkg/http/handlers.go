package http

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	vpn "github.com/clearchannelinternational/vpncheck/pkg/state"
	"html/template"
	"net/http"
	"strings"
	"time"
)

type StateHandlers struct {
	*vpn.State
}

var templateFuncs = template.FuncMap{
	"connectionName": getConnectionName,
}

func (s StateHandlers) Handler() http.Handler {

	mux := http.NewServeMux()
	mux.HandleFunc("/raw", s.rawHandler)
	mux.HandleFunc("/", s.defaultHandler)
	return mux
}

func (s StateHandlers) rawHandler(w http.ResponseWriter, r *http.Request) {

	t, err := template.ParseFiles("templates/raw.gohtml")
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to read template file: %v", err), http.StatusInternalServerError)
		return
	}

	var data = struct {
		Timestamp   string
		Connections []*ec2.VpnConnection
	}{
		fmt.Sprintf("State recorded at %s:\n", s.Timestamp),
		s.Connections,
	}
	if err := t.Execute(w, &data); err != nil {
		http.Error(w, fmt.Sprintf("Unable to render result: %v", err), http.StatusInternalServerError)
	}
	return
}

func (s StateHandlers) defaultHandler(w http.ResponseWriter, r *http.Request) {

	t, err := template.New("index.gohtml").Funcs(templateFuncs).ParseFiles("templates/index.gohtml")

	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to read template file: %v", err), http.StatusInternalServerError)
		return
	}

	var data = struct {
		Timestamp   time.Time
		Connections []*ec2.VpnConnection
	}{
		s.Timestamp,
		s.Connections,
	}
	if err := t.Execute(w, &data); err != nil {
		http.Error(w, fmt.Sprintf("Unable to render result: %v", err), http.StatusInternalServerError)
	}
	return
}

var noName = ""

var getConnectionName = func(connection ec2.VpnConnection) *string {

	for _, tag := range connection.Tags {
		if strings.ToLower(*tag.Key) == "name" {
			return tag.Value
		}
	}

	return &noName
}
