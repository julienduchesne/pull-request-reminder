package config

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

const (
	ConfigFileName = ".prr-config"
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

func ReadConfig() (config *GlobalConfig, err error) {
	envConfig := &EnvironmentConfig{}
	if err = envconfig.Process("prr", envConfig); err != nil {
		return
	}

	if envConfig.ConfigFilePath == "" {
		envConfig.ConfigFilePath = ConfigFileName
	}

	if strings.HasPrefix(envConfig.ConfigFilePath, "s3://") {
		config, err = readS3Config(envConfig.ConfigFilePath)
	} else {
		config, err = readFileConfig(envConfig.ConfigFilePath)
	}

	if err != nil {
		return nil, err
	}

	for _, team := range config.Teams {
		team.SetEnvironmentConfig(envConfig)
	}

	return
}

func readS3Config(s3Path string) (*GlobalConfig, error) {
	splitS3Path, err := url.Parse(s3Path)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse the given S3 config path: %v", err)
	}

	tempdir := os.TempDir()
	configFileName := path.Join(tempdir, ConfigFileName)
	defer os.Remove(tempdir)

	configFile, err := os.Create(configFileName)
	if err != nil {
		return nil, fmt.Errorf("Failed to create a temporary config file %q, %v", configFileName, err)
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	downloader := s3manager.NewDownloader(sess)

	if _, err = downloader.Download(configFile, &s3.GetObjectInput{
		Bucket: aws.String(splitS3Path.Host),
		Key:    aws.String(splitS3Path.Path),
	}); err != nil {
		return nil, fmt.Errorf("Failed to download the config file from S3, %v", err)
	}

	return readFileConfig(configFileName)
}

func readFileConfig(configFilePath string) (*GlobalConfig, error) {
	config := &GlobalConfig{}
	yamlFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("Unable to read the config file: %v", err)
	}
	if err = yaml.Unmarshal(yamlFile, &config); err != nil {
		return nil, err
	}
	return config, nil
}
