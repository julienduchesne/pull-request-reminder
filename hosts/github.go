package hosts

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v25/github"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type githubHost struct {
	client          *github.Client
	ctx             context.Context
	users           []string
	repositoryNames []string
}

func newGithubHost() *githubHost {
	token := os.Getenv("GITHUB_TOKEN")
	repositoryNames := strings.Split(os.Getenv("GITHUB_REPOSITORIES"), ";")
	users := strings.Split(os.Getenv("GITHUB_USERS"), ";")
	if repositoryNames[0] == "" || users[0] == "" || token == "" {
		log.Infoln("You must set GITHUB_REPOSITORIES, GITHUB_USERS and GITHUB_TOKEN to handle github")
		return nil
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &githubHost{
		client:          github.NewClient(tc),
		ctx:             ctx,
		repositoryNames: repositoryNames,
		users:           users,
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
			Author:      *githubPullRequest.User.Login,
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
			if reviewUser == pullRequest.Author {
				continue // Ignore reviews by author
			}
			if reviewer, ok := reviewerMap[reviewUser]; ok && (reviewer.Approved || reviewer.RequestedChanges) {
				continue // Already handled
			}
			reviewerMap[reviewUser] = &Reviewer{
				Username:         reviewUser,
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

func (host *githubHost) GetUsers() []string {
	return host.users
}

func (host *githubHost) GetRepositories() []*Repository {
	repositories := []*Repository{}
	for _, repositoryName := range host.repositoryNames {
		repository := NewRepository(host, repositoryName, fmt.Sprintf("https://github.com/%v", repositoryName))
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
