# tsuru-client

![Go Version](https://img.shields.io/github/go-mod/go-version/tsuru/tsuru-client)
[![Actions Status](https://github.com/tsuru/tsuru-client/workflows/Go/badge.svg)](https://github.com/tsuru/tsuru-client/actions)
[![Coverage](https://tsuru.github.io/tsuru-client/coverage/badge.svg)](https://tsuru.github.io/tsuru-client/coverage/coverage.html)

tsuru is a command line for application developers on
[tsuru](https://github.com/tsuru/tsuru).

## Tsuru plugins

Tsuru plugins are the standard way to extend tsuru-client functionality transparently.
Installing and using a plugin is done with:
```
tsuru plugin install <plugin-name> <plugin-url>
tsuru <plugin-name> <any_sub_commands_or_flags...>
```

For developing a custom plugin, read about [Developing Tsuru Plugins](./pkg/cmd/plugin.md).
