package hosts

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/ktrysmt/go-bitbucket"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

type bitbucketPullRequest struct {
	Author struct {
		Username string
	}
	Description string
	Links       map[string]struct {
		Href string
		Name string
	}
	Participants []struct {
		Approved bool
		Role     string
		User     struct {
			Username string
		}
	}
	Title string
}

func (pr *bitbucketPullRequest) ToGenericPullRequest() *PullRequest {
	reviewers := []*Reviewer{}
	for _, participant := range pr.Participants {
		if participant.Role == "REVIEWER" {
			reviewers = append(reviewers, &Reviewer{
				Approved:         participant.Approved,
				RequestedChanges: false, // not supported by bitbucket
				Username:         participant.User.Username,
			})
		}
	}

	return &PullRequest{
		Author:      pr.Author.Username,
		Description: pr.Description,
		Link:        pr.Links["html"].Href,
		Title:       pr.Title,
		Reviewers:   reviewers,
	}
}

type bitbucketListPullRequestsResponse struct {
	Values []struct {
		ID int
	}
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

func (host *bitbucketCloud) getPullRequests(owner, repoSlug string) ([]*bitbucketPullRequest, error) {
	var (
		err      error
		response interface{}
	)

	opt := &bitbucket.PullRequestsOptions{
		Owner:    owner,
		RepoSlug: repoSlug,
	}
	getPullRequestFunc := func() error {
		response, err = host.client.Repositories.PullRequests.Get(opt)
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
		pullRequests, err := host.getPullRequests(owner, slug)
		if err != nil {
			log.WithError(err).Fatalln("Caught an error while describing pull requests")
		}
		for _, pullRequest := range pullRequests {
			repository.OpenPullRequests = append(repository.OpenPullRequests, pullRequest.ToGenericPullRequest())
		}
		repositories = append(repositories, repository)
	}
	return repositories
}
