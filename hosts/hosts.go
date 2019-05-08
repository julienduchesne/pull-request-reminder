package hosts

import (
	"regexp"
	"strings"
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
}

func NewRepository(host Host, name, link string) *Repository {
	return &Repository{
		Link:                      link,
		Name:                      name,
		Host:                      host,
		OpenPullRequests:          []*PullRequest{},
		ReadyToMergePullRequests:  []*PullRequest{},
		ReadyToReviewPullRequests: []*PullRequest{},
	}
}

func (repository *Repository) HasPullRequestsToDisplay() bool {
	return len(repository.ReadyToMergePullRequests)+len(repository.ReadyToReviewPullRequests) > 0
}

type Host interface {
	GetName() string
	GetRepositories() []*Repository
	GetUsers() []string
}

func GetHosts() []Host {
	hosts := []Host{}
	if bitbucketCloud := newBitbucketCloud(); bitbucketCloud != nil {
		hosts = append(hosts, bitbucketCloud)
	}
	return hosts
}
