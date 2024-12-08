# labctl

labctl controls roman’s homelab

---

## Status

This project is in development phase. If you think it could be useful to you as well, you can download and install it using go install:

```sh
go install github.com/romantomjak/labctl@latest
```

and then copy the [example config](/labctl.hcl.example) to `~/.labctl.hcl`.

## Usage

To interact with proxmox clusters:

```sh
$ labctl pve ps
ID   NAME           NODE   STATUS   UPTIME     MEM    CPU       
100  ceph-1         pve01  running  19h56m46s  15 GB  0.27871  
101  k8s-control-1  pve01  running  8h2m32s    2.7 GB 0.30397  
102  vault          pve01  stopped  0s         0 B    0  
104  k8s-worker-1   pve01  running  8h2m45s    4.5 GB 0.19541  
$ labctl pve start vault
🚦 Will start the VMs in the following order:
  - vault
❓ Do you want to continue? [y/n] y
🚀 Starting the VMs
  - vault... OK ✅
```

or to copy an access token into clipboard and open kubernetes dashboard in browser:

```sh
$ labctl k8s dashboard
```

:sparkles:

## Contributing

You can contribute in many ways and not just by changing the code! If you have any ideas, just open an issue and tell me what you think.

Contributing code-wise - please fork the repository and submit a pull request.

# License

MIT
