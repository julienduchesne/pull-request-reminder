package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"runtime"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

const (
	s3Path   = "s3://bucket-name/path/to/config"
	s3Bucket = "bucket-name"
	s3Key    = "/path/to/config"
)

var testGlobalConfig, err = ioutil.ReadFile("./test_config.json")

func TestReadingConfigSetsEnvironmentVariables(t *testing.T) {
	t.Parallel()

	envConfig := getTestEnvConfig("")
	configReader := &Reader{
		envConfig: envConfig,
		readFunc: func(string) (*GlobalConfig, error) {
			config := &GlobalConfig{}
			yaml.Unmarshal(testGlobalConfig, config)
			return config, nil
		},
	}

	config, _ := configReader.ReadConfig()
	team := config.Teams[0]
	assert.Equal(t, envConfig.BitbucketUsername, team.Hosts.Bitbucket.Username)
	assert.Equal(t, envConfig.BitbucketPassword, team.Hosts.Bitbucket.Password)
	assert.Equal(t, envConfig.GithubToken, team.Hosts.Github.Token)
	assert.Equal(t, envConfig.SlackToken, team.Messaging.Slack.Token)
}

func TestReadFileConfig(t *testing.T) {
	tempdir := os.TempDir()
	configFileName := path.Join(tempdir, defaultConfigFileName)
	defer os.Remove(tempdir)

	ioutil.WriteFile(configFileName, testGlobalConfig, 0644)

	configReader := &Reader{
		envConfig: getTestEnvConfig(configFileName),
		readFunc:  readFileConfig,
	}
	config, err := configReader.ReadConfig()
	assert.Equal(t, log.InfoLevel, log.GetLevel()) // Default level is info
	assert.Len(t, config.Teams, 1)
	assert.Nil(t, err)
}

func TestReadS3Config(t *testing.T) {
	configReader := &Reader{
		envConfig: getTestEnvConfig(s3Path),
		readFunc:  getS3ConfigReadFunc(&mockedS3Client{t: t}),
	}
	configReader.envConfig.LogLevel = "DEBUG"
	config, err := configReader.ReadConfig()
	assert.Equal(t, log.DebugLevel, log.GetLevel()) // Log level was set to default
	assert.Len(t, config.Teams, 1)
	assert.Nil(t, err)
}

func TestCreateConfigReader(t *testing.T) {
	configReader, err := NewReader()
	assert.Nil(t, err)
	assert.Equal(t, defaultConfigFileName, configReader.envConfig.ConfigFilePath)
	expectedFunc := runtime.FuncForPC(reflect.ValueOf(readFileConfig).Pointer()).Name()
	gottenFunc := runtime.FuncForPC(reflect.ValueOf(configReader.readFunc).Pointer()).Name()
	assert.Equal(t, expectedFunc, gottenFunc)

	for key, value := range map[string]string{
		"PRR_BITBUCKET_PASSWORD": "bb_pass",
		"PRR_BITBUCKET_USERNAME": "bb_user",
		"PRR_GITHUB_TOKEN":       "gh_token",
		"PRR_SLACK_TOKEN":        "xoxb_test",
		"PRR_CONFIG":             "s3://bucket/key",
		"PRR_LOG_LEVEL":          "DEBUG",
	} {
		oldValue := os.Getenv(key)
		if oldValue != "" {
			defer os.Setenv(key, oldValue)
		} else {
			defer os.Unsetenv(key)
		}
		os.Setenv(key, value)
	}
	configReader, err = NewReader()
	assert.Nil(t, err)
	assert.Equal(t, "DEBUG", configReader.envConfig.LogLevel)
	assert.Equal(t, "bb_pass", configReader.envConfig.BitbucketPassword)
	assert.Equal(t, "bb_user", configReader.envConfig.BitbucketUsername)
	assert.Equal(t, "gh_token", configReader.envConfig.GithubToken)
	assert.Equal(t, "xoxb_test", configReader.envConfig.SlackToken)
	expectedFunc = runtime.FuncForPC(reflect.ValueOf(getS3ConfigReadFunc(nil)).Pointer()).Name()
	gottenFunc = runtime.FuncForPC(reflect.ValueOf(configReader.readFunc).Pointer()).Name()
	assert.Equal(t, expectedFunc, gottenFunc)
}

func getTestEnvConfig(path string) *EnvironmentConfig {
	return &EnvironmentConfig{
		ConfigFilePath:    path,
		BitbucketUsername: "BB_USER",
		BitbucketPassword: "BB_PASSWORD",
		GithubToken:       "GH_TOKEN",
		SlackToken:        "xoxb-stuff",
	}
}

type mockedS3Client struct {
	s3iface.S3API
	t *testing.T
}

func (mock *mockedS3Client) GetObject(input *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	assert.Equal(mock.t, s3Bucket, *input.Bucket)
	assert.Equal(mock.t, s3Key, *input.Key)
	return &s3.GetObjectOutput{Body: ioutil.NopCloser(bytes.NewReader(testGlobalConfig))}, nil
}
