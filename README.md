# pull-request-reminder
[![Build Status](https://travis-ci.org/julienduchesne/pull-request-reminder.svg?branch=master)](https://travis-ci.org/julienduchesne/pull-request-reminder)
[![codecov](https://codecov.io/gh/julienduchesne/pull-request-reminder/branch/master/graph/badge.svg)](https://codecov.io/gh/julienduchesne/pull-request-reminder)
[![Go Report Card](https://goreportcard.com/badge/github.com/julienduchesne/pull-request-reminder)](https://goreportcard.com/report/github.com/julienduchesne/pull-request-reminder)

Open source pull request reminder
* Fetches pull requests from the supported hosts
* Finds out which ones still need approvals and which ones are ready to merge 
* Posts to the configured messaging handlers (only Slack for now)

### Supported hosts
* Github
* Bitbucket

### Supported message handlers
* Slack
    * Posts to the given channel a list of all PRs still needing approvals and pings the owner when a PR is ready to merge  
    * Alternatively, sends personalized messages to all the concerned team members (those who need to act on a PR)  
![Slack](https://github.com/julienduchesne/pull-request-reminder/raw/master/slack.png)

## Configuration

### Configuration file
This app supports a configuration file with following format (JSON or YAML)
```js
{
    "teams":[
        {
            "name":"my-team",
            "age_before_notifying": "24h", // If set, will ignore PRs that have been created for less than the given time (when seeking approvals) and will ignore PRs that have been stale for less than the given time when they have been approved (when waiting for merge)
            "number_of_approvals": 1, // Number of approvals needed for a PR to be considered approved (Ignores the author's approval). Defaults to 1
            "review_pr_from_non_members": true, // If not set, PRs to the listed repositories will be ignored if they are not authored by one of the team members
            "hosts": {
                "bitbucket":{
                    "repositories":[
                        "owner/repo1",
                        "owner/repo2"
                    ],
                    "username":"user",
                    "password":"app_password",
                    "team": "my_team",
                    "find_users_in_team": true // If this attribute and `team` is set, user UUIDs will be found from the user name. An error will be raised if there is more than one match for a single user. To fix that issues, the user UUID must be set manually.
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
                    "token":"xoxb-abcd",
                    "message_users_individually": true, // If set, will send a personalized message to all the concerned team members (those who need to act on a PR)
                    "channel": "#my_channel" // If set, will send an summary message to the given channel
                }
            },
            "users":[
                {
                    "name":"John Doe",
                    "bitbucket_uuid":"{260ae11c-d3c9-4d9b-b1b0-54d3914b6c24}",
                    "github_username":"johndoe",
                    "slack_username":"@jdoe"
                }
            ]
        }
    ]
}
```

### Other Features
#### Marking pull requests as work in progress
Anytime a pull request is not ready to review, simply add `WIP` somewhere in its title. PRs marked with `WIP` are ignored by this tool

### To run
* Run the docker image located here: https://hub.docker.com/r/julienduchesne/pull-request-reminder
* Build the executable using `go build` and run it

### Environment
Credentials can also be set globally as environment variables
- **PRR_BITBUCKET_USERNAME**
- **PRR_BITBUCKET_PASSWORD**
- **PRR_GITHUB_TOKEN**
- **PRR_SLACK_TOKEN**

You can also set the config file path with the following environment variable
- **PRR_CONFIG**: This path can either be a path to a file on the local file system or a S3 path (s3://bucket/key)

You can set the logging level with the **PRR_LOG_LEVEL** environment variable. Messages sent to Slack will only be logged if you set this to `DEBUG`
