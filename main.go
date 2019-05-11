package main

import (
	"github.com/julienduchesne/pull-request-reminder/hosts"
	"github.com/julienduchesne/pull-request-reminder/messages"
)

func main() {

	repositoriesNeedingAction := []*hosts.Repository{}
	for _, host := range hosts.GetHosts() {
		for _, repository := range host.GetRepositories() {
			if repository.HasPullRequestsToDisplay() {
				repositoriesNeedingAction = append(repositoriesNeedingAction, repository)
			}
		}
	}
	for _, handler := range messages.GetHandlers() {
		handler.Notify(repositoriesNeedingAction)
	}
}
