package ip

import (
	"fmt"
	"os/exec"
	"regexp"
)

func GetPublicIP4Address() (string, error) {
	return getPublicAddress(true)
}

func GetPublicIP6Address() (string, error) {
	return getPublicAddress(false)
}

func getPublicAddress(ip4 bool) (string, error) {
	var param string
	if ip4 {
		param = "-4"
	} else {
		param = "-6"
	}

	cmd := exec.Command("curl", param, "https://www.cloudflare.com/cdn-cgi/trace")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Convert the output to a string
	outputStr := string(output)

	// Extract the IP address using a regular expression
	ip := extractIPAddress(outputStr)
	if ip == "" {
		return "", fmt.Errorf("IP address not found in the response")
	}

	return ip, nil
}

func extractIPAddress(response string) string {
	// Regular expression to match the IP address
	ipRegex := regexp.MustCompile(`\bip=(.*)\b`)

	// Find the first match in the response
	match := ipRegex.FindStringSubmatch(response)
	if len(match) == 2 {
		return match[1]
	}

	return ""
}
