package hosts

import (
	"fmt"
	"path/filepath"
	reflect "reflect"
	"regexp"
	"runtime"
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

type bitbucketRepository struct {
	Name    string
	Project struct {
		Key  string
		Name string
	}
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
	GetPullRequests(args ...string) (interface{}, error)
	GetRepositories(args ...string) (interface{}, error)
	GetTeamMembers(args ...string) (interface{}, error)
}

type bitbucketClientWrapper struct {
	client *bitbucket.Client
}

func (wrapper *bitbucketClientWrapper) GetPullRequests(args ...string) (interface{}, error) {
	return wrapper.client.Repositories.PullRequests.Get(&bitbucket.PullRequestsOptions{
		Owner:    args[0],
		RepoSlug: args[1],
		ID:       args[2],
	})
}
func (wrapper *bitbucketClientWrapper) GetRepositories(args ...string) (interface{}, error) {
	return wrapper.client.Repositories.ListForTeam(&bitbucket.RepositoriesOptions{Owner: args[0]})
}

func (wrapper *bitbucketClientWrapper) GetTeamMembers(args ...string) (interface{}, error) {
	return wrapper.client.Teams.Members(args[0])
}

type bitbucketCloud struct {
	config          *config.TeamConfig
	client          bitbucketClient
	repositoryNames []string
	projects        []string
	teamName        string
	users           map[string]config.User
}

func newBitbucketCloud(config *config.TeamConfig) *bitbucketCloud {
	bitbucketConfig := config.Hosts.Bitbucket
	return &bitbucketCloud{
		config:          config,
		client:          &bitbucketClientWrapper{client: bitbucket.NewBasicAuth(bitbucketConfig.Username, bitbucketConfig.Password)},
		repositoryNames: bitbucketConfig.Repositories,
		projects:        bitbucketConfig.Projects,
		teamName:        bitbucketConfig.Team,
	}

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
			if host.teamName == "" {
				return nil, fmt.Errorf("Bitbucket is set to find users in the team but the team name is not set")
			}
			if teamMembers, err = host.getTeamMembers(host.teamName); err != nil {
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
	log.Debug("Getting Bitbucket information")
	users, err := host.GetUsers()
	if err != nil {
		return nil, fmt.Errorf("Error fetching users from Bitbucket: %v", err)
	}

	repositoryNames := host.repositoryNames
	if len(host.projects) > 0 {
		newRepositoryNames, err := host.getRepositoriesFromProjects(host.projects)
		if err != nil {
			return nil, fmt.Errorf("Error fetching repositories from Bitbucket: %v", err)
		}
		repositoryNames = append(repositoryNames, newRepositoryNames...)
	}
	for index, repositoryName := range repositoryNames {
		if !strings.Contains(repositoryName, "/") {
			repositoryNames[index] = host.teamName + "/" + repositoryName
		}
	}

	repositories := []Repository{}
	for _, repositoryName := range utilities.Unique(repositoryNames) {
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

func (host *bitbucketCloud) getPullRequests(owner, repoSlug string, users map[string]config.User) ([]*PullRequest, error) {
	result := []*PullRequest{}

	listedPullRequests := &struct {
		Values []struct {
			ID int
		}
	}{}
	if err := host.callAPI(&listedPullRequests, host.client.GetPullRequests, owner, repoSlug, ""); err != nil {
		return nil, err
	}

	for _, listedPullRequest := range listedPullRequests.Values {
		var pullRequest bitbucketPullRequest
		if err := host.callAPI(&pullRequest, host.client.GetPullRequests, owner, repoSlug, strconv.Itoa(listedPullRequest.ID)); err != nil {
			return nil, err
		}
		result = append(result, pullRequest.ToGenericPullRequest(users))
	}

	return result, nil
}

func (host *bitbucketCloud) getRepositoriesFromProjects(projects []string) ([]string, error) {
	listedRepositories := &struct {
		Values []bitbucketRepository
	}{}
	if err := host.callAPI(&listedRepositories, host.client.GetRepositories, host.teamName); err != nil {
		return nil, err
	}
	names := []string{}
	for _, repository := range listedRepositories.Values {
		projectKey, projectName := strings.ToLower(repository.Project.Key), strings.ToLower(repository.Project.Name)
		for _, givenProject := range projects {
			givenProject = strings.ToLower(givenProject)
			if strings.Contains(givenProject, "/") {
				givenProject = strings.Split(givenProject, "/")[1]
			}
			if givenProject == projectKey || givenProject == projectName {
				names = append(names, repository.Name)
				break
			}
		}
	}
	return names, nil
}

func (host *bitbucketCloud) getTeamMembers(team string) ([]bitbucketTeamMember, error) {
	listedMembers := &struct {
		Values []bitbucketTeamMember
	}{}
	if err := host.callAPI(&listedMembers, host.client.GetTeamMembers, team); err != nil {
		return nil, err
	}
	return listedMembers.Values, nil
}

func (host *bitbucketCloud) callAPI(value interface{}, fn func(args ...string) (interface{}, error), args ...string) error {
	functionName := strings.Split(strings.TrimPrefix(filepath.Ext(runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()), "."), "-")[0]
	log.Debugf("Calling Bitbucket %s for %v", functionName, args)
	var (
		err      error
		response interface{}
	)
	if err := try.Do(func(attempt int) (bool, error) {
		response, err = fn(args...)
		if err != nil {
			log.Warnf("Failed to call %s. Waiting 10 seconds", functionName)
			secondsToSleep, _ := strconv.Atoi(utilities.GetEnv("BITBUCKET_RETRY_DELAY", "10"))
			time.Sleep(time.Duration(secondsToSleep) * time.Second)
		}
		return attempt < 5, err
	}); err != nil {
		return fmt.Errorf("Error calling Bitbucket %s for %v: %v", functionName, args, err)
	}
	if err := mapstructure.Decode(response, value); err != nil {
		return fmt.Errorf("Error parsing Bibucket response from %s for %v: %v", functionName, args, err)
	}
	return nil
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

func normalizeName(name string) string {
	transformChain := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	result, _, _ := transform.String(transformChain, name)
	return strings.ToLower(regexp.MustCompile("[^A-Za-z]").ReplaceAllString(result, ""))
}
