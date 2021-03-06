package filter

import (
	"fmt"
	"strings"

	"github.com/docker/swarm/cluster"
	"github.com/samalba/dockerclient"
)

// DependencyFilter co-schedules dependent containers on the same node.
type DependencyFilter struct {
}

// Filter is exported
func (f *DependencyFilter) Filter(config *dockerclient.ContainerConfig, nodes []cluster.Node) ([]cluster.Node, error) {
	if len(nodes) == 0 {
		return nodes, nil
	}

	// Extract containers from links.
	links := []string{}
	for _, link := range config.HostConfig.Links {
		links = append(links, strings.SplitN(link, ":", 2)[0])
	}

	// Check if --net points to a container.
	net := []string{}
	if strings.HasPrefix(config.HostConfig.NetworkMode, "container:") {
		net = append(net, strings.TrimPrefix(config.HostConfig.NetworkMode, "container:"))
	}

	candidates := []cluster.Node{}
	for _, node := range nodes {
		if f.check(config.HostConfig.VolumesFrom, node) &&
			f.check(links, node) &&
			f.check(net, node) {
			candidates = append(candidates, node)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("Unable to find a node fulfilling all dependencies: %s", f.String(config))
	}

	return candidates, nil
}

// Get a string representation of the dependencies found in the container config.
func (f *DependencyFilter) String(config *dockerclient.ContainerConfig) string {
	dependencies := []string{}
	for _, volume := range config.HostConfig.VolumesFrom {
		dependencies = append(dependencies, fmt.Sprintf("--volumes-from=%s", volume))
	}
	for _, link := range config.HostConfig.Links {
		dependencies = append(dependencies, fmt.Sprintf("--link=%s", link))
	}
	if strings.HasPrefix(config.HostConfig.NetworkMode, "container:") {
		dependencies = append(dependencies, fmt.Sprintf("--net=%s", config.HostConfig.NetworkMode))
	}
	return strings.Join(dependencies, " ")
}

// Ensure that the node contains all dependent containers.
func (f *DependencyFilter) check(dependencies []string, node cluster.Node) bool {
	for _, dependency := range dependencies {
		if node.Container(dependency) == nil {
			return false
		}
	}
	return true
}
