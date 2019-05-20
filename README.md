# pull-request-reminder
[![Build Status](https://travis-ci.org/julienduchesne/pull-request-reminder.svg?branch=master)](https://travis-ci.org/julienduchesne/pull-request-reminder)
[![codecov](https://codecov.io/gh/julienduchesne/pull-request-reminder/branch/master/graph/badge.svg)](https://codecov.io/gh/julienduchesne/pull-request-reminder)
[![Go Report Card](https://goreportcard.com/badge/github.com/julienduchesne/pull-request-reminder)](https://goreportcard.com/report/github.com/julienduchesne/pull-request-reminder)

Open source pull request reminder

## Configuration

### Configuration file
This app supports a configuration file with following format (JSON or YAML)
```json
{
    "teams":[
        {
            "name":"my-team",
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
            },
            "slack":{
                "channel":"",
                "token":"xoxb-abcd"
            },
            "users":[
                {
                    "name":"John Doe",
                    "bitbucket_username":"jdoe",
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
Simply run the executable, without any parameters