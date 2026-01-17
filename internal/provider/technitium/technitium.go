package technitium

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Tarow/dockdns/internal/constants"
	"github.com/Tarow/dockdns/internal/dns"
)

type TechnitiumProvider struct {
	apiURL        string
	username      string
	password      string
	zone          string
	token         string
	isApiToken    bool // true if using pre-created API token, false if using session token from login
	skipTLSVerify bool // true to skip TLS certificate verification
	tokenMu       sync.RWMutex
	client        *http.Client
}

type loginResponse struct {
	Status string `json:"status"`
	Token  string `json:"token"`
}

type apiResponse struct {
	Status       string          `json:"status"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	Response     json.RawMessage `json:"response,omitempty"`
}

type recordsResponse struct {
	Zone    zoneInfo `json:"zone"`
	Records []record `json:"records"`
}

type zoneInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type record struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	TTL      int    `json:"ttl"`
	RData    rData  `json:"rData"`
	Disabled bool   `json:"disabled"`
	Comments string `json:"comments,omitempty"`
}

type rData struct {
	IPAddress string `json:"ipAddress,omitempty"` // For A and AAAA records
	Value     string `json:"value,omitempty"`     // For CNAME records
	CName     string `json:"cname,omitempty"`     // Alternative CNAME field
}

// generateDockDNSComment creates a standardized comment for Technitium DNS records
// that identifies the record as managed by DockDNS and provides context about its origin.
func generateDockDNSComment(record dns.Record) string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Build the comment with available metadata
	parts := []string{
		"[DockDNS]",
		fmt.Sprintf("host=%s", hostname),
		fmt.Sprintf("updated=%s", timestamp),
	}

	// Add source-specific information
	if record.Source == "docker" {
		parts = append(parts, "source=docker")
		if record.ContainerID != "" {
			parts = append(parts, fmt.Sprintf("container_id=%s", record.ContainerID))
		}
		if record.ContainerName != "" {
			parts = append(parts, fmt.Sprintf("container_name=%s", record.ContainerName))
		}
	} else if record.Source == "static" {
		parts = append(parts, "source=static")
	} else {
		parts = append(parts, "source=dockdns")
	}

	return strings.Join(parts, " | ")
}

// New creates a new TechnitiumProvider.
// If apiToken is provided, it will be used directly for authentication.
// Otherwise, username and password will be used to obtain a session token.
// If skipTLSVerify is true, TLS certificate verification will be skipped.
func New(apiURL, username, password, apiToken, zone string, skipTLSVerify bool) (*TechnitiumProvider, error) {
	if apiURL == "" || zone == "" {
		return nil, fmt.Errorf("technitium provider requires apiURL and zone to be set")
	}

	// Ensure apiURL doesn't have trailing slash
	apiURL = strings.TrimSuffix(apiURL, "/")

	// Create HTTP client with optional TLS skip verify
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}
	if skipTLSVerify {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		slog.Warn("Technitium DNS TLS certificate verification disabled", "zone", zone)
	}

	provider := &TechnitiumProvider{
		apiURL:        apiURL,
		username:      username,
		password:      password,
		zone:          zone,
		skipTLSVerify: skipTLSVerify,
		client:        httpClient,
	}

	// If API token is provided, use it directly
	if apiToken != "" {
		provider.token = apiToken
		provider.isApiToken = true
		slog.Debug("Technitium DNS using API token authentication", "zone", zone)
		return provider, nil
	}

	// Otherwise, require username and password for login
	if username == "" || password == "" {
		return nil, fmt.Errorf("technitium provider requires either apiToken, or username and password to be set")
	}

	// Login and get token
	if err := provider.login(); err != nil {
		return nil, fmt.Errorf("failed to login to Technitium DNS: %w", err)
	}

	return provider, nil
}

func (p *TechnitiumProvider) login() error {
	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()

	loginURL := fmt.Sprintf("%s/api/user/login", p.apiURL)
	data := url.Values{}
	data.Set("user", p.username)
	data.Set("pass", p.password)

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send login request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read login response: %w", err)
	}

	var loginResp loginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	if loginResp.Status != "ok" {
		return fmt.Errorf("login failed: %s", string(body))
	}

	p.token = loginResp.Token
	slog.Debug("Technitium DNS login successful", "zone", p.zone)
	return nil
}

func (p *TechnitiumProvider) getToken() string {
	p.tokenMu.RLock()
	defer p.tokenMu.RUnlock()
	return p.token
}

func (p *TechnitiumProvider) doRequest(method, endpoint string, data url.Values) ([]byte, error) {
	return p.doRequestWithRetry(method, endpoint, data, 0)
}

func (p *TechnitiumProvider) doRequestWithRetry(method, endpoint string, data url.Values, retryCount int) ([]byte, error) {
	const maxRetries = 1 // Only retry once for invalid token

	token := p.getToken()
	if token == "" {
		// API token users should already have a token set
		if p.isApiToken {
			return nil, fmt.Errorf("API token is not set")
		}
		if err := p.login(); err != nil {
			return nil, err
		}
		token = p.getToken()
	}

	// Add token to data
	if data == nil {
		data = url.Values{}
	}
	data.Set("token", token)

	reqURL := fmt.Sprintf("%s%s", p.apiURL, endpoint)

	var req *http.Request
	var err error

	if method == "GET" {
		reqURL = fmt.Sprintf("%s?%s", reqURL, data.Encode())
		req, err = http.NewRequest("GET", reqURL, nil)
	} else {
		req, err = http.NewRequest("POST", reqURL, strings.NewReader(data.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for invalid token and retry login
	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err == nil {
		if apiResp.Status == "invalid-token" {
			// If using API token, we can't retry login
			if p.isApiToken {
				return nil, fmt.Errorf("API token is invalid or expired")
			}
			if retryCount >= maxRetries {
				return nil, fmt.Errorf("authentication failed after %d retries: invalid token", maxRetries)
			}
			slog.Debug("Token expired, re-logging in", "zone", p.zone, "retry", retryCount+1)
			if err := p.login(); err != nil {
				return nil, fmt.Errorf("re-login failed: %w", err)
			}
			// Retry request with new token
			return p.doRequestWithRetry(method, endpoint, data, retryCount+1)
		}
		if apiResp.Status == "error" {
			return nil, fmt.Errorf("API error: %s", apiResp.ErrorMessage)
		}
	}

	return body, nil
}

func (p *TechnitiumProvider) List() ([]dns.Record, error) {
	data := url.Values{}
	data.Set("domain", p.zone)
	data.Set("zone", p.zone)
	data.Set("listZone", "true")

	body, err := p.doRequest("GET", "/api/zones/records/get", data)
	if err != nil {
		return nil, fmt.Errorf("failed to list records: %w", err)
	}

	var apiResp struct {
		Status   string          `json:"status"`
		Response recordsResponse `json:"response"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse records response: %w", err)
	}

	if apiResp.Status != "ok" {
		return nil, fmt.Errorf("list records failed: %s", string(body))
	}

	var records []dns.Record
	for _, rec := range apiResp.Response.Records {
		// Skip disabled records
		if rec.Disabled {
			continue
		}

		// Only handle A, AAAA, and CNAME records
		if rec.Type != constants.RecordTypeA &&
			rec.Type != constants.RecordTypeAAAA &&
			rec.Type != constants.RecordTypeCNAME {
			continue
		}

		content := ""
		switch rec.Type {
		case constants.RecordTypeA, constants.RecordTypeAAAA:
			content = rec.RData.IPAddress
		case constants.RecordTypeCNAME:
			if rec.RData.CName != "" {
				content = rec.RData.CName
			} else {
				content = rec.RData.Value
			}
		}

		records = append(records, dns.Record{
			ID:      fmt.Sprintf("%s:%s:%s", rec.Name, rec.Type, content),
			Name:    rec.Name,
			Type:    rec.Type,
			Content: content,
			TTL:     rec.TTL,
			Comment: rec.Comments,
			Proxied: false, // Technitium doesn't have proxy feature
		})
	}

	return records, nil
}

func (p *TechnitiumProvider) Get(name string, recordType string) (dns.Record, error) {
	data := url.Values{}
	data.Set("domain", name)
	data.Set("zone", p.zone)

	body, err := p.doRequest("GET", "/api/zones/records/get", data)
	if err != nil {
		return dns.Record{}, fmt.Errorf("failed to get record: %w", err)
	}

	var apiResp struct {
		Status   string          `json:"status"`
		Response recordsResponse `json:"response"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return dns.Record{}, fmt.Errorf("failed to parse get response: %w", err)
	}

	if apiResp.Status != "ok" {
		return dns.Record{}, fmt.Errorf("get record failed: %s", string(body))
	}

	for _, rec := range apiResp.Response.Records {
		if rec.Name == name && rec.Type == recordType && !rec.Disabled {
			content := ""
			switch rec.Type {
			case constants.RecordTypeA, constants.RecordTypeAAAA:
				content = rec.RData.IPAddress
			case constants.RecordTypeCNAME:
				if rec.RData.CName != "" {
					content = rec.RData.CName
				} else {
					content = rec.RData.Value
				}
			}

			return dns.Record{
				ID:      fmt.Sprintf("%s:%s:%s", rec.Name, rec.Type, content),
				Name:    rec.Name,
				Type:    rec.Type,
				Content: content,
				TTL:     rec.TTL,
				Comment: rec.Comments,
				Proxied: false,
			}, nil
		}
	}

	return dns.Record{}, nil
}

func (p *TechnitiumProvider) Create(record dns.Record) (dns.Record, error) {
	data := url.Values{}
	data.Set("domain", record.Name)
	data.Set("zone", p.zone)
	data.Set("type", record.Type)
	data.Set("ttl", fmt.Sprintf("%d", record.TTL))

	// Generate DockDNS comment for Technitium to identify managed records
	comment := generateDockDNSComment(record)
	data.Set("comments", comment)
	record.Comment = comment

	switch record.Type {
	case constants.RecordTypeA, constants.RecordTypeAAAA:
		data.Set("ipAddress", record.Content)
	case constants.RecordTypeCNAME:
		data.Set("cname", record.Content)
	default:
		return dns.Record{}, fmt.Errorf("unsupported record type: %s", record.Type)
	}

	body, err := p.doRequest("POST", "/api/zones/records/add", data)
	if err != nil {
		return dns.Record{}, fmt.Errorf("failed to create record: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return dns.Record{}, fmt.Errorf("failed to parse create response: %w", err)
	}

	if apiResp.Status != "ok" {
		return dns.Record{}, fmt.Errorf("create record failed: %s", apiResp.ErrorMessage)
	}

	// Generate an ID for the created record
	record.ID = fmt.Sprintf("%s:%s:%s", record.Name, record.Type, record.Content)

	slog.Debug("Created record in Technitium DNS",
		"name", record.Name,
		"type", record.Type,
		"content", record.Content)

	return record, nil
}

func (p *TechnitiumProvider) Update(record dns.Record) (dns.Record, error) {
	// For Technitium, we need to parse the old values from the ID
	// ID format: name:type:oldContent
	parts := strings.Split(record.ID, ":")
	if len(parts) < 3 {
		// If no ID or invalid ID, try to find existing record
		existing, err := p.Get(record.Name, record.Type)
		if err != nil || existing.ID == "" {
			// Record doesn't exist, create it instead
			return p.Create(record)
		}
		parts = strings.Split(existing.ID, ":")
	}

	oldContent := parts[2]

	data := url.Values{}
	data.Set("domain", record.Name)
	data.Set("zone", p.zone)
	data.Set("type", record.Type)
	data.Set("ttl", fmt.Sprintf("%d", record.TTL))

	// Generate DockDNS comment for Technitium to identify managed records
	comment := generateDockDNSComment(record)
	data.Set("comments", comment)
	record.Comment = comment

	switch record.Type {
	case constants.RecordTypeA, constants.RecordTypeAAAA:
		data.Set("ipAddress", oldContent)
		data.Set("newIpAddress", record.Content)
	case constants.RecordTypeCNAME:
		// For CNAME, Technitium requires delete + add
		if err := p.Delete(dns.Record{
			ID:      record.ID,
			Name:    record.Name,
			Type:    record.Type,
			Content: oldContent,
		}); err != nil {
			return dns.Record{}, fmt.Errorf("failed to delete old CNAME: %w", err)
		}
		return p.Create(record)
	default:
		return dns.Record{}, fmt.Errorf("unsupported record type: %s", record.Type)
	}

	body, err := p.doRequest("POST", "/api/zones/records/update", data)
	if err != nil {
		return dns.Record{}, fmt.Errorf("failed to update record: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return dns.Record{}, fmt.Errorf("failed to parse update response: %w", err)
	}

	if apiResp.Status != "ok" {
		return dns.Record{}, fmt.Errorf("update record failed: %s", apiResp.ErrorMessage)
	}

	// Update the ID with new content
	record.ID = fmt.Sprintf("%s:%s:%s", record.Name, record.Type, record.Content)

	slog.Debug("Updated record in Technitium DNS",
		"name", record.Name,
		"type", record.Type,
		"content", record.Content)

	return record, nil
}

func (p *TechnitiumProvider) Delete(record dns.Record) error {
	data := url.Values{}
	data.Set("domain", record.Name)
	data.Set("zone", p.zone)
	data.Set("type", record.Type)

	switch record.Type {
	case constants.RecordTypeA, constants.RecordTypeAAAA:
		data.Set("ipAddress", record.Content)
	case constants.RecordTypeCNAME:
		// For CNAME deletion, we don't need to specify the value
		// The API will delete all CNAME records for this domain
	default:
		return fmt.Errorf("unsupported record type: %s", record.Type)
	}

	body, err := p.doRequest("POST", "/api/zones/records/delete", data)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("failed to parse delete response: %w", err)
	}

	if apiResp.Status != "ok" {
		return fmt.Errorf("delete record failed: %s", apiResp.ErrorMessage)
	}

	slog.Debug("Deleted record from Technitium DNS",
		"name", record.Name,
		"type", record.Type,
		"content", record.Content)

	return nil
}
