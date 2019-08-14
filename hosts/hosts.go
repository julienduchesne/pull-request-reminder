package hosts

import (
	"fmt"
	"regexp"
	"strings"
	"time"

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
	CreateTime  time.Time
	UpdateTime  time.Time
}

// IsApproved returns true if the pull request is approved and ready to merge
func (pr *PullRequest) IsApproved(team map[string]config.User, numberOfApprovals int) bool {
	approvalsGotten := 0
	for _, reviewer := range pr.TeamReviewers(team) {
		if reviewer.Approved {
			approvalsGotten++
		}
	}
	return approvalsGotten >= numberOfApprovals
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
type Repository interface {
	GetHost() Host
	GetLink() string
	GetName() string
	GetPullRequestsToDisplay() (readyToMerge []*PullRequest, readyToReview []*PullRequest)
	HasPullRequestsToDisplay() bool
}

// RepositoryImpl is the implementation of the Repository interface.
type RepositoryImpl struct {
	Host Host
	Link string
	Name string

	OpenPullRequests []*PullRequest
}

// NewRepository creates a RepositoryImpl instance
func NewRepository(host Host, name, link string, openPullRequests []*PullRequest) *RepositoryImpl {
	repository := &RepositoryImpl{
		Link:             link,
		Name:             name,
		Host:             host,
		OpenPullRequests: openPullRequests,
	}
	return repository
}

// GetHost returns a repository's host
func (repository *RepositoryImpl) GetHost() Host {
	return repository.Host
}

// GetLink returns a repository's URL
func (repository *RepositoryImpl) GetLink() string {
	return repository.Link
}

// GetName returns a repository's name
func (repository *RepositoryImpl) GetName() string {
	return repository.Name
}

// GetPullRequestsToDisplay returns all pull requests that are either waiting for approvals or ready to merge
func (repository *RepositoryImpl) GetPullRequestsToDisplay() (readyToMerge []*PullRequest, readyToReview []*PullRequest) {
	config := repository.GetHost().GetConfig()

	readyToMerge, readyToReview = []*PullRequest{}, []*PullRequest{}
	for _, pullRequest := range repository.OpenPullRequests {

		var logIgnoredPullRequest = func(message string) {
			log.Infof("%s: %s (%s) ignored because %s", repository.Name, pullRequest.Title, pullRequest.Link, message)
		}

		if pullRequest.IsWIP() {
			logIgnoredPullRequest("Marked WIP")
			continue
		}
		if len(pullRequest.TeamReviewers(repository.Host.GetUsers())) == 0 {
			logIgnoredPullRequest("No reviewers")
			continue
		}
		if pullRequest.CreateTime.After(time.Now().Add(-config.AgeBeforeNotifying)) {
			logIgnoredPullRequest(fmt.Sprintf("Not old enough. It hasn't been created for %v", config.AgeBeforeNotifying))
			continue
		}

		if pullRequest.IsApproved(repository.Host.GetUsers(), config.GetNumberOfNeededApprovals()) {
			if !pullRequest.IsFromOneOfUsers(repository.Host.GetUsers()) {
				logIgnoredPullRequest("Not from one of the team's users")
				continue
			}
			if pullRequest.UpdateTime.After(time.Now().Add(-config.AgeBeforeNotifying)) {
				logIgnoredPullRequest(fmt.Sprintf("Merge not overdue, hasn't been stale for %v", config.AgeBeforeNotifying))
				continue
			}
			readyToMerge = append(readyToMerge, pullRequest)
		} else {
			if !config.ReviewPRsFromNonMembers && !pullRequest.IsFromOneOfUsers(repository.Host.GetUsers()) {
				logIgnoredPullRequest("Not from one of the team's users")
				continue
			}
			readyToReview = append(readyToReview, pullRequest)
		}
	}
	return
}

// HasPullRequestsToDisplay returns true if at least one of the pull requests needs action by the team (ready to merge or needs approval)
func (repository *RepositoryImpl) HasPullRequestsToDisplay() bool {
	readyToMerge, readyToReview := repository.GetPullRequestsToDisplay()
	return len(readyToMerge)+len(readyToReview) > 0
}

// Host represents a SCM provider
type Host interface {
	GetConfig() *config.TeamConfig
	GetName() string
	GetRepositories() []Repository
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
