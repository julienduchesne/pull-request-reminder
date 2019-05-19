package config

type TeamConfig struct {
	Name      string
	Bitbucket struct {
		Username     string
		Password     string
		Repositories []string
	}

	Github struct {
		Repositories []string
		Token        string
	}

	Slack struct {
		Channel string
		Token   string
	}

	Users []struct {
		Name              string
		BitbucketUsername string `yaml:"bitbucket_username"`
		GithubUsername    string `yaml:"github_username"`
		SlackUsername     string `yaml:"slack_username"`
	}
}

func (config *TeamConfig) SetEnvironmentConfig(envConfig *EnvironmentConfig) {
	if config.Bitbucket.Username == "" {
		config.Bitbucket.Username = envConfig.BitbucketUsername
	}
	if config.Bitbucket.Password == "" {
		config.Bitbucket.Password = envConfig.BitbucketPassword
	}
	if config.Github.Token == "" {
		config.Github.Token = envConfig.GithubToken
	}
	if config.Slack.Token == "" {
		config.Slack.Token = envConfig.SlackToken
	}
}

func (config *TeamConfig) GetBitbucketUsers() []string {
	users := []string{}
	for _, user := range config.Users {
		if user.BitbucketUsername != "" {
			users = append(users, user.BitbucketUsername)
		}
	}
	return users
}

func (config *TeamConfig) IsBitbucketConfigured() bool {
	return len(config.Bitbucket.Repositories) > 0 && len(config.GetBitbucketUsers()) > 0 &&
		config.Bitbucket.Username != "" && config.Bitbucket.Password != ""
}

func (config *TeamConfig) GetGithubUsers() []string {
	users := []string{}
	for _, user := range config.Users {
		if user.GithubUsername != "" {
			users = append(users, user.GithubUsername)
		}
	}
	return users
}

func (config *TeamConfig) IsGithubConfigured() bool {
	return len(config.Github.Repositories) > 0 && len(config.GetGithubUsers()) > 0 &&
		config.Github.Token != ""
}
