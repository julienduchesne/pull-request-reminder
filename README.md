# pull-request-reminder
[![Build Status](https://travis-ci.org/julienduchesne/pull-request-reminder.svg?branch=master)](https://travis-ci.org/julienduchesne/pull-request-reminder)
[![codecov](https://codecov.io/gh/julienduchesne/pull-request-reminder/branch/master/graph/badge.svg)](https://codecov.io/gh/julienduchesne/pull-request-reminder)
[![Go Report Card](https://goreportcard.com/badge/github.com/julienduchesne/pull-request-reminder)](https://goreportcard.com/report/github.com/julienduchesne/pull-request-reminder)

Open source pull request reminder

### Supported hosts
* Github
* Bitbucket

### Supported message handlers
* Slack

## Configuration

### Configuration file
This app supports a configuration file with following format (JSON or YAML)
```json
{
    "teams":[
        {
            "name":"my-team",
            "hosts": {
                "bitbucket":{
                    "repositories":[
                        "owner/repo1",
                        "owner/repo2"
                    ],
                    "username":"user",
                    "password":"app_password"
                },
                "github":{
                    "repositories":[
                        "account/repo1",
                        "account/repo2"
                    ],
                    "token":"mytoken"
                }
            },
            "messaging": {
                "slack":{
                    "channel":"",
                    "token":"xoxb-abcd"
                }
            },
            "users":[
                {
                    "name":"John Doe",
                    "bitbucket_uuid":"{260ae11c-d3c9-4d9b-b1b0-54d3914b6c24}",
                    "github_username":"johndoe",
                    "slack_username":"jdoe"
                }
            ]
        }
    ]
}
```


### Environment
Credentials can also be set globally as environment variables
- **PRR_BITBUCKET_USERNAME**
- **PRR_BITBUCKET_PASSWORD**
- **PRR_GITHUB_TOKEN**
- **PRR_SLACK_TOKEN**

You can also set the config file path with the following environment variable
- **PRR_CONFIG**: This path can either be a path to a file on the local file system or a S3 path (s3://bucket/key)

### To run
* Run the docker image located here: https://hub.docker.com/r/julienduchesne/pull-request-reminder
* Build the executable using `go build` and run it