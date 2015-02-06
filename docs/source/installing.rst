.. Copyright 2015 tsuru-client authors. All rights reserved.
   Use of this source code is governed by a BSD-style
   license that can be found in the LICENSE file.

Installing
==========

There are several ways to install `tsuru-client`:

- `Downloading binaries (Mac OS X and Linux)`_
- `Using homebrew (Mac OS X only)`_
- `Using the PPA (Ubuntu only)`_
- `Using AUR (ArchLinux only)`_
- `Build from source (Linux and Mac OS X)`_

Downloading binaries (Mac OS X and Linux)
-----------------------------------------

We provide pre-built binaries for OS X and Linux, only for the amd64
architecture. You can download these binaries directly from the releases page
of the project:

    * tsuru: https://github.com/tsuru/tsuru-client/releases

Using homebrew (Mac OS X only)
------------------------------

If you use Mac OS X and `homebrew <http://mxcl.github.com/homebrew/>`_, you may
use a custom tap to install ``tsuru``. First you need to add the tap:

.. highlight:: bash

::

    $ brew tap tsuru/homebrew-tsuru

Now you can install tsuru:

.. highlight:: bash

::

    $ brew install tsuru

Whenever a new version of any of tsuru's clients is out, you can just run:

.. highlight:: bash

::

    $ brew update
    $ brew upgrade tsuru

For more details on taps, check `homebrew documentation
<https://github.com/Homebrew/homebrew/wiki/brew-tap>`_.

**NOTE:** tsuru requires Go 1.2 or higher. Make sure you have the last version
of Go installed in your system.

Using the PPA (Ubuntu only)
---------------------------

Ubuntu users can install tsuru clients using ``apt-get`` and the `tsuru PPA
<https://launchpad.net/~tsuru/+archive/ppa>`_. You'll need to add the PPA
repository locally and run an ``apt-get update``:

.. highlight:: bash

::

    $ sudo apt-add-repository ppa:tsuru/ppa
    $ sudo apt-get update

Now you can install tsuru's clients:

.. highlight:: bash

::

    $ sudo apt-get install tsuru-client

Using AUR (ArchLinux only)
--------------------------

Archlinux users can build and install tsuru client from AUR repository,
Is needed to have installed `yaourt <http://archlinux.fr/yaourt-en>`_ program.

You can run:


.. highlight:: bash

::

    $ yaourt -S tsuru

Build from source (Linux and Mac OS X)
--------------------------------------

.. note::

    If you're feeling adventurous, you can try it on other systems, like
    FreeBSD, OpenBSD or even Windows. Please let us know about your progress!

`tsuru client source <https://github.com/tsuru/tsuru-client>`_ is written in `Go
<http://golang.org>`_, so before installing tsuru from source, please make sure
you have `installed and configured Go <http://golang.org/doc/install>`_.

With Go installed and configured, you can use ``go get`` to install any of
tsuru's clients:

.. highlight:: bash

::

    $ go get github.com/tsuru/tsuru-client/tsuru
