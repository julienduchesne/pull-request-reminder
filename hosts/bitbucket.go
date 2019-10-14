package hosts

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/julienduchesne/pull-request-reminder/utilities"
	"github.com/ktrysmt/go-bitbucket"
	"github.com/matryer/try"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
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

type bitbucketTeamMember struct {
	CreatedOn string `mapstructure:"created_on"`
	Links     map[string]struct {
		Href string
		Name string
	}
	DisplayName string `mapstructure:"display_name"`
	Nickname    string `mapstructure:"nickname"`
	AccountID   string `mapstructure:"account_id"`
	UUID        string `mapstructure:"uuid"`
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
	users           map[string]config.User
}

func newBitbucketCloud(config *config.TeamConfig) *bitbucketCloud {
	bitbucketConfig := config.Hosts.Bitbucket
	return &bitbucketCloud{
		config:          config,
		client:          &bitbucketClientWrapper{client: bitbucket.NewBasicAuth(bitbucketConfig.Username, bitbucketConfig.Password)},
		repositoryNames: bitbucketConfig.Repositories,
	}

}

func (host *bitbucketCloud) getPullRequests(owner, repoSlug string, users map[string]config.User) ([]*PullRequest, error) {
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
		if err != nil {
			log.Warnf("Failed to fetch pull requests for repo %s. Waiting 10 seconds", repoSlug)
			secondsToSleep, _ := strconv.Atoi(utilities.GetEnv("BITBUCKET_RETRY_DELAY", "10"))
			time.Sleep(time.Duration(secondsToSleep) * time.Second)
		}
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
			if err != nil {
				log.Warnf("Failed to fetch pull request %d for repo %s. Waiting 10 seconds", listedPullRequest.ID, repoSlug)
				secondsToSleep, _ := strconv.Atoi(utilities.GetEnv("BITBUCKET_RETRY_DELAY", "10"))
				time.Sleep(time.Duration(secondsToSleep) * time.Second)
			}
			return attempt < 5, err
		}); err != nil {
			return nil, fmt.Errorf("Error fetching the pull request with ID %v from %v/%v in Bitbucket", listedPullRequest.ID, owner, repoSlug)
		}

		var pullRequest bitbucketPullRequest
		if err = mapstructure.Decode(response, &pullRequest); err != nil {
			return nil, err
		}
		result = append(result, pullRequest.ToGenericPullRequest(users))
	}

	return result, nil
}

func (host *bitbucketCloud) GetConfig() *config.TeamConfig {
	return host.config
}

func (host *bitbucketCloud) GetName() string {
	return "Bitbucket"
}

func (host *bitbucketCloud) GetUsers() (map[string]config.User, error) {
	if host.users == nil {
		host.users = map[string]config.User{}

		// List all bitbucket team members
		var (
			err         error
			teamMembers []bitbucketTeamMember
		)
		findUsersInTeam := host.config.Hosts.Bitbucket.FindUsersInTeam
		if findUsersInTeam {
			team := host.config.Hosts.Bitbucket.Team
			if team == "" {
				return nil, fmt.Errorf("Bitbucket is set to find users in the team but the team name is not set")
			}
			if teamMembers, err = host.getTeamMembers(team); err != nil {
				return nil, err
			}
		}

		for _, user := range host.config.Users {
			// If a team member's name matches with a user missing a UUID, use the team member's UUID
			if user.BitbucketUUID == "" && findUsersInTeam {
				for _, member := range teamMembers {
					if normalizeName(member.DisplayName) == normalizeName(user.Name) || normalizeName(member.Nickname) == normalizeName(user.Name) {
						if user.BitbucketUUID != "" {
							return nil, fmt.Errorf("User %s has multiple matches for bitbucket users. Please set the UUID directly", user.Name)
						}
						user.BitbucketUUID = member.UUID
					}
				}
			}

			// If the UUID of a user is set, we're good to go. We can use that user
			if user.BitbucketUUID != "" {
				if userWithSameUUID, ok := host.users[user.BitbucketUUID]; ok {
					return nil, fmt.Errorf("The users %s and %s have the same UUID (%s)", user.Name, userWithSameUUID.Name, user.BitbucketUUID)
				}
				host.users[user.BitbucketUUID] = user
			} else {
				log.Warningf("User %s has no set Bitbucket UUID", user.Name)
			}
		}
	}

	return host.users, nil
}

func (host *bitbucketCloud) GetRepositories() ([]Repository, error) {
	users, err := host.GetUsers()
	if err != nil {
		return nil, fmt.Errorf("Error fetching users from Bitbucket: %v", err)
	}

	repositories := []Repository{}
	for _, repositoryName := range host.repositoryNames {
		splitRepository := strings.Split(repositoryName, "/")
		owner, slug := splitRepository[0], splitRepository[1]
		pullRequests, err := host.getPullRequests(owner, slug, users)
		if err != nil {
			return nil, fmt.Errorf("Caught an error while describing pull requests: %v", err)
		}
		repository := NewRepository(host, repositoryName, fmt.Sprintf("https://bitbucket.org/%v", repositoryName), pullRequests)
		repositories = append(repositories, repository)
	}
	return repositories, nil
}

func (host *bitbucketCloud) getTeamMembers(team string) ([]bitbucketTeamMember, error) {
	listedMembers := &struct {
		Values []bitbucketTeamMember
	}{}
	var (
		err      error
		response interface{}
	)
	if err := try.Do(func(attempt int) (bool, error) {
		response, err = host.client.GetTeamMembers(team)
		if err != nil {
			log.Warn("Failed to fetch Bitbucket users. Waiting 10 seconds")
			secondsToSleep, _ := strconv.Atoi(utilities.GetEnv("BITBUCKET_RETRY_DELAY", "10"))
			time.Sleep(time.Duration(secondsToSleep) * time.Second)
		}
		return attempt < 5, err
	}); err != nil {
		return nil, fmt.Errorf("Error fetching members from team %s: %v", team, err)
	}
	if err := mapstructure.Decode(response, &listedMembers); err != nil {
		return nil, fmt.Errorf("Error parsing team members from bitbucket response: %v", err)
	}

	return listedMembers.Values, nil
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

func normalizeName(name string) string {
	transformChain := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	result, _, _ := transform.String(transformChain, name)
	return strings.ToLower(regexp.MustCompile("[^A-Za-z]").ReplaceAllString(result, ""))
}
