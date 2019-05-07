package hosts

type Repository struct {
	Link string
	Host Host
	Name string

	OpenPullRequests          []PullRequest
	ReadyToMergePullRequests  []PullRequest
	ReadyToReviewPullRequests []PullRequest
}

func NewRepository(host Host, name, link string) *Repository {
	return &Repository{
		Link:                      link,
		Name:                      name,
		Host:                      host,
		OpenPullRequests:          []PullRequest{},
		ReadyToMergePullRequests:  []PullRequest{},
		ReadyToReviewPullRequests: []PullRequest{},
	}
}

func (repository *Repository) HasPullRequestsToDisplay() bool {
	return len(repository.ReadyToMergePullRequests)+len(repository.ReadyToReviewPullRequests) > 0
}

type Host interface {
	GetName() string
	GetRepositories() []*Repository
	GetUsers() []string
}

type PullRequest interface {
	GetDescription() string
	GetLink() string
	GetTitle() string
	IsApproved(teamUsernames []string) bool
	IsFromOneOfUsers(usernames []string) bool
	IsWIP() bool
	Reviewers(teamUsernames []string) []PullRequestParticipant
}

type User interface {
	GetUsername() string
}

type PullRequestParticipant interface {
	User
	HasApproved() bool
}

func GetHosts() []Host {
	hosts := []Host{}
	if bitbucketCloud := newBitbucketCloud(); bitbucketCloud != nil {
		hosts = append(hosts, bitbucketCloud)
	}
	return hosts
}
