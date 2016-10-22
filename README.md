# cinf

This is `cinf`, short for container info, a command line tool to view namespaces and cgroups, the stuff that makes up Linux containers such as Docker, rkt/appc, or OCI/runc. It might be useful for low-level container prodding, when you need to understand what's going on under the hood. Read more here: [Containers are a lie](https://medium.com/@mhausenblas/containers-are-a-lie-2521afda1f81) …

## Install

Simply download the Linux binary:

    $ curl -s -L https://github.com/mhausenblas/cinf/releases/download/v0.3.0-alpha/cinf -o cinf
    $ sudo mv cinf /usr/local/bin
    $ sudo chmod +x /usr/local/bin/cinf

Or build from source (note that you'll get the latest, experimental version via this method):

    $ go get github.com/olekukonko/tablewriter
    $ go get github.com/mhausenblas/cinf
    $ GOOS=linux go build
    $ godoc -http=":6060"

## Use

### To see all namespaces

To list all available namespaces and a summary of how many processes are in them along with the users and the top-level command line executed, simply do the following:

    $ sudo cinf
    
     NAMESPACE   TYPE  NPROCS  USER                      CMD
     
     4026532295  ipc   1       1000                      sleep10000
     4026532195  ipc   2       0,104                     nginx: master proces
     4026532393  mnt   1       0                         md5sum/dev/urandom
     4026531840  mnt   99      0,1,101,102,106,1000      /sbin/init
     4026531839  ipc   100     0,1,101,102,106,1000      /sbin/init
     4026531837  user  104     0,1,101,102,104,106,1000  /sbin/init
     4026532293  mnt   1       1000                      sleep10000
     4026532193  mnt   2       0,104                     nginx: master proces
     4026532194  uts   2       0,104                     nginx: master proces
     4026532396  pid   1       0                         md5sum/dev/urandom
     4026531956  net   100     0,1,101,102,106,1000      /sbin/init
     4026531856  mnt   1       0
     4026532196  pid   2       0,104                     nginx: master proces
     4026532198  net   2       0,104                     nginx: master proces
     4026531838  uts   100     0,1,101,102,106,1000      /sbin/init
     4026531836  pid   100     0,1,101,102,106,1000      /sbin/init
     4026532294  uts   1       1000                      sleep10000
     4026532296  pid   1       1000                      sleep10000
     4026532298  net   1       1000                      sleep10000
     4026532394  uts   1       0                         md5sum/dev/urandom
     4026532395  ipc   1       0                         md5sum/dev/urandom
     4026532398  net   1       0                         md5sum/dev/urandom

To dig into specific aspects of a namespace or cgroup, there are three arguments you can provide:

- `-namespace` … List details about namespace with provided ID.
- `-cgroup` … List details of a cgroup a process belongs to in format `CGROUP_HIERARCHY:PID`
- `-pid` … List namespaces the process with provided process ID is in.

Let's have a look at each of the options in the following.

### To dig into a namespace

Assuming we're interested in more information on namespace `4026532398`, we would do the following:

    $ sudo cinf -namespace 4026532398
    
     PID   PPID  NAME    CMD                 NTHREADS  CGROUPS                                                                                   STATE
     
     9422  9405  md5sum  md5sum/dev/urandom  1         11:name=systemd:/docker/c35f489277eb6e13842a24ea217a64035ac33b281b127c218b04744155aede26  R (running)
                                                       10:hugetlb:/docker/c35f489277eb6e13842a24ea217a64035ac33b281b127c218b04744155aede26
                                                       9:perf_event:/docker/c35f489277eb6e13842a24ea217a64035ac33b281b127c218b04744155aede26
                                                       8:blkio:/docker/c35f489277eb6e13842a24ea217a64035ac33b281b127c218b04744155aede26
                                                       7:freezer:/docker/c35f489277eb6e13842a24ea217a64035ac33b281b127c218b04744155aede26
                                                       6:devices:/docker/c35f489277eb6e13842a24ea217a64035ac33b281b127c218b04744155aede26
                                                       5:memory:/docker/c35f489277eb6e13842a24ea217a64035ac33b281b127c218b04744155aede26
                                                       4:cpuacct:/docker/c35f489277eb6e13842a24ea217a64035ac33b281b127c218b04744155aede26
                                                       3:cpu:/docker/c35f489277eb6e13842a24ea217a64035ac33b281b127c218b04744155aede26
                                                       2:cpuset:/docker/c35f489277eb6e13842a24ea217a64035ac33b281b127c218b04744155aede26

### To dig into a cgroup

Using namespace `4026532398` from the previous section as starting point, we could now look up the concrete resource consumption of a process in the context of a cgroup. So we take the process with PID `9422` and say we want to verify if the memory limit we set (`-m 100M`, see below) is indeed in place, that is, we're using cgroup hierarchy ID `5` (note that the following output has been edited down to the interesting bits):

    $ sudo cinf --cgroup 5:9422
    
     CONTROLFILE                         VALUE
     
     cgroup.clone_children               0
     memory.failcnt                      0
     memory.max_usage_in_bytes           1060864
     memory.move_charge_at_immigrate     0
     notify_on_release                   0
     memory.kmem.tcp.usage_in_bytes      0
     memory.usage_in_bytes               1019904
     memory.use_hierarchy                0
     cgroup.procs                        9422
     memory.kmem.limit_in_bytes          18446744073709551615
     memory.kmem.max_usage_in_bytes      245760
     memory.kmem.tcp.limit_in_bytes      18446744073709551615
     memory.kmem.tcp.max_usage_in_bytes  0
     tasks                               9422
     memory.limit_in_bytes               104857600
     memory.kmem.tcp.failcnt             0
     memory.oom_control                  oom_kill_disable 0 under_oom 0

Above we can see the maximum memory usage of this Docker container (around 1MB, from `memory.max_usage_in_bytes`) as well as that the limit we asked for is indeed in place (see `memory.limit_in_bytes`).

### To dig into a process

It is also possible to list the namespaces a specific process is in, let's take again the one with PID `9422` as an example:

    $ sudo cinf --pid 9422
    
     NAMESPACE   TYPE
     
     4026532393  mnt
     4026532394  uts
     4026532395  ipc
     4026532396  pid
     4026532398  net
     4026531837  user

Note that if you want to see detailed debug messages, you can do that via a `DEBUG` environment variable, like so: `sudo DEBUG=true cinf`.

The meaning of the columns is as follows:

- Overview (without arguments):
  - `NAMESPACE` … the namespace ID
  - `TYPE` … the type of namespace, see also [explanation of the namespaces](#overview-on-linux-namespaces-and-cgroups) below
  - `NPROCS` … number of processes in the namespace
  - `USER` … user IDs in the namespace
  - `CMD` … command line of the root process
- Detailed namespace view (`-namespace`):
  - `PID` … process ID
  - `PPID` … process ID of parent
  - `NAME` … process name
  - `CMD` … process command line
  -  `NTHREADS`… number of threads
  - `CGROUPS` … summary of the attached cgroups
  - `STATE` … process state
- Detailed cgroups view (`-cgroup`):
  - `CONTROLFILE` … the name of the control file, see also [cgroup man pages](#reading-material) below
  - `VALUE` … the content of the control file
- Detailed process view (`-pid`):
  - `NAMESPACE` … the namespace ID
  - `TYPE` … the type of namespace

### Walkthrough

For above outcomes I've used the following setup:

    $ cat /etc/*-release
    DISTRIB_ID=Ubuntu
    DISTRIB_RELEASE=14.04
    DISTRIB_CODENAME=trusty
    DISTRIB_DESCRIPTION="Ubuntu 14.04.2 LTS"
    NAME="Ubuntu"
    VERSION="14.04.2 LTS, Trusty Tahr"
    ID=ubuntu
    ID_LIKE=debian
    PRETTY_NAME="Ubuntu 14.04.2 LTS"
    VERSION_ID="14.04"
    HOME_URL="http://www.ubuntu.com/"
    SUPPORT_URL="http://help.ubuntu.com/"
    BUG_REPORT_URL="http://bugs.launchpad.net/ubuntu/"

First, I launched three Docker containers (long-running, daemonized):

    $ sudo docker run -d nginx
    $ sudo docker run -m 100M -d busybox md5sum /dev/urandom
    $ sudo docker run --user=1000 -d busybox sleep 10000

Resulting in the following Docker process listing:

    $ sudo docker ps
    CONTAINER ID        IMAGE               COMMAND                  CREATED             STATUS              PORTS               NAMES
    4121c6474b95        busybox             "sleep 1000"             32 seconds ago      Up 31 seconds                           reverent_euclid
    4c3ed0a58889        busybox             "md5sum /dev/urandom"    21 minutes ago      Up 21 minutes                           berserk_northcutt
    2f86cbc34a4d        nginx               "nginx -g 'daemon off"   21 hours ago        Up 21 hours         80/tcp, 443/tcp     amazing_kare

As well as (note: edited down to the important bits) the general Linux process listing:

    $ ps faux
    USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
    root         1  0.0  0.2  33636  2960 ?        Ss   Oct17   0:00 /sbin/init
    ...
    root     12972  0.0  3.9 757236 40704 ?        Ssl  01:55   0:18 /usr/bin/dockerd --raw-logs
    root     12981  0.0  0.9 299096  9384 ?        Ssl  01:55   0:01  \_ docker-containerd -l unix:///var/run/docker/libcontainerd/docker-containerd.sock --shim docker-containerd-shi
    root     13850  0.0  0.4 199036  4180 ?        Sl   01:58   0:00      \_ docker-containerd-shim 2f86cbc34a4d823be149935fa9a6dc176d161cebc719c60c7f95986c62ea7032 /var/run/docker/l
    root     13867  0.0  0.2  31752  2884 ?        Ss   01:58   0:00      |   \_ nginx: master process nginx -g daemon off;
    sshd     13889  0.0  0.1  32144  1664 ?        S    01:58   0:00      |       \_ nginx: worker process
    root     17642  0.0  0.4 199036  4188 ?        Sl   11:54   0:00      \_ docker-containerd-shim 4c3ed0a58889f772a5f8791528f69e78b8eeccfdcce420192b2e056f609e0b08 /var/run/docker/l
    root     17661 99.2  0.0   1172     4 ?        Rs   11:54  23:37      |   \_ md5sum /dev/urandom
    root     18340  0.0  0.4 199036  4144 ?        Sl   12:16   0:00      \_ docker-containerd-shim 4121c6474b951338b01833f90e9c3b20cc13144f132cf3534b088dc09262112b /var/run/docker/l
    vagrant  18353  0.0  0.0   1164     4 ?        Ss   12:16   0:00          \_ sleep 1000

Above you can see the three Docker container running with PIDs `13889`, `17661`, and `18353`.

## References

### Overview on Linux namespaces and cgroups

- Mount/`CLONE_NEWNS` (since Linux 2.4.19) via `mount`, `/proc/$PID/mounts`: filesystem mount points
- UTS/`CLONE_NEWUTS` (since Linux 2.6.19) via `uname -n`, `hostname -f` : nodename/hostname and (NIS) domain name
- IPC/`CLONE_NEWIPC` (since Linux 2.6.19) via `/proc/sys/fs/mqueue`, `/proc/sys/kernel`, `/proc/sysvipc`: interprocess communication resource isolation: System V IPC objects, POSIX message queues
- PID/`CLONE_NEWPID` (since Linux 2.6.24) via `/proc/$PID/status -> NSpid, NSpgid`: process ID number space isolation: PID inside/PID outside the namespace; PID namespaces can be nested
- Network/`CLONE_NEWNET` (completed in Linux 2.6.29) via `ip netns list`, `/proc/net`, `/sys/class/net`: network system resources: network devices, IP addresses, IP routing tables, port numbers, etc.
- User/`CLONE_NEWUSER` (completed in Linux 3.8) via `id`, `/proc/$PID/uid_map`, `/proc/$PID/gid_map`: user and group ID number space isolation. UID+GIDs inside/outside the namespace
- Cgroup/`CLONE_NEWCGROUP` (since Linux 4.6) via `/proc/$PID/cgroup`, `/sys/fs/cgroup/`: cgroups
- To list all namespaces of a process: `ls -l /proc/$PID/ns`

### Tooling and libs

- [lsns](http://karelzak.blogspot.ie/2015/12/lsns8-new-command-to-list-linux.html) via [karelzak/util-linux](https://github.com/karelzak/util-linux)
- [c9s/goprocinfo](https://github.com/c9s/goprocinfo)
- [shirou/gopsutil](https://github.com/shirou/gopsutil/)
- [yadutaf/ctop](https://github.com/yadutaf/ctop)

Note that the output format `cinf` uses is modelled after `lsns`, so kudos to Karel for the inspiration.

### Reading material

- [man namespaces](http://man7.org/linux/man-pages/man7/namespaces.7.html)
- [man cgroups](http://man7.org/linux/man-pages/man7/cgroups.7.html)
  - [cpuset](https://www.kernel.org/doc/Documentation/cgroup-v1/cpusets.txt)
  - [cpu](https://www.kernel.org/doc/Documentation/scheduler/sched-bwc.txt)
  - [cpuacct](https://www.kernel.org/doc/Documentation/cgroup-v1/cpuacct.txt)
  - [memory](https://www.kernel.org/doc/Documentation/cgroup-v1/memory.txt)
  - [devices](https://www.kernel.org/doc/Documentation/cgroup-v1/devices.txt)
  - [blkio](https://www.kernel.org/doc/Documentation/cgroup-v1/blkio-controller.txt)
  - perf_event
  - [net_cls](https://www.kernel.org/doc/Documentation/cgroup-v1/net_cls.txt)
- [man lsns](http://man7.org/linux/man-pages/man8/lsns.8.html)
- [Hands on Linux sandbox with namespaces and cgroups](https://blogs.rdoproject.org/7761/hands-on-linux-sandbox-with-namespaces-and-cgroups), Tristan Cacqueray (2015)
- [Namespaces in operation, part 1: namespaces overview](https://lwn.net/Articles/531114/), lwn.net (2013)
- [Resource management: Linux kernel Namespaces and cgroups](http://www.haifux.org/lectures/299/netLec7.pdf), Rami Rosen (2013)
- [THE `/proc` FILESYSTEM](https://www.mjmwired.net/kernel/Documentation/filesystems/proc.txt),  Terrehon Bowden et al (1999 - 2009)
