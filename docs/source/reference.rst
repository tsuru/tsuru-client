Reference
~~~~~~~~~

Managing remote tsuru server endpoints
======================================

.. tsuru-command:: target

.. tsuru-command:: target-add
   :title: Add a new target
.. tsuru-command:: target-list
   :title: List existing targets
.. tsuru-command:: target-set
   :title: Set a target as current
.. tsuru-command:: target-remove
   :title: Removes an existing target

Check current version
=====================

.. tsuru-command:: version

Authentication
==============

.. tsuru-command:: user-create
   :title: Create a user
.. tsuru-command:: user-remove
   :title: Remove your user from tsuru server
.. tsuru-command:: user-info
   :title: Retrieve information about the current user
.. tsuru-command:: login
   :title: Login
.. tsuru-command:: logout
   :title: Logout
.. tsuru-command:: change-password
   :title: Change user's password
.. tsuru-command:: reset-password
   :title: Resets user's password
.. tsuru-command:: token-show
   :title: Show current valid API token
.. tsuru-command:: token-regenerate
   :title: Regenerate API token

Team management
===============

.. tsuru-command:: team-create
   :title: Create a new team
.. tsuru-command:: team-remove
   :title: Remove a team from tsuru
.. tsuru-command:: team-list
   :title: List teams current user is member

Authorization
=============

.. tsuru-command:: permission-list
   :title: List all available permissions
.. tsuru-command:: role-add
   :title: Create a new role
.. tsuru-command:: role-list
   :title: List all created roles
.. tsuru-command:: role-info
   :title: Info about specific role
.. tsuru-command:: role-permission-add
   :title: Add a permission to a role
.. tsuru-command:: role-permission-remove
   :title: Remove a permission from a role
.. tsuru-command:: role-assign
   :title: Assign a role to a user
.. tsuru-command:: role-dissociate
   :title: Dissociate a role from a user

Applications
============

Guessing application names
--------------------------

Some application related commands that are described below have the optional
parameter ``-a/--app``, used to specify the name of the application.

If this parameter is omitted, tsuru will try to *guess* the application's name
based on the git repository's configuration. It will try to find a remote labeled
**tsuru**, and parse its URL.

If no remote named **tsuru** is found, tsuru will try to use the current directory
name as the application's name.


.. tsuru-command:: platform-list
   :title: List of available platforms

.. tsuru-command:: plan-list
   :title: List of available plans

.. tsuru-command:: app-create
   :title: Create an application
.. tsuru-command:: app-plan-change
   :title: Change the application plan
.. tsuru-command:: app-remove
   :title: Remove an application
.. tsuru-command:: app-list
   :title: List your applications
.. tsuru-command:: app-info
   :title: Display information about an application
.. tsuru-command:: app-log
   :title: Show logs of an application
.. tsuru-command:: app-stop
   :title: Stop an application
.. tsuru-command:: app-start
   :title: Start an application
.. tsuru-command:: app-restart
   :title: Restart an application
.. tsuru-command:: app-swap
   :title: Swap the routing between two applications
.. tsuru-command:: unit-add
   :title: Add new units to an application
.. tsuru-command:: unit-remove
   :title: Remove units from an application
.. tsuru-command:: app-team-owner-set
   :title: Change an application team owner
.. tsuru-command:: app-grant
   :title: Allow a team to access an application
.. tsuru-command:: app-revoke
   :title: Revoke a team's access to an application
.. tsuru-command:: app-run
   :title: Run an arbitrary command in application's containers
.. tsuru-command:: app-shell
   :title: Open a shell to an application's container
.. tsuru-command:: app-deploy
   :title: Deploy
.. tsuru-command:: app-deploy-list
   :title: List deploys
.. tsuru-command:: app-deploy-rollback
   :title: Rollback deploy


Public Keys
===========

.. tsuru-command:: key-add
   :title: Add SSH public key
.. tsuru-command:: key-remove
   :title: Remove SSH public key
.. tsuru-command:: key-list
   :title: List SSH public keys


Services
========

.. tsuru-command:: service-list
   :title: List available services and instances
.. tsuru-command:: service-add
   :title: Create a new service instance
.. tsuru-command:: service-remove
   :title: Remove a service instance
.. tsuru-command:: service-info
   :title: Display information about a service
.. tsuru-command:: service-status
   :title: Check if a service instance is up
.. tsuru-command:: service-doc
   :title: Check if a service instance is up
.. tsuru-command:: service-bind
   :title: Bind an application to a service instance
.. tsuru-command:: service-unbind
   :title: Unbind an application from a service instance
.. tsuru-command:: service-instance-grant
   :title: Grant access to a team in service instance
.. tsuru-command:: service-instance-revoke
   :title: Revoke access to a team in service instance


Environment variables
=====================

Applications running on tsuru should use environment variables to handle
configurations. As an example, if you need to connect with a third party service
like twitter’s API, your application is going to need things like an ``api_key``.

In tsuru, the recommended way to expose these values to applications is using
environment variables. To make this easy, tsuru provides commands to set and get
environment variables in a running application.

.. tsuru-command:: env-set
   :title: Set environment variables
.. tsuru-command:: env-get
   :title: Show environment variables
.. tsuru-command:: env-unset
   :title: Unset environment variables


Plugin management
=================

Plugins allow extending tsuru client's functionality. Plugins are executables
existing in ``$HOME/.tsuru/plugins``.

Installing a plugin
-------------------

There are two ways to install. The first way is to manually copy your plugin to
``$HOME/.tsuru/plugins``.  The other way is to use ``tsuru plugin-install``
command.


.. tsuru-command:: plugin-install
   :title: Install a plugin
.. tsuru-command:: plugin-list
   :title: List installed plugins
.. tsuru-command:: plugin-remove
   :title: Remove a plugin

Executing a plugin
------------------

To execute a plugin just follow the pattern ``tsuru <plugin-name> <args>``:

.. highlight:: bash

::

    $ tsuru <plugin-name>
    <plugin-output>

CNAME management
================

.. tsuru-command:: cname-add
   :title: Add a CNAME to the app
.. tsuru-command:: cname-remove
   :title: Remove a CNAME from the app

Pool
====

.. tsuru-command:: pool-list
   :title: List available pool

.. tsuru-command:: app-pool-change
   :title: Change an app's pool

Events
======

.. tsuru-command:: event-list
   :title: List all events

.. tsuru-command:: event-info
   :title: Show detailed information about an event

.. tsuru-command:: event-cancel
   :title: Cancel an event

Installer
=========

.. tsuru-command:: install
   :title: Install Tsuru and it's components

.. tsuru-command:: uninstall
  :title: Uninstall Tsuru and it's components
