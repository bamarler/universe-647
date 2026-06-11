# Remote-Access Resilience Runbook

Why this exists: on 2026-06-11 a miscapitalized `make down` removed every
container — including Tailscale, which ran *inside* the Docker stack it was
protecting. Total remote lockout; recovery required the physical console.
This runbook removes that failure class with three independent layers:

| Layer | Survives | Cost |
|---|---|---|
| 1. Host-level tailscaled | anything Docker/compose does | $0 |
| 2. Intel AMT (vPro) | a dead OS — console + power below Linux | $0 (+$7 dummy plug, optional) |
| 3. Deadman watchdog | transient daemon/network wedges, hands-free | $0 |

Do layer 1 **while physically at home** — the cutover itself is the one moment
where a mistake could cost access.

---

## Layer 1 — Move Tailscale from container to host systemd

The container (`stacks/core/compose.yaml`, service `tailscale`) is the only way
in; Tailscale's own docs frame the Docker image as a per-service sidecar, not a
host lifeline. The host install runs as a systemd unit with `Restart=on-failure`
and zero coupling to docker.service.

Current container config to replicate: `network_mode: host`, hostname from
`TS_HOSTNAME` (= `universe-647`), and — **important** — the node is tagged
`tag:server` via `TS_EXTRA_ARGS: --advertise-tags=tag:server`. ACLs key off
that tag, and tagged nodes have key expiry disabled. The host node must carry
the same tag.

```bash
# 1. Generate an auth key: admin console → Settings → Keys → new auth key,
#    tags: tag:server, reusable: no, expiry: short. (A tagged auth key is the
#    clean way to bring up a tagged node.)

# 2. Install tailscale on the host (Ubuntu 24.04):
curl -fsSL https://tailscale.com/install.sh | sh

# 3. Bring the host node up under a TEMPORARY name, alongside the container:
sudo tailscale up --authkey tskey-auth-XXXXX \
  --hostname universe-647-host --advertise-tags=tag:server

# 4. VERIFY before touching the container — from BOTH laptop and phone:
#    ssh <user>@universe-647-host   (or the new node's 100.x address)
#    Do not proceed until both work.

# 5. Retire the container node:
docker stop tailscale && docker rm tailscale
#    Admin console → Machines → delete the OLD universe-647 (container) node.

# 6. Reclaim the name (frees the MagicDNS name for the host node):
sudo tailscale up --hostname universe-647 --advertise-tags=tag:server
#    Re-verify ssh universe-647 from laptop + phone.

# 7. Confirm the unit is enabled and self-restarting:
systemctl is-enabled tailscaled   # → enabled
systemctl cat tailscaled | grep Restart
```

Afterwards (separate session): remove the `tailscale` service block from
`stacks/core/compose.yaml`, drop `TS_AUTHKEY`/`TS_HOSTNAME` from core's .env
(+ re-encrypt), update CLAUDE.md, and retire the "nohup for core stack" rule —
it existed only because compose operations could kill the access path.

## Layer 2 — Intel AMT (vPro): the free out-of-band console

AMT (Active Management Technology) is an independent management computer inside
the chipset with its own network stack on the wired NIC, alive whenever AC is
connected. It provides remote power control, BIOS access, and hardware KVM even
when Linux is dead. The i7-10700T + EliteDesk 800 G6 Mini is a vPro platform.

Provisioning (at the machine):
1. Reboot, press **Ctrl+P** at POST → MEBx menu. Default password `admin`;
   it forces a strong replacement on first login. **Save it in 1Password.**
2. Sanity check: the menu must say **"Intel AMT Configuration"**. If it only
   shows "Intel Standard Manageability", this unit lacks KVM (some HP SKUs
   ship the lesser ME) — layer 2 then isn't available; stop here.
3. Enable: Intel AMT Configuration → Activate Network Access; under
   User Consent set KVM consent as you prefer (no-consent = usable headless);
   ensure KVM Feature Selection is enabled.
4. Headless note: hardware KVM renders the GPU output — with no monitor there
   is nothing to render. A ~$7 HDMI dummy plug fixes it. Without one, power
   control + serial-over-LAN still work.
5. From the laptop (on the LAN or over the tailnet once layer 1 is done):
   web UI at `http://<server-LAN-IP>:16992`, or MeshCommander
   (https://meshcommander.com) for full KVM.

Security rules (non-negotiable):
- **LAN-only, forever.** Never port-forward 16992–16995. AMT is pre-OS and
  cannot join the tailnet itself — reach it by going through the tailnet to a
  device on the LAN (the server itself, post-layer-1, routes you home; or the
  laptop when physically home).
- AMT has a real CVE history (it sits below everything). Mitigation = LAN-only
  + strong MEBx password + keep HP BIOS/ME firmware updated (fwupd / HP SoftPaqs).
- Set BIOS **Power → After Power Loss → Power On** while you're in there: after
  any outage the box boots unattended (containers are `restart: unless-stopped`,
  tailscaled is systemd-enabled — the whole stack self-restores).

## Layer 3 — Deadman watchdog

Files live in this repo: `scripts/tailscale-watchdog.sh`,
`scripts/systemd/tailscale-watchdog.{service,timer}`.

Behavior: every 2 minutes check tailscale health; after 2 consecutive failures
restart tailscaled; after 5, reboot the host (never within 10 minutes of boot,
so it cannot boot-loop). All actions logged to the journal.

```bash
sudo cp scripts/tailscale-watchdog.sh /usr/local/bin/
sudo chmod +x /usr/local/bin/tailscale-watchdog.sh
sudo cp scripts/systemd/tailscale-watchdog.{service,timer} /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now tailscale-watchdog.timer

# Verify + fire drill:
systemctl list-timers tailscale-watchdog.timer
sudo systemctl stop tailscaled        # simulate a wedge
journalctl -t tailscale-watchdog -f   # within ~4 min: failure logs + restart
```

## Hygiene (from any device, anytime)

**ACL lockout insurance.** Admin console → Access Controls: add a `tests`
section — the console refuses to save any policy that fails its tests, so a
future ACL edit can't lock your devices out of SSH:

```jsonc
"tests": [
  {
    "src": "b.marler@northeastern.edu",
    "accept": ["tag:server:22"]
  }
]
```

**sshd: verify, don't change.** Keep plain OpenSSH (independent failure domain
from tailscaled — do NOT switch to Tailscale SSH, which couples shell and
transport). Confirm key-only:

```bash
sudo sshd -T | grep -E 'passwordauthentication|kbdinteractiveauthentication|permitrootlogin'
# want: passwordauthentication no, kbdinteractiveauthentication no, permitrootlogin no (or prohibit-password)
```

**Tailnet Lock: currently OFF — leave it off for now.** It protects against a
compromised Tailscale control plane adding rogue nodes; at a 3-device personal
tailnet the realistic availability cost (new nodes locked out until signed by
another signing node; unrecoverable tailnet if all signing nodes + disablement
secrets are lost) outweighs that threat — this week demonstrated which risk is
live. If enabling later: run `tailscale lock init` from the laptop, keep ≥2
signing nodes, and **immediately save all printed disablement secrets to
1Password** — they are shown exactly once.

## Post-cutover cleanup checklist

- [ ] Remove `tailscale` service from `stacks/core/compose.yaml`
- [ ] Remove `TS_AUTHKEY`/`TS_HOSTNAME` from `stacks/core/.env` + `make encrypt`
- [ ] Update CLAUDE.md (traffic flow, port-binding rule, container counts)
- [ ] Retire the "nohup for core stack compose ops" rule
- [ ] `rm -r /srv/u647/tailscale` (old container state) once stable for a week
