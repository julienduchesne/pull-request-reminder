package hosts

import (
	"regexp"
	"strings"

	"github.com/julienduchesne/pull-request-reminder/config"
	log "github.com/sirupsen/logrus"
)

// Reviewer represents a user that approves, requests changes or has not reviewed yet
type Reviewer struct {
	Approved         bool
	RequestedChanges bool
	User             config.User
}

// PullRequest represent a pull (or merge) request on a SCM provider
type PullRequest struct {
	Author      config.User
	Description string
	Link        string
	Reviewers   []*Reviewer
	Title       string
}

// IsApproved returns true if the pull request is approved and ready to merge
func (pr *PullRequest) IsApproved(team map[string]config.User) bool {
	for _, reviewer := range pr.TeamReviewers(team) {
		if reviewer.Approved {
			return true
		}
	}
	return false
}

// IsFromOneOfUsers returns true if the pull request was submitted by one of the given users
func (pr *PullRequest) IsFromOneOfUsers(team map[string]config.User) bool {
	for _, teamMember := range team {
		if pr.Author.Name == teamMember.Name {
			return true
		}
	}
	return false
}

// IsWIP returns true if the pull request is marked as a work in progress
func (pr *PullRequest) IsWIP() bool {
	titleWithoutSpecialChars := regexp.MustCompile("[^a-zA-Z]+").ReplaceAllString(pr.Title, " ")
	for _, word := range strings.Split(titleWithoutSpecialChars, " ") {
		if strings.ToLower(word) == "wip" {
			return true
		}
	}
	return false
}

// TeamReviewers returns all the reviewers that are in the given list of usernames (the team)
func (pr *PullRequest) TeamReviewers(team map[string]config.User) []*Reviewer {
	reviewers := []*Reviewer{}
	for _, reviewer := range pr.Reviewers {
		for _, teamMember := range team {
			if reviewer.User.Name == teamMember.Name {
				reviewers = append(reviewers, reviewer)
			}
		}
	}
	return reviewers
}

// Repository represents a repository on a SCM provider
type Repository struct {
	Link string
	Host Host
	Name string

	OpenPullRequests          []*PullRequest
	ReadyToMergePullRequests  []*PullRequest
	ReadyToReviewPullRequests []*PullRequest
}

// NewRepository creates a Repository instance
func NewRepository(host Host, name, link string, openPullRequests []*PullRequest) *Repository {
	repository := &Repository{
		Link:                      link,
		Name:                      name,
		Host:                      host,
		OpenPullRequests:          openPullRequests,
		ReadyToMergePullRequests:  []*PullRequest{},
		ReadyToReviewPullRequests: []*PullRequest{},
	}
	repository.categorizePullRequests()
	return repository

}

func (repository *Repository) categorizePullRequests() {
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
}

// HasPullRequestsToDisplay returns true if at least one of the pull requests needs action by the team (ready to merge or needs approval)
func (repository *Repository) HasPullRequestsToDisplay() bool {
	return len(repository.ReadyToMergePullRequests)+len(repository.ReadyToReviewPullRequests) > 0
}

// Host represents a SCM provider
type Host interface {
	GetName() string
	GetRepositories() []*Repository
	GetUsers() map[string]config.User
}

// GetHosts returns all configured Hosts (SCM providers)
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
