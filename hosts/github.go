package hosts

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v25/github"
	"github.com/julienduchesne/pull-request-reminder/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type githubHost struct {
	client          *github.Client
	ctx             context.Context
	users           map[string]config.User
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
		client:          github.NewClient(tc),
		ctx:             ctx,
		repositoryNames: githubConfig.Repositories,
		users:           config.GetGithubUsers(),
	}

}

func (host *githubHost) getPullRequests(owner, repoSlug string) ([]*PullRequest, error) {
	var result = []*PullRequest{}

	response, _, err := host.client.PullRequests.List(host.ctx, owner, repoSlug, &github.PullRequestListOptions{State: "open"})
	if err != nil {
		return nil, err
	}

	for _, githubPullRequest := range response {
		pullRequest := &PullRequest{
			Author:      host.users[*githubPullRequest.User.Login],
			Description: *githubPullRequest.Body,
			Link:        *githubPullRequest.HTMLURL,
			Title:       *githubPullRequest.Title,
			Reviewers:   []*Reviewer{},
		}

		allGithubReviews := []*github.PullRequestReview{}
		currentPage, lastPage := 1, 1
		for currentPage <= lastPage {
			reviews, response, err := host.client.PullRequests.ListReviews(host.ctx, owner, repoSlug, *githubPullRequest.Number, &github.ListOptions{Page: currentPage})
			if err != nil {
				return nil, err
			}
			lastPage = response.LastPage
			currentPage++
			allGithubReviews = append(allGithubReviews, reviews...)
		}

		reviewerMap := map[string]*Reviewer{}
		for i := len(allGithubReviews) - 1; i >= 0; i-- {
			review := allGithubReviews[i]
			reviewUser := *review.User.Login
			if reviewUser == pullRequest.Author.Name {
				continue // Ignore reviews by author
			}
			if reviewer, ok := reviewerMap[reviewUser]; ok && (reviewer.Approved || reviewer.RequestedChanges) {
				continue // Already handled
			}
			reviewerMap[reviewUser] = &Reviewer{
				User:             host.users[reviewUser],
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

func (host *githubHost) GetName() string {
	return "Github"
}

func (host *githubHost) GetUsers() map[string]config.User {
	return host.users
}

func (host *githubHost) GetRepositories() []*Repository {
	repositories := []*Repository{}
	for _, repositoryName := range host.repositoryNames {
		splitRepository := strings.Split(repositoryName, "/")
		owner, slug := splitRepository[0], splitRepository[1]
		pullRequests, err := host.getPullRequests(owner, slug)
		if err != nil {
			log.WithError(err).Fatalln("Caught an error while describing pull requests")
		}
		repository := NewRepository(host, repositoryName, fmt.Sprintf("https://github.com/%v", repositoryName), pullRequests)
		repositories = append(repositories, repository)
	}
	return repositories
}
