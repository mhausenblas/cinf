# cinf

[![Go Report Card](https://goreportcard.com/badge/github.com/mhausenblas/cinf)](https://goreportcard.com/report/github.com/mhausenblas/cinf)

This is `cinf`, short for container info, a command line tool to view namespaces and cgroups, the stuff that makes up Linux containers such as Docker, rkt/appc, or OCI/runc. It might be useful for low-level container prodding, when you need to understand what's going on under the hood. Read more here: [Containers are a lie](https://medium.com/@mhausenblas/containers-are-a-lie-2521afda1f81) …

Contents:

- [Install](#install) `cinf`
- [Use](#use) `cinf` to
  - [see all namespaces](#to-see-all-namespaces)
  - dig into a [namespace](#to-dig-into-a-namespace)
  - dig into a [cgroup](#to-dig-into-a-cgroup)
  - dig into a [process](#to-dig-into-a-process) 
  - [monitor a process](#to-monitor-a-process)
- [CLI reference](#cli-reference)
- [Background](#background) on namespaces and cgroups

## Install

Simply download the Linux binary:

    $ curl -s -L https://github.com/mhausenblas/cinf/releases/download/v0.4.0-alpha/cinf -o cinf
    $ sudo mv cinf /usr/local/bin
    $ sudo chmod +x /usr/local/bin/cinf

Or build from source (note that you'll get the latest, experimental version via this method):

    $ go get github.com/olekukonko/tablewriter
    $ go get github.com/mhausenblas/cinf
    $ go get github.com/buger/goterm
    $ GOOS=linux go build
    $ godoc -http=":6060"

Note that the package docs are also available [online](https://godoc.org/github.com/mhausenblas/cinf/namespaces).

## Use

The following sections show basic usage. For a complete end-to-end usage, see the [walkthrough](walkthrough.md).

Note that if you want to see detailed debug messages, you can do that via a `DEBUG` environment variable, like so: `sudo DEBUG=true cinf`.

### To see all namespaces

To list all available namespaces and a summary of how many processes are in them along with the user IDs and the top-level command line executed, simply do the following:

    $ sudo cinf
    
    NAMESPACE   TYPE  NPROCS  USERS                     CMD
    
    4026532396  pid   1       1000                      sleep10000
    4026532398  net   1       1000                      sleep10000
    4026531837  user  109     0,1,101,102,104,106,1000  /sbin/init
    4026532196  pid   2       0,104                     nginx: master proces
    4026532293  mnt   1       0                         md5sum/dev/urandom
    4026532298  net   1       0                         md5sum/dev/urandom
    4026532393  mnt   1       1000                      sleep10000
    4026532296  pid   1       0                         md5sum/dev/urandom
    4026531840  mnt   104     0,1,101,102,106,1000      /sbin/init
    4026531839  ipc   105     0,1,101,102,106,1000      /sbin/init
    4026531836  pid   105     0,1,101,102,106,1000      /sbin/init
    4026532193  mnt   2       0,104                     nginx: master proces
    4026532198  net   2       0,104                     nginx: master proces
    4026531838  uts   105     0,1,101,102,106,1000      /sbin/init
    4026532194  uts   2       0,104                     nginx: master proces
    4026532294  uts   1       0                         md5sum/dev/urandom
    4026532394  uts   1       1000                      sleep10000
    4026532395  ipc   1       1000                      sleep10000
    4026531956  net   105     0,1,101,102,106,1000      /sbin/init
    4026531856  mnt   1       0
    4026532195  ipc   2       0,104                     nginx: master proces
    4026532295  ipc   1       0                         md5sum/dev/urandom

### To dig into a namespace

Assuming we're interested in more information on namespace `4026532398`, we would do the following:

    $ sudo cinf --namespace 4026532193

### To dig into a cgroup

Let's dig into a specific cgroup (with hierarchy ID `3`) of a process (with PID `27681`):

    $ sudo cinf --cgroup 27681:3

### To dig into a process

It is also possible to list the namespaces a specific process is in:

    $ sudo cinf --pid 27681

### To monitor a process

The interactive, `top` like mode of `cinf` is as follows. Let's say we want to monitor the control files `memory.usage_in_bytes`, `cpuacct.usage`, and `blkio.throttle.io_service_bytes` for process with PID `27681`: 

    $ sudo cinf --mon 27681:memory.usage_in_bytes,cpuacct.usage,blkio.throttle.io_service_bytes

Note that a more detailed usage description is available via the [walkthrough](walkthrough.md).

## CLI reference

There are three arguments you can provide to `cinf`, to dig into specific aspects of a namespace, cgroup, or process:

- `--namespace $NAMESPACE_ID` … List details about namespace with provided ID.
- `--cgroup $PID:$CGROUP_HIERARCHY_ID` … List details of a cgroup a process belongs to.
- `--pid $PID` … List namespaces the process with provided process ID is in.
- `--mon $PID:$CF1,$CF2,…` … Monitor process with provided process ID and the control files specified.

The meaning of the output columns is as follows:

- Overview (without arguments):
  - `NAMESPACE` … the namespace ID
  - `TYPE` … the type of namespace, see also [explanation of the namespaces](#overview-on-linux-namespaces-and-cgroups) below
  - `NPROCS` … number of processes in the namespace
  - `USERS` … user IDs in the namespace
  - `CMD` … command line of the root process
- Detailed namespace view (`--namespace`):
  - `PID` … process ID
  - `PPID` … process ID of parent
  - `NAME` … process name
  - `CMD` … process command line
  - `NTHREADS`… number of threads
  - `CGROUPS` … summary of the attached cgroups
  - `STATE` … process state
- Detailed cgroups view (`--cgroup`):
  - `CONTROLFILE` … the name of the control file, see also [cgroup man pages](#reading-material) below
  - `VALUE` … the content of the control file
- Detailed process view (`--pid`):
  - `NAMESPACE` … the namespace ID
  - `TYPE` … the type of namespace
- Monitor process view (`--mon`):
  - `PID` … process ID
  - `PPID` … process ID of parent
  - `UIDS` … real, effective, saved set, filesystem user ID
  - `STATE` … process state
  - `NAMESPACE` … the IDs of the namespaces the process is in  
  - `CONTROLFILE` … the name of the control file, see also [cgroup man pages](#reading-material) below
  - `VALUE` … the content of the control file

## Background

I developed `cinf` because existing tools like `systemd-cgtop` or `lsns` didn't do what I wanted. 
Also, I needed it for educational purposes (like training sessions, etc.). Note that the output format `cinf` uses is modelled after [lsns](http://karelzak.blogspot.ie/2015/12/lsns8-new-command-to-list-linux.html), so kudos to Karel for the inspiration.

If you want to learn more about container building blocks such as namespaces and cgroups, check out [containerz.info](http://containerz.info/).
