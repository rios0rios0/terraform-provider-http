package main

import (
	"bytes"
	"crypto/tls"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"io"
	"net/http"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return &schema.Provider{
				Schema: map[string]*schema.Schema{
					"url": {
						Type:     schema.TypeString,
						Required: true,
					},
					"basic_auth": {
						Type:     schema.TypeList,
						Optional: true,
						MaxItems: 1,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"username": {
									Type:     schema.TypeString,
									Required: true,
								},
								"password": {
									Type:      schema.TypeString,
									Required:  true,
									Sensitive: true,
								},
							},
						},
					},
					"ignore_tls": {
						Type:     schema.TypeBool,
						Optional: true,
						Default:  false,
					},
				},
				ResourcesMap: map[string]*schema.Resource{
					"http_request": resourceHTTPRequest(),
				},
			}
		},
	})
}

func resourceHTTPRequest() *schema.Resource {
	return &schema.Resource{
		Create: resourceHTTPRequestCreate,
		Read:   resourceHTTPRequestRead,
		Update: resourceHTTPRequestUpdate,
		Delete: resourceHTTPRequestDelete,

		Schema: map[string]*schema.Schema{
			"path": {
				Type:     schema.TypeString,
				Required: true,
			},
			"method": {
				Type:     schema.TypeString,
				Required: true,
			},
			"headers": {
				Type:     schema.TypeMap,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"request_body": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"response_body": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"response_code": {
				Type:     schema.TypeInt,
				Computed: true,
			},
		},
	}
}

func resourceHTTPRequestCreate(d *schema.ResourceData, m interface{}) error {
	return resourceHTTPRequestUpdate(d, m)
}

func resourceHTTPRequestRead(d *schema.ResourceData, m interface{}) error {
	// No-op: All data is already in state
	return nil
}

func resourceHTTPRequestUpdate(d *schema.ResourceData, m interface{}) error {
	providerConfig := m.(*schema.ResourceData)
	baseURL := providerConfig.Get("url").(string)
	ignoreTLS := providerConfig.Get("ignore_tls").(bool)

	path := d.Get("path").(string)
	method := d.Get("method").(string)
	headers := d.Get("headers").(map[string]interface{})
	requestBody := d.Get("request_body").(string)

	var req *http.Request
	var err error

	if requestBody != "" {
		req, err = http.NewRequest(method, baseURL+path, bytes.NewBuffer([]byte(requestBody)))
	} else {
		req, err = http.NewRequest(method, baseURL+path, nil)
	}

	if err != nil {
		return err
	}

	for k, v := range headers {
		req.Header.Set(k, v.(string))
	}

	if v, ok := providerConfig.GetOk("basic_auth"); ok {
		auth := v.([]interface{})[0].(map[string]interface{})
		username := auth["username"].(string)
		password := auth["password"].(string)
		req.SetBasicAuth(username, password)
	}

	client := &http.Client{}
	if ignoreTLS {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.Transport = tr
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	d.Set("response_body", string(body))
	d.Set("response_code", resp.StatusCode)
	d.SetId(path)

	return nil
}

func resourceHTTPRequestDelete(d *schema.ResourceData, m interface{}) error {
	d.SetId("")
	return nil
}
