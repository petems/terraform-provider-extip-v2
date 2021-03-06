package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const testDataSourceParameterValue = `
data "extip" "parameter_tests" {
  %s = "%s"
}
`

var parametertests = []struct {
	parameter  string
	value      string
	errorRegex string
}{
	{"this_doesnt_exist", "foo", "An argument named \"this_doesnt_exist\" is not expected here."},
	{"resolver", "https://notrealsite.fakeurl", "lookup notrealsite.fakeurl.+no such host"},
	{"resolver", "not-a-valid-url", "expected \"resolver\" to have a host, got not-a-valid-url"},
}

func TestAccDataSourceExtipInvalidParameters(t *testing.T) {
	for _, tt := range parametertests {
		resource.UnitTest(t, resource.TestCase{
			PreCheck:          func() { testAccPreCheck(t) },
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config:      fmt.Sprintf(testDataSourceParameterValue, tt.parameter, tt.value),
					ExpectError: regexp.MustCompile(tt.errorRegex),
				},
			},
		})
	}
}

const testDataSourceConfigBasic = `
data "extip" "http_test" {
  resolver = "%s/meta_%s.txt"
}
output "ipaddress" {
  value = data.extip.http_test.ipaddress
}
`

var mockedtestserrors = []struct {
	path       string
	errorRegex string
}{
	{"404", "HTTP request error. Response code: 404"},
	{"timeout", "context deadline exceeded"},
	{"hijack", "transport connection broken|unexpected EOF"},
	{"body_error", "unexpected EOF"},
}

func TestMockedResponsesErrors(t *testing.T) {

	for _, tt := range mockedtestserrors {
		var TestHTTPMock *httptest.Server
		if tt.path == "body_error" {
			// For some reason I cant get this to work as a specific path, so creating a different server for it
			TestHTTPMock = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "1")
			}))
			defer TestHTTPMock.Close()
		} else {
			TestHTTPMock = setUpMockHTTPServer()
			defer TestHTTPMock.Close()
		}
		resource.UnitTest(t, resource.TestCase{
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config:      fmt.Sprintf(testDataSourceConfigBasic, TestHTTPMock.URL, tt.path),
					ExpectError: regexp.MustCompile(tt.errorRegex),
				},
			},
		})
	}
}

func setUpMockHTTPServer() *httptest.Server {
	Server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			w.Header().Set("Content-Type", "text/plain")
			if r.URL.Path == "/meta_200.txt" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("127.0.0.1"))
			} else if r.URL.Path == "/meta_404.txt" {
				w.WriteHeader(http.StatusNotFound)
			} else if r.URL.Path == "/meta_hijack.txt" {
				w.WriteHeader(100)
				w.Write([]byte("Hello3"))
				hj, _ := w.(http.Hijacker)
				conn, _, _ := hj.Hijack()
				conn.Close()
			} else if r.URL.Path == "/meta_timeout.txt" {
				time.Sleep(2000 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("127.0.0.1"))
			} else if r.URL.Path == "/meta_non_ip.txt" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("HELLO!"))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}),
	)

	return Server
}
