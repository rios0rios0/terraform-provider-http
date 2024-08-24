package main

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"io/ioutil"
	"net/http"
)

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return &schema.Provider{
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
			"url": {
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
	url := d.Get("url").(string)
	method := d.Get("method").(string)
	headers := d.Get("headers").(map[string]interface{})

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}

	for k, v := range headers {
		req.Header.Set(k, v.(string))
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	d.Set("response_body", string(body))
	d.Set("response_code", resp.StatusCode)
	d.SetId(url)

	return nil
}

func resourceHTTPRequestDelete(d *schema.ResourceData, m interface{}) error {
	d.SetId("")
	return nil
}
