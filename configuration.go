package main

import (
	"fmt"
	. "github.com/jayshrivastava/buoy/client"
	. "github.com/jayshrivastava/buoy/node"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type yml struct {
	Cluster struct {
		Nodes []struct {
			Internal string `yaml:"internal"`
			External string `yaml:"external"`
			Host     string `yaml:"host"`
		} `yaml:"nodes"`
	} `yaml:"cluster"`
}

type configuration struct {
	node   []NodeConfig
	client ClientConfig
}

func (c *yml) parse() error {

	yamlFile, err := ioutil.ReadFile("configuration.yml")
	if err != nil {
		return fmt.Errorf("Could not read config file")
	}
	err = yaml.Unmarshal(yamlFile, c)
	if err != nil {
		return fmt.Errorf("Could not parse config file")
	}

	return nil
}

func CreateConfig() (*configuration, error) {
	yml := yml{}
	err := yml.parse()
	if err != nil {
		return nil, err
	}

	nodeConfigs := []NodeConfig{}
	clientConfig := ClientConfig{}
	clientConfig.NodeHosts = map[int32]string{}
	for id, node := range yml.Cluster.Nodes {
		nc := NodeConfig{}
		nc.Id = int32(id)
		nc.InternalOutbound = node.Internal
		nc.ExternalOutbound = node.External
		nc.PeerNodeHosts = map[int32]string{}
		for id2, node2 := range yml.Cluster.Nodes {
			if id2 != id {
				nc.PeerNodeHosts[int32(id2)] = fmt.Sprintf("%s:%s", node2.Host, node2.Internal)
			}
		}
		nodeConfigs = append(nodeConfigs, nc)

		clientConfig.NodeHosts[int32(id)] = fmt.Sprintf("%s:%s", node.Host, node.External)
	}

	return &configuration{node: nodeConfigs, client: clientConfig}, nil
}
