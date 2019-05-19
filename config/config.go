package config

import (
	"io/ioutil"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

type EnvironmentConfig struct {
	BitbucketUsername string `envconfig:"bitbucket_username"`
	BitbucketPassword string `envconfig:"bitbucket_password"`
	GithubToken       string `envconfig:"github_token"`
	SlackToken        string `envconfig:"slack_token"	`
}

type GlobalConfig struct {
	Teams []*TeamConfig
}

func ReadConfig(configFilePath string) (*GlobalConfig, error) {
	config := &GlobalConfig{}
	yamlFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(yamlFile, &config); err != nil {
		return nil, err
	}

	envConfig := &EnvironmentConfig{}
	if err = envconfig.Process("prr", envConfig); err != nil {
		return nil, err
	}

	for _, team := range config.Teams {
		team.SetEnvironmentConfig(envConfig)
	}

	return config, nil
}
