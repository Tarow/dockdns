package docker

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func fetchContainerByLabel(label string) []types.Container {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		panic(err)
	}

	var publicContainers []types.Container

	// Find exposed containers (public-chain middleware applied)
	labelKeyRegexString := "traefik\\.http\\.routers\\..*\\.middlewares"
	labelKeyRegex, err := regexp.Compile(labelKeyRegexString)
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		for key, value := range container.Labels {
			if labelKeyRegex.MatchString(key) && strings.Contains(value, "public-chain") {
				publicContainers = append(publicContainers, container)
			}
		}
	}
	return publicContainers
}

func extractDomainFromLabel(labelValue string) (string, error) {
	// Regular expression to match the domain in the label value
	domainRegex := regexp.MustCompile(os.Getenv("HOST_EXTRACTION_REGEX"))

	// Find the first match in the label value
	match := domainRegex.FindStringSubmatch(labelValue)
	if len(match) == 2 {
		return match[1], nil
	}

	return "", fmt.Errorf("domain not found in the label value")
}

func extractDomainsFromContainers(containers []types.Container) ([]string, error) {
	labelKeyRegexString := `traefik\.http\.routers\..*\.rule`
	labelKeyRegex, err := regexp.Compile(labelKeyRegexString)
	if err != nil {
		panic(err)
	}

	var domains []string

	for _, container := range containers {
		for key, value := range container.Labels {
			matched := labelKeyRegex.MatchString(key)
			if err != nil {
				return nil, err
			}
			if matched {
				domain, err := extractDomainFromLabel(value)
				if err != nil {
					return nil, err
				}
				domains = append(domains, domain)
			}
		}
	}

	return domains, nil
}
