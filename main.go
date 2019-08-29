package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/julienduchesne/pull-request-reminder/config"
	"github.com/julienduchesne/pull-request-reminder/hosts"
	"github.com/julienduchesne/pull-request-reminder/messages"
)

func main() {
	configReader, err := config.NewReader()
	if err != nil {
		log.WithError(err).Fatalln("Error while initializing the configuration reader")
	}
	config, err := configReader.ReadConfig()
	if err != nil {
		log.WithError(err).Fatalln("Error while reading the configuration")
	}
	for _, team := range config.Teams {
		repositories := getRepositoriesNeedingAction(hosts.GetHosts(team))
		if err = handleRepositories(messages.GetHandlers(team), repositories); err != nil {
			log.WithError(err).Fatalln("Error while handling messages")
		}
	}
}

func getRepositoriesNeedingAction(teamHosts []hosts.Host) []hosts.Repository {
	repositoriesNeedingAction := []hosts.Repository{}
	for _, host := range teamHosts {
		repositories, err := host.GetRepositories()
		if err != nil {
			log.WithError(err).Fatalf("Error while fetching repositories from %s", host.GetName())
		}
		for _, repository := range repositories {
			if repository.HasPullRequestsToDisplay() {
				repositoriesNeedingAction = append(repositoriesNeedingAction, repository)
			}
		}
	}
	return repositoriesNeedingAction
}

func handleRepositories(handlers []messages.MessageHandler, repositories []hosts.Repository) error {
	if len(repositories) > 0 {
		for _, handler := range handlers {
			if err := handler.Notify(repositories); err != nil {
				return err
			}
		}
	}
	return nil
}
