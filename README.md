# tsuru-client

[![Actions Status](https://github.com/tsuru/tsuru-client/workflows/Go/badge.svg)](https://github.com/tsuru/tsuru-client/actions)
[![codecov](https://codecov.io/gh/tsuru/tsuru-client/branch/master/graph/badge.svg)](https://codecov.io/gh/tsuru/tsuru-client)

tsuru is a command line for application developers on
[tsuru](https://github.com/tsuru/tsuru).

## reporting issues

Please report issues to the
[tsuru/tsuru](https://github.com/tsuru/tsuru/issues) repository.


## Environment variables

The following environment variables can be used to configure the client:

### API configuration

* `TSURU_TARGET`: the tsuru API endpoint.
* `TSURU_TOKEN`: the tsuru API token.

### Other configuration

* `TSURU_CLIENT_FORCE_CHECK_UPDATES`: boolean on whether to force checking for
  updates. When `true`, it hangs if no response from remote server! (default: unset)
* `TSURU_CLIENT_LOCAL_TIMEOUT`: timeout for performing local non-critical operations
  (eg: writing preferences to `~/.tsuru/config.json`). (default: 1s)
* `TSURU_CLIENT_SELF_UPDATE_SNOOZE_DURATION`: snooze the self-updating process for
  the given duration. (default: 0s)
