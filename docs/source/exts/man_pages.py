# -*- coding: utf-8 -*-
from datetime import datetime
from json import load
from sys import stdout
from subprocess import Popen
from subprocess import PIPE


# Tries to load command line options from file
data = None
try:
    with open("cmds.json", mode="r") as cmdsfile:
        data = load(cmdsfile)
except IOError:
    print >> stdout, "Error during file open. Try to run 'make docs' " \
        "before generating the tsuru man pages"
    raise

# Date of the man page creation
today = datetime.now().strftime("%d %B %Y")

# Issues url
issues = "https://github.com/tsuru/tsuru/issues"

# Manpage header comments
header = './" Manpage for tsuru project\n' \
         './" Refer to github.com/tsuru/tsuru-client to correct errors\n'

# Current Version
version_data = Popen(['tsuru', 'version'], stdout=PIPE)
version = version_data.communicate()[0].split()[-1]

# Manpage title
title = '.TH man 8 "%s" "%s" "tsuru man page"\n' % (today, version)

# Section NAME
name = '.SH NAME\n' \
       'tsuru \- open source Platform as a Service (PaaS)\n'

# Section SYNOPSIS
synopsis = '.SH SYNOPSIS\n' \
           'tsuru command [args]\n'

# Section DESCRIPTION
description = '.SH DESCRIPTION\n' \
              'tsuru is an extensible and open source Platform as a Service' \
              '(PaaS) that makes application deployments faster and easier.' \
              'tsuru is an open source polyglot cloud application platform' \
              '(PaaS). With tsuru, you don’t need to think about servers ' \
              'at all.' \
              'As an application developer, you can:\n' \
              '\tWrite apps in the programming language of your choice,\n' \
              '\tBack apps with add-on resources such as SQL and NoSQL ' \
              'databases, including memcached, redis, and many others.\n' \
              '\tManage apps using the tsuru command-line tool\n' \
              '\tDeploy apps using the Git revision control system'


# Command line options
options = ['.SH OPTIONS\n']

for key, values in data.items():
    desc = values.get('desc', "")
    usage = values.get('usage', "")
    options.append('.TP\n.B %s\n%s\n%s\n' % (key, desc, usage))
options_str = "".join(options).encode("utf-8")

# Section Bugs
bugs = '.SH BUGS\nComments and bug reports concerning tsuru project ' \
       'should be refered on %s\n' % (issues)

# man page file
with open("tsuru.8", mode="w") as manfile:
    manfile.write("%s%s%s%s%s%s%s" % (header, title, name,
        synopsis, description, options_str, bugs))
    print >> stdout, "tsuru man pages saved to %s" % (manfile.name)
