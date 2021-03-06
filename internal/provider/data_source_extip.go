package provider

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceExtip() *schema.Resource {
	return &schema.Resource{
		// This description is used by the documentation generator and the language server.
		Description: "The `extip` data source returns an IP address from an external resolver.",

		ReadContext: dataSourceExtipRead,

		Schema: map[string]*schema.Schema{
			"ipaddress": {
				Type:     schema.TypeString,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"resolver": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "https://checkip.amazonaws.com/",
				Description: "The URL to use to resolve the external IP address\nIf not set, defaults to https://checkip.amazonaws.com/",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
			"client_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1000,
				Description: "The time to wait for a response in ms\nIf not set, defaults to 1000 (1 second). Setting to 0 means infinite (no timeout)",
				Elem: &schema.Schema{
					Type: schema.TypeInt,
				},
			},
			"validate_ip": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Validate if the returned response is a valid ip address",
				Elem: &schema.Schema{
					Type: schema.TypeBool,
				},
			},
		},
	}
}

func getExternalIPFrom(service string, clientTimeout int) (string, error) {

	var netClient = &http.Client{
		Timeout: time.Duration(clientTimeout) * time.Millisecond,
	}

	rsp, err := netClient.Get(service)
	if err != nil {
		return "", err
	}

	defer rsp.Body.Close()

	if rsp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP request error. Response code: %d", rsp.StatusCode)
	}

	buf, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return "", err
	}

	return string(bytes.TrimSpace(buf)), nil
}

func dataSourceExtipRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	resolver := d.Get("resolver").(string)

	clientTimeout := d.Get("client_timeout").(int)

	ip, err := getExternalIPFrom(resolver, clientTimeout)

	if v, ok := d.GetOkExists("validate_ip"); ok {
		if v.(bool) {
			ipParse := net.ParseIP(ip)
			if ipParse == nil {
				return diag.FromErr(err)
			}
		}
	}

	if err == nil {
		d.Set("ipaddress", string(ip))
		d.SetId(time.Now().UTC().String())
	} else {
		return diag.FromErr(err)
	}

	return nil
}
