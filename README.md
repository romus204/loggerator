# Loggerator 

Loggerator is a flexible, configurable log translator from a Kubernetes cluster to a telegram messenger

---
## Features
- Flexible filters based on regular expressions for selecting logs
- Flexible filters based on regular expressions for replacing sensitive data
- Support for streaming logs from multiple containers in parallel
- Minimum external dependencies
- Support for topics in telegram
- Environment variable substitution in configuration

> Note: Telegram has a limit of 20 messages per minute for a bot to send to a single chat. This limitation is taken into account here.
> Therefore, if you plan to handle a large number of errors that do not fit into this limit, you can create several bots, launch an instance of the application for each of them and divide the containers between them.

## Quick start
- With docker 
```shell
docker run -v /Users/romus204/projects/loggerator/config/config.yaml:/loggerator/config.yaml -v /Users/romus204/.kube/other_configs/prod.yaml:/loggerator/prod.yaml -d --name loggerator --restart always romus204/loggerator:latest
```

- From source
    - build app with `go buid`
    - run it with flag `-config path_to_config`

## Configuration

### Environment Variables
You can use environment variables in your configuration by using the `${ENV_VAR_NAME}` syntax. The application will automatically substitute these values with the corresponding environment variables at startup.

Example:
```yaml
telegram:
  token: "${TELEGRAM_BOT_TOKEN}"  # Will be replaced with the value of TELEGRAM_BOT_TOKEN env var
kube:
  config: "${KUBE_CONFIG_PATH}"   # Will be replaced with the value of KUBE_CONFIG_PATH env var
  namespace: "${KUBE_NAMESPACE}"  # Will be replaced with the value of KUBE_NAMESPACE env var
```

If an environment variable is not set, the original `${ENV_VAR_NAME}` string will be preserved in the configuration.

### Config example

```yaml
telegram:
  token: "${TELEGRAM_BOT_TOKEN}" # telegram bot token
  chat: -100123123 # chat id (if you use topics, you need to add -100 to the id)
  topics: # optional, if u use telegram chat as a forum
     container-name-1: 41
     container-name-2: 38
     container-name-3: 22
     container-name-4: 19
     container-name-5: 56

kube:
  config: "${KUBE_CONFIG_PATH}" # path to kube config 
  namespace: "${KUBE_NAMESPACE}"
  replacements: # Substitutions in the final text of logs, for example, hides the real token of the user who made the request 
  - target: '\bBearer eyJ[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+\b'
    replacement: "Bearer _"
  filter: # filter for logs, support regexp (only rows containing these filters will be sent)
    - '.*"level":"error".*'
    - '.*panic.*'
  target: # Pods for stream logs
    - pod: "^pod-name-1-production-*" # pod name, also support regexp
      container: # container names
        - "container-name-1" 
        - "container-name-2" 
        - "container-name-3" 
        - "container-name-4" 
    - pod: "^pod-name-2-production-*" 
      container:
        - "container-name-5" 
```
