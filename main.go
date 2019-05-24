package main

import (
	"log"

	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/julienduchesne/pull-request-reminder/hosts"
	"github.com/julienduchesne/pull-request-reminder/messages"
)

func main() {
	configReader, err := config.NewConfigReader()
	if err != nil {
		log.Fatalf("Error while initializing the configuration reader: %v", err)
	}
	config, err := configReader.ReadConfig()
	if err != nil {
		log.Fatalf("Error while reading the configuration: %v", err)
	}
	for _, team := range config.Teams {
		repositoriesNeedingAction := []*hosts.Repository{}
		for _, host := range hosts.GetHosts(team) {
			for _, repository := range host.GetRepositories() {
				if repository.HasPullRequestsToDisplay() {
					repositoriesNeedingAction = append(repositoriesNeedingAction, repository)
				}
			}
		}
		if len(repositoriesNeedingAction) > 0 {
			for _, handler := range messages.GetHandlers(team) {
				handler.Notify(repositoriesNeedingAction)
			}
		}
	}
}
