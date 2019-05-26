package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/julienduchesne/pull-request-reminder/hosts"
	"github.com/julienduchesne/pull-request-reminder/messages"
	"github.com/stretchr/testify/assert"
)

const (
	testRepositoryName           = "TestRepository"
	testRepositoryWithoutPRsName = "BadRepository"
)

func TestCallMainWithMinimalConfig(t *testing.T) {
	tempDir := os.TempDir()
	configPath := path.Join(tempDir, "config_file")
	ioutil.WriteFile(configPath, []byte("{}"), 0644)

	if oldValue := os.Getenv("PRR_CONFIG"); oldValue != "" {
		defer os.Setenv("PRR_CONFIG", oldValue)
	} else {
		defer os.Unsetenv("PRR_CONFIG")
	}
	os.Setenv("PRR_CONFIG", configPath)
	main()
}

func TestGetRepositories(t *testing.T) {
	t.Parallel()

	repositories := getRepositoriesNeedingAction([]hosts.Host{&mockHost{}})
	assert.Len(t, repositories, 1)
	assert.Equal(t, testRepositoryName, repositories[0].Name)
}

func TestHandleRepositories(t *testing.T) {
	t.Parallel()

	testRepository := &hosts.Repository{Name: testRepositoryName}
	handleRepositories([]messages.MessageHandler{&mockMessageHandler{t: t}}, []*hosts.Repository{testRepository})
}

type mockHost struct{}

func (host *mockHost) GetName() string {
	return "mock"
}

func (host *mockHost) GetUsers() []string {
	return []string{"mock"}
}

func (host *mockHost) GetRepositories() []*hosts.Repository {
	return []*hosts.Repository{
		&hosts.Repository{
			Name: testRepositoryName,
			ReadyToMergePullRequests: []*hosts.PullRequest{
				&hosts.PullRequest{},
			},
		},
		&hosts.Repository{
			Name: testRepositoryWithoutPRsName,
		},
	}
}

type mockMessageHandler struct {
	t *testing.T
}

func (handler *mockMessageHandler) Notify(repositories []*hosts.Repository) error {
	assert.Equal(handler.t, testRepositoryName, repositories[0].Name)
	return nil
}
