package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/golang/mock/gomock"

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepository := hosts.NewMockRepository(ctrl)
	mockRepository.EXPECT().HasPullRequestsToDisplay().Return(true).AnyTimes()
	mockRepository.EXPECT().GetName().Return(testRepositoryName).AnyTimes()
	mockRepositoryWithoutPRs := hosts.NewMockRepository(ctrl)
	mockRepositoryWithoutPRs.EXPECT().HasPullRequestsToDisplay().Return(false).AnyTimes()
	mockRepositoryWithoutPRs.EXPECT().GetName().Return(testRepositoryWithoutPRsName).AnyTimes()

	mockHost := hosts.NewMockHost(ctrl)
	mockHost.EXPECT().GetRepositories().Return([]hosts.Repository{mockRepository, mockRepositoryWithoutPRs}, nil)

	repositories := getRepositoriesNeedingAction([]hosts.Host{mockHost})
	assert.Len(t, repositories, 1)
	assert.Equal(t, testRepositoryName, repositories[0].GetName())
}

func TestHandleRepositories(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	testRepository := &hosts.RepositoryImpl{Name: testRepositoryName}
	repositories := []hosts.Repository{testRepository}

	mockMessageHandler := messages.NewMockMessageHandler(ctrl)
	mockMessageHandler.EXPECT().Notify(repositories).Times(1)

	handleRepositories([]messages.MessageHandler{mockMessageHandler}, repositories)
}
