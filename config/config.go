package config

import (
	"fmt"
	"io/ioutil"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

type EnvironmentConfig struct {
	ConfigFilePath string `envconfig:"config"`

	BitbucketUsername string `envconfig:"bitbucket_username"`
	BitbucketPassword string `envconfig:"bitbucket_password"`
	GithubToken       string `envconfig:"github_token"`
	SlackToken        string `envconfig:"slack_token"	`
}

type GlobalConfig struct {
	Teams []*TeamConfig
}

func ReadConfig() (*GlobalConfig, error) {
	config := &GlobalConfig{}
	envConfig := &EnvironmentConfig{}
	if err := envconfig.Process("prr", envConfig); err != nil {
		return nil, err
	}
	if envConfig.ConfigFilePath == "" {
		envConfig.ConfigFilePath = ".prr-config"
	}

	yamlFile, err := ioutil.ReadFile(envConfig.ConfigFilePath)
	if err != nil {
		return nil, fmt.Errorf("Unable to read the config file: %v", err)
	}
	if err = yaml.Unmarshal(yamlFile, &config); err != nil {
		return nil, err
	}

	for _, team := range config.Teams {
		team.SetEnvironmentConfig(envConfig)
	}

	return config, nil
}
