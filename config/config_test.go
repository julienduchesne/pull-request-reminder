package config

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

const (
	testGlobalConfig = `{
    "teams": [
        {
            "name": "test_team",
            "bitbucket": {
                "repositories": [
                    "test_repo"
                ]
            },
            "github": {
                "repositories": [
                    "test_repo"
                ]
            },
            "slack": {
                "channel": "#my_channel"
            },
            "users": [
                {
                    "name": "John Doe",
                    "github_username": "jdoe",
                    "bitbucket_username": "jdoe",
                    "slack_username": "jdoe"
                }
            ]
        }
    ]
}`
	s3Path   = "s3://bucket-name/path/to/config"
	s3Bucket = "bucket-name"
	s3Key    = "/path/to/config"
)

func TestReadingConfigSetsEnvironmentVariables(t *testing.T) {
	t.Parallel()

	envConfig := getTestEnvConfig("")
	configReader := &ConfigReader{
		envConfig: envConfig,
		readFunc: func(string) (*GlobalConfig, error) {
			config := &GlobalConfig{}
			yaml.Unmarshal([]byte(testGlobalConfig), config)
			return config, nil
		},
	}

	config, _ := configReader.ReadConfig()
	team := config.Teams[0]
	assert.Equal(t, envConfig.BitbucketUsername, team.Bitbucket.Username)
	assert.Equal(t, envConfig.BitbucketPassword, team.Bitbucket.Password)
	assert.Equal(t, envConfig.GithubToken, team.Github.Token)
	assert.Equal(t, envConfig.SlackToken, team.Slack.Token)
}

func TestReadFileConfig(t *testing.T) {
	tempdir := os.TempDir()
	configFileName := path.Join(tempdir, defaultConfigFileName)
	defer os.Remove(tempdir)

	ioutil.WriteFile(configFileName, []byte(testGlobalConfig), 0644)

	configReader := &ConfigReader{
		envConfig: getTestEnvConfig(configFileName),
		readFunc:  readFileConfig,
	}
	config, err := configReader.ReadConfig()
	assert.Len(t, config.Teams, 1)
	assert.Nil(t, err)
}

func TestReadS3Config(t *testing.T) {
	configReader := &ConfigReader{
		envConfig: getTestEnvConfig(s3Path),
		readFunc:  getS3ConfigReadFunc(&mockedS3Client{t: t}),
	}
	config, err := configReader.ReadConfig()
	assert.Len(t, config.Teams, 1)
	assert.Nil(t, err)
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
	return &s3.GetObjectOutput{Body: ioutil.NopCloser(strings.NewReader(testGlobalConfig))}, nil
}
