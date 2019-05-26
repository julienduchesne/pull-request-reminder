package config

// TeamConfig represents the full configuration needed to handle a team.
// Since teams are all independent, this struct is passed to all handlers and
// it needs to contain all the necessary information to do the whole job
type TeamConfig struct {
	Name  string
	Hosts struct {
		Bitbucket BitbucketConfig
		Github    GithubConfig
	}
	Messaging struct {
		Slack SlackConfig
	}
	Users []User
}

// BitbucketConfig represents a team's bitbucket configuration
type BitbucketConfig struct {
	Username     string
	Password     string
	Repositories []string
}

// GithubConfig represents a team's github configuration
type GithubConfig struct {
	Repositories []string
	Token        string
}

// SlackConfig represents a team's slack configuration
type SlackConfig struct {
	Channel string
	Token   string
}

// User represents a team member's configuration
type User struct {
	Name           string
	BitbucketUUID  string `yaml:"bitbucket_uuid"`
	GithubUsername string `yaml:"github_username"`
	SlackUsername  string `yaml:"slack_username"`
}

// GetBitbucketUsers returns a map of all bitbucket users'
func (config *TeamConfig) GetBitbucketUsers() map[string]User {
	users := map[string]User{}
	for _, user := range config.Users {
		if user.BitbucketUUID != "" {
			users[user.BitbucketUUID] = user
		}
	}
	return users
}

// IsBitbucketConfigured returns true if all necessary configurations are set to handle Bitbucket
func (config *TeamConfig) IsBitbucketConfigured() bool {
	bitbucketConfig := config.Hosts.Bitbucket
	return len(bitbucketConfig.Repositories) > 0 && len(config.GetBitbucketUsers()) > 0 &&
		bitbucketConfig.Username != "" && bitbucketConfig.Password != ""
}

// GetGithubUsers returns a map of all github users'
func (config *TeamConfig) GetGithubUsers() map[string]User {
	users := map[string]User{}
	for _, user := range config.Users {
		if user.GithubUsername != "" {
			users[user.GithubUsername] = user
		}
	}
	return users
}

// IsGithubConfigured returns true if all necessary configurations are set to handle Github
func (config *TeamConfig) IsGithubConfigured() bool {
	githubConfig := config.Hosts.Github
	return len(githubConfig.Repositories) > 0 && len(config.GetGithubUsers()) > 0 &&
		githubConfig.Token != ""
}

func (config *TeamConfig) setEnvironmentConfig(envConfig *EnvironmentConfig) {
	bitbucketConfig := &config.Hosts.Bitbucket
	githubConfig := &config.Hosts.Github
	slackConfig := &config.Messaging.Slack
	if bitbucketConfig.Username == "" {
		bitbucketConfig.Username = envConfig.BitbucketUsername
	}
	if bitbucketConfig.Password == "" {
		bitbucketConfig.Password = envConfig.BitbucketPassword
	}
	if githubConfig.Token == "" {
		githubConfig.Token = envConfig.GithubToken
	}
	if slackConfig.Token == "" {
		slackConfig.Token = envConfig.SlackToken
	}
}
