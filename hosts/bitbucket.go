package hosts

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/ktrysmt/go-bitbucket"
	"github.com/matryer/try"
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

type bitbucketClient interface {
	GetPullRequests(owner, slug, id string) (interface{}, error)
	GetTeamMembers(team string) (interface{}, error)
}

type bitbucketClientWrapper struct {
	client *bitbucket.Client
}

func (wrapper *bitbucketClientWrapper) GetPullRequests(owner, slug, id string) (interface{}, error) {
	return wrapper.client.Repositories.PullRequests.Get(&bitbucket.PullRequestsOptions{
		Owner:    owner,
		RepoSlug: slug,
		ID:       id,
	})
}

func (wrapper *bitbucketClientWrapper) GetTeamMembers(team string) (interface{}, error) {
	return wrapper.client.Teams.Members(team)
}

type bitbucketCloud struct {
	config          *config.TeamConfig
	client          bitbucketClient
	repositoryNames []string
}

func newBitbucketCloud(config *config.TeamConfig) *bitbucketCloud {
	bitbucketConfig := config.Hosts.Bitbucket
	return &bitbucketCloud{
		config:          config,
		client:          &bitbucketClientWrapper{client: bitbucket.NewBasicAuth(bitbucketConfig.Username, bitbucketConfig.Password)},
		repositoryNames: bitbucketConfig.Repositories,
	}

}

func (host *bitbucketCloud) getPullRequests(owner, repoSlug string) ([]*PullRequest, error) {
	var (
		response interface{}
		err      error
	)

	result := []*PullRequest{}
	listedPullRequests := &struct {
		Values []struct {
			ID int
		}
	}{}

	if err := try.Do(func(attempt int) (bool, error) {
		response, err = host.client.GetPullRequests(owner, repoSlug, "")
		return attempt < 5, err
	}); err != nil {
		return nil, fmt.Errorf("Error fetching pull requests from %v/%v in Bitbucket", owner, repoSlug)
	}

	if err = mapstructure.Decode(response, &listedPullRequests); err != nil {
		return nil, err
	}

	for _, listedPullRequest := range listedPullRequests.Values {
		if err := try.Do(func(attempt int) (bool, error) {
			response, err = host.client.GetPullRequests(owner, repoSlug, strconv.Itoa(listedPullRequest.ID))
			return attempt < 5, err
		}); err != nil {
			return nil, fmt.Errorf("Error fetching the pull request with ID %v from %v/%v in Bitbucket", listedPullRequest.ID, owner, repoSlug)
		}

		var pullRequest bitbucketPullRequest
		if err = mapstructure.Decode(response, &pullRequest); err != nil {
			return nil, err
		}
		result = append(result, pullRequest.ToGenericPullRequest(host.GetUsers()))
	}

	return result, nil
}

func (host *bitbucketCloud) GetConfig() *config.TeamConfig {
	return host.config
}

func (host *bitbucketCloud) GetName() string {
	return "Bitbucket"
}

func (host *bitbucketCloud) GetUsers() map[string]config.User {
	users := map[string]config.User{}
	for _, user := range host.config.Users {
		if user.BitbucketUUID != "" {
			users[user.BitbucketUUID] = user
		}
	}
	return users
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
