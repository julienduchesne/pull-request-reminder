package hosts

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cenkalti/backoff"
	"github.com/julienduchesne/pull-request-reminder/config"
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

type bitbucketCloud struct {
	client          *bitbucket.Client
	users           []string
	repositoryNames []string
}

func newBitbucketCloud(config *config.TeamConfig) *bitbucketCloud {
	return &bitbucketCloud{
		client:          bitbucket.NewBasicAuth(config.Bitbucket.Username, config.Bitbucket.Password),
		repositoryNames: config.Bitbucket.Repositories,
		users:           config.GetBitbucketUsers(),
	}

}

func (host *bitbucketCloud) getPullRequests(owner, repoSlug string) ([]*PullRequest, error) {
	var (
		err      error
		response interface{}
		result   = []*PullRequest{}
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

	listedPullRequests := &struct {
		Values []struct {
			ID int
		}
	}{}
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
		result = append(result, pullRequest.ToGenericPullRequest())
	}

	return result, nil
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
		repository.OpenPullRequests = pullRequests
		repositories = append(repositories, repository)
	}
	return repositories
}
