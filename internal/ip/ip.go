package ip

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
)

func GetPublicIP4Address() (string, error) {
	return getPublicAddress(true)
}

func GetPublicIP6Address() (string, error) {
	return getPublicAddress(false)
}

func getPublicAddress(ip4 bool) (string, error) {
	var proto string
	if ip4 {
		proto = "tcp4"
	} else {
		proto = "tcp6"
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, proto, addr)
			},
		},
	}
	resp, err := client.Get("https://www.cloudflare.com/cdn-cgi/trace")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return extractIPAddress(string(body))
}

func extractIPAddress(response string) (string, error) {
	// Regular expression to match the IP address
	ipRegex := regexp.MustCompile(`\bip=(.*)\b`)

	// Find the first match in the response
	match := ipRegex.FindStringSubmatch(response)
	if len(match) > 1 {
		return match[1], nil
	}

	return "", fmt.Errorf("could not extract ip address from content: %s", response)
}
