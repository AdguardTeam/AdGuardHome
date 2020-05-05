package home

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/AdguardTeam/AdGuardHome/util"
	"github.com/AdguardTeam/golibs/log"
)

// Webhook objectd
type Webhook struct {
	URL           string   `yaml:"url"`
	Authorization string   `yaml:"authorization"`
	Events        []string `yaml:"categories"`
}

type webhookPayload struct {
	Events []string `json:"events"`
}

func webhookHandleEvent(e string) {
	client := &http.Client{
		Timeout: time.Second,
	}
	for _, v := range config.Webhooks {
		if util.ContainsString(v.Events, e) < 0 {
			// this hook does not subscribe to this event
			continue
		}
		body := &webhookPayload{
			Events: []string{e},
		}
		buf := new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			log.Debug("Failed to marshal Webhook payload")
			continue
		}

		req, err := http.NewRequest("POST", v.URL, buf)
		if err != nil {
			log.Debug("Failed to notify %s of event '%s'", v.URL, e)
			continue
		}
		req.Header.Add("Content-Type", "application/json")
		if v.Authorization != "" {
			req.Header.Add("Authorization", v.Authorization)
		}
		resp, err := client.Do(req)
		if err == nil {
			log.Debug("Notified %s of event '%s' and got response '%s'", v.URL, e, resp.Status)
		}
		resp.Body.Close()
	}
}
