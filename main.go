package main

import (
	"log"

	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/julienduchesne/pull-request-reminder/hosts"
	"github.com/julienduchesne/pull-request-reminder/messages"
)

func main() {
	configReader, err := config.NewReader()
	if err != nil {
		log.Fatalf("Error while initializing the configuration reader: %v", err)
	}
	config, err := configReader.ReadConfig()
	if err != nil {
		log.Fatalf("Error while reading the configuration: %v", err)
	}
	for _, team := range config.Teams {
		repositories := getRepositoriesNeedingAction(hosts.GetHosts(team))
		handleRepositories(messages.GetHandlers(team), repositories)
	}
}

func getRepositoriesNeedingAction(teamHosts []hosts.Host) []*hosts.Repository {
	repositoriesNeedingAction := []*hosts.Repository{}
	for _, host := range teamHosts {
		for _, repository := range host.GetRepositories() {
			if repository.HasPullRequestsToDisplay() {
				repositoriesNeedingAction = append(repositoriesNeedingAction, repository)
			}
		}
	}
	return repositoriesNeedingAction
}

func handleRepositories(handlers []messages.MessageHandler, repositories []*hosts.Repository) {
	if len(repositories) > 0 {
		for _, handler := range handlers {
			handler.Notify(repositories)
		}
	}
}
