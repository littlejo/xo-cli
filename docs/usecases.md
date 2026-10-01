# Use cases

Task-oriented recipes that put several `xo` commands together to achieve
something real. Each use case shows the commands in order, the expected output,
and how to script it. For the full command reference see
[usage.md](usage.md); for the architecture see [development.md](development.md).

A use case is *complete* when every step is a typed `xo` command (no `xo rest`
needed) and the final state is verified.

## Table of contents

- [Add a disk to a VM](#add-a-disk-to-a-vm)

---

## Add a disk to a VM

Attach a new virtual disk (VDI) to a running or halted VM — the equivalent of
adding a second hard drive. This is the most common storage task and touches
two resources: the **VDI** (the disk, which lives on a storage repository) and
the **VBD** (the attachment that plugs the disk into the VM).

```text
   1. create the VDI  on an SR        xo vdi create
   2. attach it to the VM  (VBD)      xo vbd create
   3. hot-plug it (VM is running)     xo vbd connect
   4. verify                        xo vm vdis / xo vbd list
```

### Prerequisites

- A configured profile (`xo configure`) that can reach the pool.
- A **storage repository (SR)** the VM can see — usually the pool's shared SR.
  List them with `xo sr list`. A VDI can only live on an SR that is available
  to the hosts of the VM's pool (a local SR works only if the VM runs on that
  host).
- The VM's UUID (`xo vm list`) and, if the disk is bootable or read-only, a
  decision about `--bootable` / `--mode`.

### Step 1 — find a target SR and size

```sh
xo sr list
```

```
ID                                    NAME            TYPE   SIZE      USAGE     CONTAINER
------------------------------------  --------------  -----  --------  --------  ------------------------------------
aaaaaaaa-bbbb-cccc-dddd-000000000001  Local storage   lvm    200GB     95GB      aaaaaaaa-bbbb-cccc-dddd-000000000009
bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb  NFS data        nfs    2TB       1.2TB     aaaaaaaa-bbbb-cccc-dddd-000000000009
```

Pick the shared SR (e.g. `bbbbbbbb-…`) and a virtual size (`10G`, `50G`, or a
raw byte count).

### Step 2 — create the VDI

```sh
xo vdi create data-01 --sr bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb --size 10G
```

```
VDI created (id 44444444-4444-4444-8444-444444444444)

Attach it to a VM with: xo vbd create --vm <vm-id> --vdi 44444444-4444-4444-8444-444444444444
```

The created VDI is **not** attached to anything yet. Useful flags:
`--description`, `--tags a,b`, and `--shared` (let several VMs attach the same
VDI at once, read/write — only sensible on a shared SR and a VM that supports
it).

> The VDI id is what you need for the next step. Capture it in a script
> (`--output json` makes this robust, see below).

### Step 3 — attach it to the VM (create the VBD)

```sh
xo vbd create --vm 550e8400-e29b-41d4-a716-446655440001 --vdi 44444444-4444-4444-8444-444444444444
```

```
VBD created (id 55555555-5555-4555-8555-555555555555): VDI 44444444-… attached to VM 550e8400-…

If the VM is running, hot-plug the disk with: xo vbd connect 55555555-5555-4555-8555-555555555555
```

Creating the VBD does **not** make the disk appear inside a running guest —
that is the hot-plug in the next step. Optional flags: `--mode RO` (read-only
attachment) and `--bootable` (make this a boot device).

### Step 4 — hot-plug (only if the VM is running)

```sh
xo vbd connect 55555555-5555-4555-8555-555555555555
```

```
Requested connect of VBD 55555555-… (VM 550e8400-…, VDI 44444444-…) (task-123)
```

- **VM running** → run `xo vbd connect` to attach the disk without a reboot.
  The guest OS then sees a new block device (e.g. `/dev/xvdb` on a typical
  Linux guest).
- **VM halted** → skip this step; the disk is attached automatically at boot.

To remove the disk later without deleting it, hot-unplug it first with
`xo vbd disconnect <vbd-id>` (running VM), then detach with
`xo vbd delete <vbd-id> --yes`, and finally remove the disk with
`xo vdi delete <vdi-id> --yes`.

### Step 5 — verify

```sh
xo vm vdis 550e8400-e29b-41d4-a716-446655440001
```

```
ID                                    NAME         TYPE    SIZE    USAGE   SR
------------------------------------  -----------  ------  -------  ------  ------------------------------------
11111111-1111-4111-8111-111111111111  system disk  system  10.74GB  5.37GB  aaaaaaaa-bbbb-cccc-dddd-000000000001
44444444-4444-4444-8444-444444444444  data-01      user    10.74GB  0B      bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb
```

Or inspect the attachment directly:

```sh
xo vbd list --vm 550e8400-e29b-41d4-a716-446655440001
```

```
ID                                    VM                                VDI                                DEVICE  MODE  ATTACHED
------------------------------------  --------------------------------  --------------------------------  ------  ----  --------
33333333-3333-4333-8333-333333333333  550e8400-e29b-41d4-a716-446655440001  11111111-1111-4111-8111-111111111111  xvda    RW    yes
55555555-5555-4555-8555-555555555555  550e8400-e29b-41d4-a716-446655440001  44444444-4444-4444-8444-444444444444  -       RW    no
```

The new VBD shows the assigned `DEVICE` (e.g. `xvdb`) once attached/hot-plugged.

### Scripting it end to end

Machine-readable output keeps only the requested data on stdout, so the steps
chain cleanly. The VDI id is the only value passed between steps:

```sh
set -euo pipefail
POOL_SR="bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
VM="550e8400-e29b-41d4-a716-446655440001"

# 1. create the disk, keep only its id
VDI_ID=$(xo vdi create data-01 --sr "$POOL_SR" --size 10G --output json | jq -r '.vdi')

# 2. attach it, keep the VBD id
VBD_ID=$(xo vbd create --vm "$VM" --vdi "$VDI_ID" --output json | jq -r '.vbd')

# 3. hot-plug if the VM is running (no-op / refused if halted — check state first)
if [ "$(xo vm get "$VM" --query 'power_state' --output text)" = "Running" ]; then
  xo vbd connect "$VBD_ID"
fi

# 4. verify
xo vm vdis "$VM" --query "[?name_label=='data-01'].id"
```

`--output json` for `vdi create` prints `{"action":"create","vdi":"<id>"}` and
for `vbd create` prints `{"action":"create","vm":"…","vdi":"…","vbd":"<id>"}` —
exactly the fields needed, nothing else.

### Removing the disk (reverse order)

```sh
xo vbd disconnect "$VBD_ID"     # hot-unplug (running VM)
xo vbd delete "$VBD_ID" --yes   # detach (the VDI is kept)
xo vdi delete "$VDI_ID" --yes   # delete the disk (irreversible — needs --yes)
```

The order matters: detach the VBD before deleting the VDI, and a VDI that is
still attached to any VM cannot be deleted.

### Troubleshooting

| Symptom | Likely cause / fix |
| ------- | ------------------ |
| `cannot create VDI: … SR not found` or `invalid --sr id` | Wrong SR UUID. Use `xo sr list`; the SR must be visible to the VM's pool. |
| `cannot attach VDI … to VM …` | The VDI's SR is not available to the VM's pool (local SR on another host, or the VM is on a different pool). Pick a shared SR. |
| Disk created but the guest doesn't see it | The VM is running and you skipped the hot-plug. Run `xo vbd connect <vbd-id>`. |
| `cannot delete VDI` | The VDI is still attached. Run `xo vbd list --vm <id>` and `xo vbd delete` each VBD first. |
| Hot-plug refused | The guest OS or the disk type doesn't support hot-plug; reboot the VM instead. |

Every failure is reported as `Error: …` on **stderr**; machine-readable output
on stdout stays clean, so scripts can branch on the exit code and the stderr
message.
