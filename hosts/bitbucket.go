package hosts

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/ktrysmt/go-bitbucket"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

type bitbucketListPullRequestsResponse struct {
	Values []*bitbucketPullRequest
}

type link struct {
	Href string
	Name string
}

type bitbucketPullRequest struct {
	Author       *bitbucketUser
	Description  string
	ID           int
	Links        map[string]*link
	Participants []*bitbucketPullRequestParticipant
	Title        string
}

func (pr *bitbucketPullRequest) GetAuthor() User {
	return pr.Author
}

func (pr *bitbucketPullRequest) GetDescription() string {
	return pr.Description
}

func (pr *bitbucketPullRequest) GetLink() string {
	return pr.Links["html"].Href
}

func (pr *bitbucketPullRequest) GetTitle() string {
	return pr.Title
}

func (pr *bitbucketPullRequest) IsFromOneOfUsers(usernames []string) bool {
	for _, username := range usernames {
		if pr.Author.Username == username {
			return true
		}
	}
	return false
}

func (pr *bitbucketPullRequest) IsWIP() bool {
	titleWithoutSpecialChars := regexp.MustCompile("[^a-zA-Z]+").ReplaceAllString(pr.Title, " ")
	for _, word := range strings.Split(titleWithoutSpecialChars, " ") {
		if strings.ToLower(word) == "wip" {
			return true
		}
	}
	return false
}

func (pr *bitbucketPullRequest) Reviewers(teamUsernames []string) []PullRequestParticipant {
	reviewers := []PullRequestParticipant{}
	for _, participant := range pr.Participants {
		for _, teamUsername := range teamUsernames {
			if participant.Role == "REVIEWER" && participant.User.Username == teamUsername {
				reviewers = append(reviewers, participant)
			}
		}
	}
	return reviewers
}

func (pr *bitbucketPullRequest) IsApproved(teamUsernames []string) bool {
	for _, reviewer := range pr.Reviewers(teamUsernames) {
		if reviewer.HasApproved() {
			return true
		}
	}
	return false
}

type bitbucketPullRequestParticipant struct {
	Approved bool
	Role     string
	User     *bitbucketUser
}

func (participant *bitbucketPullRequestParticipant) GetUsername() string {
	return participant.User.Username
}

func (participant *bitbucketPullRequestParticipant) HasApproved() bool {
	return participant.Approved
}

type bitbucketUser struct {
	Username string
}

func (user *bitbucketUser) GetUsername() string {
	return user.Username
}

func getRepositoryPullRequests(client *bitbucket.Client, owner, repoSlug string) ([]*bitbucketPullRequest, error) {
	var (
		err      error
		response interface{}
	)

	opt := &bitbucket.PullRequestsOptions{
		Owner:    owner,
		RepoSlug: repoSlug,
	}
	getPullRequestFunc := func() error {
		response, err = client.Repositories.PullRequests.Get(opt)
		return err
	}

	err = backoff.Retry(getPullRequestFunc, backoff.NewExponentialBackOff())
	if err != nil {
		return nil, err
	}

	listedPullRequests, detailedpullRequests := &bitbucketListPullRequestsResponse{}, []*bitbucketPullRequest{}
	if err = mapstructure.Decode(response, &listedPullRequests); err != nil {
		return nil, err
	}

	for _, listedPullRequest := range listedPullRequests.Values {
		opt.ID = strconv.Itoa(listedPullRequest.ID)
		err = backoff.Retry(getPullRequestFunc, backoff.NewExponentialBackOff())
		if err != nil {
			return nil, err
		}
		var pullRequest bitbucketPullRequest
		if err = mapstructure.Decode(response, &pullRequest); err != nil {
			return nil, err
		}
		detailedpullRequests = append(detailedpullRequests, &pullRequest)
	}

	return detailedpullRequests, nil
}

type bitbucketCloud struct {
	client          *bitbucket.Client
	users           []string
	repositoryNames []string
}

func newBitbucketCloud() *bitbucketCloud {
	username := os.Getenv("BITBUCKET_USERNAME")
	password := os.Getenv("BITBUCKET_PASSWORD")
	repositoryNames := strings.Split(os.Getenv("BITBUCKET_REPOSITORIES"), ";")
	users := strings.Split(os.Getenv("BITBUCKET_USERS"), ";")
	if repositoryNames[0] == "" || users[0] == "" {
		log.Infoln("You must set BITBUCKET_REPOSITORIES and BITBUCKET_USERS to handle bitbucket")
		return nil
	}
	if username == "" || password == "" {
		log.Infoln("You must set BITBUCKET_USERNAME and BITBUCKET_PASSWORD to handle bitbucket")
		return nil
	}
	return &bitbucketCloud{
		client:          bitbucket.NewBasicAuth(username, password),
		repositoryNames: repositoryNames,
		users:           users,
	}

}

func (host *bitbucketCloud) GetName() string {
	return "Bitbucket"
}

func (host *bitbucketCloud) GetUsers() []string {
	return host.users
}

func (host *bitbucketCloud) GetRepositories() []*Repository {
	repositories := []*Repository{}
	for _, repositoryName := range host.repositoryNames {
		repository := NewRepository(host, repositoryName, fmt.Sprintf("https://bitbucket.org/%v", repositoryName))
		splitRepository := strings.Split(repositoryName, "/")
		owner, slug := splitRepository[0], splitRepository[1]
		pullRequests, err := getRepositoryPullRequests(host.client, owner, slug)
		if err != nil {
			log.WithError(err).Fatalln("Caught an error while describing pull requests")
		}
		for _, pullRequest := range pullRequests {
			repository.OpenPullRequests = append(repository.OpenPullRequests, pullRequest)
		}
		repositories = append(repositories, repository)
	}
	return repositories
}
