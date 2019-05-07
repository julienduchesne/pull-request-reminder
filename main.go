package main

import (
	"github.com/julienduchesne/pull-request-reminder/hosts"
	"github.com/julienduchesne/pull-request-reminder/messages"
	log "github.com/sirupsen/logrus"
)

func main() {

	repositoriesNeedingAction := []*hosts.Repository{}
	for _, host := range hosts.GetHosts() {
		for _, repository := range host.GetRepositories() {
			for _, pullRequest := range repository.OpenPullRequests {
				var logIgnoredPullRequest = func(message string) {
					log.Info(repository.Name, "->", pullRequest.GetLink(), " ignored because: ", message)
				}

				if !pullRequest.IsFromOneOfUsers(host.GetUsers()) {
					logIgnoredPullRequest("Not from one of the team's users")
					continue
				}
				if pullRequest.IsWIP() {
					logIgnoredPullRequest("Marked WIP")
					continue
				}
				if len(pullRequest.Reviewers(host.GetUsers())) == 0 {
					logIgnoredPullRequest("No reviewers")
					continue
				}

				if pullRequest.IsApproved(host.GetUsers()) {
					repository.ReadyToMergePullRequests = append(repository.ReadyToMergePullRequests, pullRequest)
				} else {
					repository.ReadyToReviewPullRequests = append(repository.ReadyToReviewPullRequests, pullRequest)
				}
			}
			if repository.HasPullRequestsToDisplay() {
				repositoriesNeedingAction = append(repositoriesNeedingAction, repository)
			}
		}
	}
	for _, handler := range messages.GetHandlers() {
		handler.Notify(repositoriesNeedingAction)
	}
}
