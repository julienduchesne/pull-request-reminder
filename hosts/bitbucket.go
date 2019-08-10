package hosts

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/ktrysmt/go-bitbucket"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
)

type bitbucketPullRequest struct {
	Author struct {
		UUID string
	}
	CreatedOn   string `mapstructure:"created_on"`
	UpdatedOn   string `mapstructure:"updated_on"`
	Description string
	Links       map[string]struct {
		Href string
		Name string
	}
	Participants []struct {
		Approved bool
		Role     string
		User     struct {
			UUID string
		}
	}
	Title string
}

func (pr *bitbucketPullRequest) ToGenericPullRequest(users map[string]config.User) *PullRequest {
	reviewers := []*Reviewer{}
	for _, participant := range pr.Participants {
		if participant.Role == "REVIEWER" {
			reviewers = append(reviewers, &Reviewer{
				Approved:         participant.Approved,
				RequestedChanges: false, // not supported by bitbucket, maybe with open tasks?
				User:             users[participant.User.UUID],
			})
		}
	}

	genericPullRequest := &PullRequest{
		Author:      users[pr.Author.UUID],
		Description: pr.Description,
		Link:        pr.Links["html"].Href,
		Title:       pr.Title,
		Reviewers:   reviewers,
	}

	var err error
	if genericPullRequest.CreateTime, err = time.Parse(time.RFC3339Nano, pr.CreatedOn); err != nil {
		log.Warningf("Error parsing create date %s from PR %s", pr.CreatedOn, pr.Title)
	}
	if genericPullRequest.UpdateTime, err = time.Parse(time.RFC3339Nano, pr.UpdatedOn); err != nil {
		log.Warningf("Error parsing update date %s from PR %s", pr.UpdatedOn, pr.Title)
	}

	return genericPullRequest
}

type bitbucketClientInterface interface {
	GetPullRequests(owner, slug, id string) (interface{}, error)
}

type bitbucketClientWrapper struct {
	client *bitbucket.Client
}

func newBitbucketClientWrapper(config config.BitbucketConfig) *bitbucketClientWrapper {
	return &bitbucketClientWrapper{client: bitbucket.NewBasicAuth(config.Username, config.Password)}
}

func (wrapper *bitbucketClientWrapper) GetPullRequests(owner, slug, id string) (interface{}, error) {
	var (
		err      error
		response interface{}
	)
	opt := &bitbucket.PullRequestsOptions{
		Owner:    owner,
		RepoSlug: slug,
	}
	if id != "" {
		opt.ID = id
	}
	getPullRequestFunc := func() error {
		response, err = wrapper.client.Repositories.PullRequests.Get(opt)
		return err
	}
	err = backoff.Retry(getPullRequestFunc, backoff.NewExponentialBackOff())
	if err != nil {
		return nil, err
	}
	return response, err
}

type bitbucketCloud struct {
	client          bitbucketClientInterface
	users           map[string]config.User
	repositoryNames []string
}

func newBitbucketCloud(config *config.TeamConfig) *bitbucketCloud {
	bitbucketConfig := config.Hosts.Bitbucket
	return &bitbucketCloud{
		client:          newBitbucketClientWrapper(bitbucketConfig),
		repositoryNames: bitbucketConfig.Repositories,
		users:           config.GetBitbucketUsers(),
	}

}

func (host *bitbucketCloud) getPullRequests(owner, repoSlug string) ([]*PullRequest, error) {
	result := []*PullRequest{}
	listedPullRequests := &struct {
		Values []struct {
			ID int
		}
	}{}

	response, err := host.client.GetPullRequests(owner, repoSlug, "")
	if err != nil {
		return nil, fmt.Errorf("Error fetching pull requests from %v/%v in Bitbucket", owner, repoSlug)
	}
	if err = mapstructure.Decode(response, &listedPullRequests); err != nil {
		return nil, err
	}

	for _, listedPullRequest := range listedPullRequests.Values {
		if response, err = host.client.GetPullRequests(owner, repoSlug, strconv.Itoa(listedPullRequest.ID)); err != nil {
			return nil, err
		}
		var pullRequest bitbucketPullRequest
		if err = mapstructure.Decode(response, &pullRequest); err != nil {
			return nil, err
		}
		result = append(result, pullRequest.ToGenericPullRequest(host.users))
	}

	return result, nil
}

func (host *bitbucketCloud) GetName() string {
	return "Bitbucket"
}

func (host *bitbucketCloud) GetUsers() map[string]config.User {
	return host.users
}

func (host *bitbucketCloud) GetRepositories() []Repository {
	repositories := []Repository{}
	for _, repositoryName := range host.repositoryNames {
		splitRepository := strings.Split(repositoryName, "/")
		owner, slug := splitRepository[0], splitRepository[1]
		pullRequests, err := host.getPullRequests(owner, slug)
		if err != nil {
			log.WithError(err).Fatalln("Caught an error while describing pull requests")
		}
		repository := NewRepository(host, repositoryName, fmt.Sprintf("https://bitbucket.org/%v", repositoryName), pullRequests)
		repositories = append(repositories, repository)
	}
	return repositories
}
