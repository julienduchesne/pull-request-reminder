package hosts

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v25/github"
	"github.com/julienduchesne/pull-request-reminder/config"
	"golang.org/x/oauth2"
)

type githubClient interface {
	ListPullRequests(owner string, repo string, opt *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	ListReviews(owner, repo string, number int, opt *github.ListOptions) ([]*github.PullRequestReview, *github.Response, error)
}

type githubClientWrapper struct {
	client *github.Client
	ctx    context.Context
}

func (wrapper *githubClientWrapper) ListPullRequests(owner string, repo string, opt *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	return wrapper.client.PullRequests.List(wrapper.ctx, owner, repo, opt)
}

func (wrapper *githubClientWrapper) ListReviews(owner, repo string, number int, opt *github.ListOptions) ([]*github.PullRequestReview, *github.Response, error) {
	return wrapper.client.PullRequests.ListReviews(wrapper.ctx, owner, repo, number, opt)
}

type githubHost struct {
	config          *config.TeamConfig
	client          githubClient
	repositoryNames []string
}

func newGithubHost(config *config.TeamConfig) *githubHost {
	githubConfig := config.Hosts.Github
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubConfig.Token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &githubHost{
		config: config,
		client: &githubClientWrapper{
			client: github.NewClient(tc),
			ctx:    ctx,
		},
		repositoryNames: githubConfig.Repositories,
	}

}

func (host *githubHost) getPullRequests(owner, repoSlug string, users map[string]config.User) ([]*PullRequest, error) {
	var result = []*PullRequest{}

	response, _, err := host.client.ListPullRequests(owner, repoSlug, &github.PullRequestListOptions{State: "open"})
	if err != nil {
		return nil, fmt.Errorf("Error fetching pull requests from %s/%s in Github: %v", owner, repoSlug, err)
	}

	for _, githubPullRequest := range response {
		pullRequest := &PullRequest{
			Author:      users[*githubPullRequest.User.Login],
			Description: *githubPullRequest.Body,
			Link:        *githubPullRequest.HTMLURL,
			Title:       *githubPullRequest.Title,
			Reviewers:   []*Reviewer{},
			CreateTime:  *githubPullRequest.CreatedAt,
			UpdateTime:  *githubPullRequest.UpdatedAt,
		}

		allGithubReviews := []*github.PullRequestReview{}
		currentPage, lastPage := 1, 1
		for currentPage <= lastPage {
			reviews, response, err := host.client.ListReviews(owner, repoSlug, *githubPullRequest.Number, &github.ListOptions{Page: currentPage})
			if err != nil {
				return nil, fmt.Errorf("Error fetching reviews from the pull request with ID %v from %s/%s in Github: %v", *githubPullRequest.Number, owner, repoSlug, err)
			}
			lastPage = response.LastPage
			currentPage++
			allGithubReviews = append(allGithubReviews, reviews...)
		}

		reviewerMap := map[string]*Reviewer{}
		for i := len(allGithubReviews) - 1; i >= 0; i-- {
			review := allGithubReviews[i]
			reviewUser := *review.User.Login
			if reviewUser == pullRequest.Author.GithubUsername {
				continue // Ignore reviews by author
			}
			if reviewer, ok := reviewerMap[reviewUser]; ok && (reviewer.Approved || reviewer.RequestedChanges) {
				continue // Already handled
			}
			reviewerMap[reviewUser] = &Reviewer{
				User:             users[reviewUser],
				Approved:         *review.State == "APPROVED",
				RequestedChanges: *review.State == "CHANGES_REQUESTED",
			}
		}

		for _, reviewer := range reviewerMap {
			pullRequest.Reviewers = append(pullRequest.Reviewers, reviewer)
		}

		result = append(result, pullRequest)
	}

	return result, nil
}

func (host *githubHost) GetConfig() *config.TeamConfig {
	return host.config
}

func (host *githubHost) GetName() string {
	return "Github"
}

func (host *githubHost) GetUsers() (map[string]config.User, error) {
	return host.config.GetGithubUsers(), nil
}

func (host *githubHost) GetRepositories() ([]Repository, error) {
	users, _ := host.GetUsers()

	repositories := []Repository{}
	for _, repositoryName := range host.repositoryNames {
		splitRepository := strings.Split(repositoryName, "/")
		owner, slug := splitRepository[0], splitRepository[1]
		pullRequests, err := host.getPullRequests(owner, slug, users)
		if err != nil {
			return nil, fmt.Errorf("Caught an error while describing pull requests: %v", err)
		}
		repository := NewRepository(host, repositoryName, fmt.Sprintf("https://github.com/%v", repositoryName), pullRequests)
		repositories = append(repositories, repository)
	}
	return repositories, nil
}
