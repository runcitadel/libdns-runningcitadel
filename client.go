package runningcitadel

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/libdns/libdns"
)

func (p *Provider) createRecord(ctx context.Context, record libdns.Record) (cfDNSRecord, error) {
	cfRec, err := cloudflareRecord(record)
	if err != nil {
		return cfDNSRecord{}, err
	}
	jsonBytes, err := json.Marshal(cfRec)
	if err != nil {
		return cfDNSRecord{}, err
	}

	reqURL := fmt.Sprintf("%s/api/dns/create", baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return cfDNSRecord{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	var result cfDNSRecord
	_, err = p.doAPIRequest(req, &result)
	if err != nil {
		return cfDNSRecord{}, err
	}

	return result, nil
}

// updateRecord updates a DNS record. oldRec must have an ID.
// Only the non-empty fields in newRec will be changed.
func (p *Provider) updateRecord(ctx context.Context, oldRec, newRec cfDNSRecord) (cfDNSRecord, error) {
	reqURL := fmt.Sprintf("%s/api/dns/records/%s", baseURL, oldRec.ID)
	jsonBytes, err := json.Marshal(newRec)
	if err != nil {
		return cfDNSRecord{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return cfDNSRecord{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	var result cfDNSRecord
	_, err = p.doAPIRequest(req, &result)
	return result, err
}

type FindOptions struct {
	Match     string `json:"match,omitempty"`
	Name      string `json:"name"`
	Order     string `json:"order,omitempty"`
	PerPage   int    `json:"perPage,omitempty"`
	Content   string `json:"content,omitempty"`
	Type      string `json:"type,omitempty"`
	Direction string `json:"direction,omitempty"`
}

func (p *Provider) getDNSRecords(ctx context.Context, zone string, rec libdns.Record, matchContent bool) ([]cfDNSRecord, error) {
	query := FindOptions{
		Name:    rec.Name,
		Type:    rec.Type,
		PerPage: 100,
	}
	if matchContent {
		query.Content = rec.Value
	}

	reqURL := fmt.Sprintf("%s/api/dns/find", baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	var results []cfDNSRecord
	_, err = p.doAPIRequest(req, &results)
	return results, err
}

// doAPIRequest authenticates the request req and does the round trip. It returns
// the decoded response from Cloudflare if successful; otherwise it returns an
// error including error information from the API if applicable. If result is a
// non-nil pointer, the result field from the API response will be decoded into
// it for convenience.
func (p *Provider) doAPIRequest(req *http.Request, result interface{}) (json.RawMessage, error) {
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(p.Username+":"+p.Password)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return json.RawMessage{}, err
	}
	defer resp.Body.Close()

	// Fail if status is not 2xx
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return json.RawMessage{}, fmt.Errorf("got error status: HTTP %d", resp.StatusCode)
	}

	var respData json.RawMessage
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		return json.RawMessage{}, err
	}

	return respData, err
}

const baseURL = "https://runningcitadel.com"
