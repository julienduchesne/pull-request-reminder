package config

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

const defaultConfigFileName = ".prr-config"

// EnvironmentConfig represents all configurations that can be set using environment variables
type EnvironmentConfig struct {
	ConfigFilePath string `envconfig:"config"`

	BitbucketUsername string `envconfig:"bitbucket_username"`
	BitbucketPassword string `envconfig:"bitbucket_password"`
	GithubToken       string `envconfig:"github_token"`
	SlackToken        string `envconfig:"slack_token"`
}

// GlobalConfig represents the read configuration file
type GlobalConfig struct {
	Teams []*TeamConfig
}

type ConfigReader struct {
	envConfig *EnvironmentConfig
	readFunc  func(string) (*GlobalConfig, error)
}

func NewConfigReader() (*ConfigReader, error) {
	envConfig := &EnvironmentConfig{}
	if err := envconfig.Process("prr", envConfig); err != nil {
		return nil, err
	}
	if envConfig.ConfigFilePath == "" {
		envConfig.ConfigFilePath = defaultConfigFileName
	}
	configReader := &ConfigReader{envConfig: envConfig}

	if strings.HasPrefix(envConfig.ConfigFilePath, "s3://") {
		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
		configReader.readFunc = getS3ConfigReadFunc(s3.New(sess))
	} else {
		configReader.readFunc = readFileConfig
	}

	return configReader, nil
}

func (configReader *ConfigReader) ReadConfig() (config *GlobalConfig, err error) {
	if config, err = configReader.readFunc(configReader.envConfig.ConfigFilePath); err != nil {
		return nil, err
	}
	for _, team := range config.Teams {
		team.setEnvironmentConfig(configReader.envConfig)
	}
	return
}

func getS3ConfigReadFunc(client s3iface.S3API) func(string) (*GlobalConfig, error) {
	return func(s3Path string) (*GlobalConfig, error) {
		splitS3Path, err := url.Parse(s3Path)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse the given S3 config path: %v", err)
		}

		tempdir := os.TempDir()
		configFileName := path.Join(tempdir, defaultConfigFileName)
		defer os.Remove(tempdir)

		configFile, err := os.Create(configFileName)
		if err != nil {
			return nil, fmt.Errorf("Failed to create a temporary config file %q, %v", configFileName, err)
		}
		defer configFile.Close()

		var resp *s3.GetObjectOutput
		if resp, err = client.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(splitS3Path.Host),
			Key:    aws.String(splitS3Path.Path),
		}); err != nil {
			return nil, fmt.Errorf("Failed to download the config file from S3, %v", err)
		}

		if _, err = io.Copy(configFile, resp.Body); err != nil {
			return nil, fmt.Errorf("Failed to write the config file downloaded from S3, %v", err)
		}

		return readFileConfig(configFileName)
	}
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
