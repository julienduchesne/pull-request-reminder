package hosts

import (
	"regexp"
	"strings"

	"github.com/julienduchesne/pull-request-reminder/config"
	log "github.com/sirupsen/logrus"
)

type Reviewer struct {
	Approved         bool
	RequestedChanges bool
	Username         string
}

type PullRequest struct {
	Author      string
	Description string
	Link        string
	Reviewers   []*Reviewer
	Title       string
}

func (pr *PullRequest) IsApproved(teamUsernames []string) bool {
	for _, reviewer := range pr.TeamReviewers(teamUsernames) {
		if reviewer.Approved {
			return true
		}
	}
	return false
}
func (pr *PullRequest) IsFromOneOfUsers(teamUsernames []string) bool {
	for _, username := range teamUsernames {
		if pr.Author == username {
			return true
		}
	}
	return false
}

func (pr *PullRequest) IsWIP() bool {
	titleWithoutSpecialChars := regexp.MustCompile("[^a-zA-Z]+").ReplaceAllString(pr.Title, " ")
	for _, word := range strings.Split(titleWithoutSpecialChars, " ") {
		if strings.ToLower(word) == "wip" {
			return true
		}
	}
	return false
}
func (pr *PullRequest) TeamReviewers(teamUsernames []string) []*Reviewer {
	reviewers := []*Reviewer{}
	for _, reviewer := range pr.Reviewers {
		for _, teamUsername := range teamUsernames {
			if reviewer.Username == teamUsername {
				reviewers = append(reviewers, reviewer)
			}
		}
	}
	return reviewers
}

type Repository struct {
	Link string
	Host Host
	Name string

	OpenPullRequests          []*PullRequest
	ReadyToMergePullRequests  []*PullRequest
	ReadyToReviewPullRequests []*PullRequest
	pullRequestsCategorized   bool
}

func NewRepository(host Host, name, link string) *Repository {
	return &Repository{
		Link:                      link,
		Name:                      name,
		Host:                      host,
		OpenPullRequests:          []*PullRequest{},
		ReadyToMergePullRequests:  []*PullRequest{},
		ReadyToReviewPullRequests: []*PullRequest{},
		pullRequestsCategorized:   false,
	}
}

func (repository *Repository) CategorizePullRequests() {
	if repository.pullRequestsCategorized {
		return
	}

	for _, pullRequest := range repository.OpenPullRequests {

		var logIgnoredPullRequest = func(message string) {
			log.Info(repository.Name, "->", pullRequest.Link, " ignored because: ", message)
		}

		if !pullRequest.IsFromOneOfUsers(repository.Host.GetUsers()) {
			logIgnoredPullRequest("Not from one of the team's users")
			continue
		}
		if pullRequest.IsWIP() {
			logIgnoredPullRequest("Marked WIP")
			continue
		}
		if len(pullRequest.TeamReviewers(repository.Host.GetUsers())) == 0 {
			logIgnoredPullRequest("No reviewers")
			continue
		}

		if pullRequest.IsApproved(repository.Host.GetUsers()) {
			repository.ReadyToMergePullRequests = append(repository.ReadyToMergePullRequests, pullRequest)
		} else {
			repository.ReadyToReviewPullRequests = append(repository.ReadyToReviewPullRequests, pullRequest)
		}
	}
	repository.pullRequestsCategorized = true
}

func (repository *Repository) HasPullRequestsToDisplay() bool {
	if !repository.pullRequestsCategorized {
		repository.CategorizePullRequests()
	}
	return len(repository.ReadyToMergePullRequests)+len(repository.ReadyToReviewPullRequests) > 0
}

type Host interface {
	GetName() string
	GetRepositories() []*Repository
	GetUsers() []string
}

func GetHosts(config *config.TeamConfig) []Host {
	hosts := []Host{}
	if config.IsBitbucketConfigured() {
		hosts = append(hosts, newBitbucketCloud(config))
	} else {
		log.Infoln("Bitbucket is not configured")
	}
	if config.IsGithubConfigured() {
		hosts = append(hosts, newGithubHost(config))
	} else {
		log.Infoln("Github is not configured")
	}
	return hosts
}
