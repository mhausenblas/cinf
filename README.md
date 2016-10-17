# cinf—container info

This is `cinf` (short for container info), a command line tool to view namespaces and cgroups, the stuff that makes up Linux containers such as Docker, rkt/appc, or OCI/runc. It might be useful for low-level container prodding, when you need to understand what's going on under the hood.


## Install

TBD.

## Use

TBD.

## References

Overview:

- Mount/`CLONE_NEWNS` (since Linux 2.4.19) via `mount`, `/proc/$PID/mounts`: filesystem mount points
- UTS/`CLONE_NEWUTS` (since Linux 2.6.19) via `uname -n`, `hostname -f` : nodename/hostname and (NIS) domain name
- IPC/`CLONE_NEWIPC` (since Linux 2.6.19) via `/proc/sys/fs/mqueue`, `/proc/sys/kernel`, `/proc/sysvipc`: interprocess communication resource isolation: System V IPC objects, POSIX message queues
- PID/`CLONE_NEWPID` (since Linux 2.6.24) via `/proc/$PID/ns`, `/proc/$PID/status -> NSpid, NSpgid`: process ID number space isolation: PID inside/PID outside the namespace; PID namespaces can be nested
- Network/`CLONE_NEWNET` (completed in Linux 2.6.29) via `ip netns list`, `/proc/net`, `/sys/class/net`: network system resources: network devices, IP addresses, IP routing tables, port numbers, etc.
- User/`CLONE_NEWUSER` (completed in Linux 3.8) via `id`, `/proc/$PID/uid_map`, `/proc/$PID/gid_map`: user and group ID number space isolation. UID+GIDs inside/outside the namespace
- Cgroup/`CLONE_NEWCGROUP` (since Linux 4.6) via `/proc/$PID/ns/cgroup`: cgroup root directory
- To list all namespaces of a process: `ls -l /proc/$PID/ns`

Tooling and libs:

- [lsns](http://karelzak.blogspot.ie/2015/12/lsns8-new-command-to-list-linux.html) via [karelzak/util-linux](https://github.com/karelzak/util-linux)
- [c9s/goprocinfo](https://github.com/c9s/goprocinfo)
- [shirou/gopsutil](https://github.com/shirou/gopsutil/)
- [yadutaf/ctop](https://github.com/yadutaf/ctop)


### Material

- [man lsns](http://man7.org/linux/man-pages/man8/lsns.8.html)
- [man namespaces](http://man7.org/linux/man-pages/man7/namespaces.7.html)
- [Hands on Linux sandbox with namespaces and cgroups](https://blogs.rdoproject.org/7761/hands-on-linux-sandbox-with-namespaces-and-cgroups), Tristan Cacqueray (2015)
- [Namespaces in operation, part 1: namespaces overview](https://lwn.net/Articles/531114/), lwn.net (2013)
- [Resource management: Linux kernel Namespaces and cgroups](http://www.haifux.org/lectures/299/netLec7.pdf), Rami Rosen (2013)
- [THE `/proc` FILESYSTEM](https://www.mjmwired.net/kernel/Documentation/filesystems/proc.txt),  Terrehon Bowden et al (1999 - 2009)


### Development

Build local:

    $ GOOS=linux go build

Build via CI/CD: 

    TBD