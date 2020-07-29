package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type client struct {
	c http.Client
}

func (this client) dispatch(endpoint string, payload string) error {
	var body struct {
		Text string `json:"text"`
	}
	body.Text = payload

	v, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal body of %s", payload)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(v))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := this.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("response has status %q and body %q", resp.Status, string(rb))
	}
	return nil
}

func (this client) do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error
	backoff := 100 * time.Millisecond
	maxRetries := 3

	for retries := 0; retries < maxRetries; retries++ {
		resp, err = this.c.Do(req)
		if err == nil {
			break
		}
		time.Sleep(backoff)
		backoff *= 2
	}
	return resp, err
}
