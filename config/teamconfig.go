package config

import (
	"time"
)

// TeamConfig represents the full configuration needed to handle a team.
// Since teams are all independent, this struct is passed to all handlers and
// it needs to contain all the necessary information to do the whole job
type TeamConfig struct {
	Name                    string        `yaml:"name"`
	AgeBeforeNotifying      time.Duration `yaml:"age_before_notifying"`
	NumberOfApprovals       int           `yaml:"number_of_approvals"`
	ReviewPRsFromNonMembers bool          `yaml:"review_pr_from_non_members"`
	Hosts                   struct {
		Bitbucket BitbucketConfig `yaml:"bitbucket"`
		Github    GithubConfig    `yaml:"github"`
	}
	Messaging struct {
		Slack SlackConfig `yaml:"slack"`
	}
	Users []User `yaml:"users"`
}

// BitbucketConfig represents a team's bitbucket configuration
type BitbucketConfig struct {
	Username        string   `yaml:"username"`
	Password        string   `yaml:"password"`
	Repositories    []string `yaml:"repositories"`
	Projects        []string `yaml:"projects"`
	Team            string   `yaml:"team"`
	FindUsersInTeam bool     `yaml:"find_users_in_team"`
}

// GithubConfig represents a team's github configuration
type GithubConfig struct {
	Repositories []string `yaml:"repositories"`
	Token        string   `yaml:"token"`
}

// SlackConfig represents a team's slack configuration
type SlackConfig struct {
	Channel                  string `yaml:"channel"`
	MessageUsersIndividually bool   `yaml:"message_users_individually"`
	Token                    string `yaml:"token"`

	DebugUser string `yaml:"debug_user"`
}

// User represents a team member's configuration
type User struct {
	Name           string `yaml:"name"`
	BitbucketUUID  string `yaml:"bitbucket_uuid"`
	GithubUsername string `yaml:"github_username"`
	SlackUsername  string `yaml:"slack_username"`
}

// GetNumberOfNeededApprovals returns the number of approvals needed for a pull request to be considered accepted.
// It simply returns the configured number with a minimum of 1
func (config *TeamConfig) GetNumberOfNeededApprovals() int {
	if config.NumberOfApprovals <= 0 {
		return 1
	}
	return config.NumberOfApprovals
}

// IsBitbucketConfigured returns true if all necessary configurations are set to handle Bitbucket
func (config *TeamConfig) IsBitbucketConfigured() bool {
	bitbucketConfig := config.Hosts.Bitbucket
	return len(bitbucketConfig.Repositories)+len(bitbucketConfig.Projects) > 0 && bitbucketConfig.Username != "" && bitbucketConfig.Password != ""
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
