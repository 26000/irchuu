OpenRC initscript and config for IRChuu~
---------------------

OpenRC is an init system used primarily on Gentoo and its derivatives.
This guide describes how to install and setup openrc scripts for single irchuu or for several instances.

#### `conf.d-irchuu` file
Contains daemon launch settings.

* `user` and `group` parameters set daemon user and group
* `data_home` and `config_home` are injected to irchuu's env as `XDG_DATA_HOME` and `XDG_CONFIG_HOME` respectively. If you want them to remain default, you may as well leave them blank
* `irchuu_exec` is path to executable. It must be set
* `irchuu_log` is path to the bot log file

#### Installation (single instance)
    1. Copy `init.d-irchuu` to `/etc/init.d/irchuu`
    2. Copy `conf.d-irchuu` to `/etc/conf.d/irchuu` and alter parameters in it if necessary

Now launch irchuu to check if everything is correct: `/etc/init.d/irchuu start`.

Don't forget to add the service to default runlevel, if you want irchuu to start automatically after reboots: `rc-update add irchuu default`.

#### Installation (multiple instances)
  1. Copy `init.d-irchuu` to `/etc/init.d/irchuu`
  2. Copy `conf.d-irchuu` to `/etc/conf.d/irchuu-instance-1`, `/etc/conf.d/irchuu-instance-2`, etc.
  3. Create symlinks to `/etc/init.d/irchuu` with paths `/etc/init.d/irchuu-instance-1`, `/etc/init.d/irchuu-instance-2`, etc.

On start, `/etc/init.d/irchuu-<instance>` will source file with respective name from `/etc/conf.d`, so pay attention to name: e.g. `/etc/init.d/irchuu-instance-1` will take variables values from `/etc/conf.d/irchuu-instance-1` file.

This way you can have several instances with different config files, data dirs, log files, or even irchuu versions.

Of course, instead of symlinks you can just copy original initscript, e.g. if you want to change something inside the script itself.

Also, don't forget to add services to default runlevel: `rc-update add irchuu-instance-1 default`, `rc-update add irchuu-instance-2 default`, etc.
