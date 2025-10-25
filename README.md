# isomer

Just like chemical [isomers](https://en.wikipedia.org/wiki/Isomer) share the same molecular formula but differ in structure, [Isomer](https://github.com/PapyrusVIP/isomer) uses the same underlying traffic attributes (like port or IP) to dispatch traffic in different ways.

Isomer is a flexible eBPF socket dispatcher.
You can define how traffic is dispatched to processes running on the same machine.
Unlike the fixed model of BSD `bind`, Isomer gives you a programmable control plane for traffic dispatching.

* You can bind to multiple or all ports on an IP
* You can bind to a subnet instead of an IP
* You can bind to multiple or all ports on a subnet

**Note:** Requires at least Linux v5.10 which is when the [`sk_lookup`](https://www.kernel.org/doc/html/v5.10/bpf/prog_sk_lookup.html) program type was released.

> We want to hear your feedback, please open issues if you have any feature requests or bug reports! Feel free to star the project if you like it!

## Quickstart

```sh
# Install and load isomer
$ go install github.com/PapyrusVIP/isomer/cmd/isomctl@latest
$ sudo isomctl load

# Send port 4321 traffic on all loopback IPs to the foo label.
$ sudo isomctl bind foo tcp 127.0.0.0/8 4321

# Set up a server and register the listening socket under the foo label
$ nc -k -l 127.0.0.1 9999 &
$ sudo isomctl register-pid $! foo tcp 127.0.0.1 9999

# Send a message!
$ echo $USER | nc -q 1 127.0.0.23 4321
```

The real power is in the `bind` command.

```sh
# Send HTTP traffic on a /24 to the foo label.
$ sudo isomctl bind foo tcp 127.0.0.0/24 80
$ echo $USER | nc -q 1 127.0.0.123 80

# Send TCP traffic on all ports of a specific IP to the foo label.
$ sudo isomctl bind foo tcp 127.0.0.22 0
$ echo $USER | nc -q 1 127.0.0.22 $((1 + $RANDOM))
```

## Integrating with isomer

TCP servers are compatible with isomer out of the box. For UDP you need to
set some additional socket options and change the way you send replies.

In general, you will have to **register your sockets with isomer**. The easiest
way is to use `tubectl register-pid` combined with a systemd service of
[Type=notify](https://www.freedesktop.org/software/systemd/man/systemd.service.html#Type=). 
It's also possible to use systemd socket activation combined with `tubectl register`, but this setup is more complicated than `register-pid`.

**[The example](example/README.md) shows how to use `register-pid` with a TCP
and UDP echo server.**

Testing
---

`isomer` requires at least Linux v5.10 with unprivileged bpf enabled.

```sh
$ sysctl kernel.unprivileged_bpf_disabled
kernel.unprivileged_bpf_disabled = 0 # must be zero
$ make test
```
