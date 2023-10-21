# Developing Tsuru Plugins

Tsuru plugins are the standard way to extend tsuru-client functionality transparently.

A tsuru-client plugin is any runnable file, located inside `~/.tsuru/plugins` directory.
It works by finding the runnable file (with or without extension) with that plugin name.

A simple working example:
```bash
cat > ~/.tsuru/plugins/myplugin.sh <<"EOF"
#!/bin/sh
echo "Hello from tsuru plugin ${TSURU_PLUGIN_NAME}!"
echo "  called with args: $@"
EOF

chmod +x ~/.tsuru/plugins/myplugin.sh

tsuru myplugin subcommands -flags

##### printed:
# Hello from tsuru plugin myplugin!
#   called with args: subcommands -flags
```

You may find available tsuru plugins on github, by searching for the topic [`tsuru-plugin`](https://github.com/topics/tsuru-plugin).
(If you are developing a plugin, please tag your github repo).

## Distributing a tsuru plugin

The best way to distribute a tsuru plugin is making it compatible with `tsuru plugin install`.
There are different approaches for distributing the plugin,
depending on the language used for building it.

### script-like single file
If the plugin is bundled as a **script-like single file** (eg: shell script, python, ruby, etc...)
you may make it available for download on a public URL.
The name of the file is irrelevant on this case.

### bundle of multiple files
If the plugin is bundled as **multiple files**, you should compact them inside a `.tar.gz` or `.zip` file,
and make it available for download on a public URL.
In this case, the file entrypoint must has the same name as the plugin (file extension is optional).
The CLI will call the binary at `~/.tsuru/plugins/myplugin/myplugin[.ext]`.

### compiled binary
If the plugin is bundled as a **compiled binary**, you should create a `manifest.json` file
(as defined on issue [#172](https://github.com/tsuru/tsuru-client/issues/172))
which tells where to download the appropriate binary:
```json
{
  "SchemaVersion": "1.0",
  "Metadata": {
    "Name": "<pluginName>",
    "Version": "<pluginVersion>"
  },
  "UrlPerPlatform": {
    "<os>/<arch>": "<os_arch_url>",
    ...
  }
}
```

Each supported os/arch (check the latest release), should be compacted as `.tar.gz` or `.zip` file.
All files (`manifest.json` and all compacted binaries) must available for download on public URLs.

An example of such a plugin, hosted on github, is the
[`rpaasv2` plugin](https://github.com/tsuru/rpaas-operator/issues/124),
installed using this [manifest.json](https://github.com/tsuru/rpaas-operator/releases/latest/download/manifest.json).

## Available ENV variables

When a plugin is called, the main tsuru-client passes some additional environment variables:

| env               | description                                            |
| ----------------- | ------------------------------------------------------ |
| TSURU_TARGET      | tsuru server url (eg: https://tsuru.io)                |
| TSURU_TOKEN       | tsuru authentication token                             |
| TSURU_VERBOSITY   | 0: default, 1: log requests, 2: log responses          |
| TSURU_FORMAT      | output format (json, table, etc...) as in [printer.OutputFormat](../printer/printer.go)    |
| TSURU_PLUGIN_NAME | name called from the main tsuru-client                 |
