# ⚡ Wireshell

> **Serverless, Zero-Config, Peer-to-Peer WireGuard VPN for Windows & Linux.**  
> No central servers. No third-party control planes. No port forwarding. Pure high-speed direct P2P mesh.

---

## 🌟 Why Wireshell?

Most traditional VPNs (and even modern mesh tools like Tailscale or ZeroTier) rely on central coordination servers, proprietary control planes, or paid subscriptions to negotiate connections. 

**Wireshell is built different:**
- **Zero Central Servers:** You don't have to host, configure, or trust any coordination server.
- **Direct P2P Speed:** Encrypted packets travel directly between your devices over UDP. There are no intermediate relay servers slowing down your connection or snooping on traffic.
- **WireGuard Protocol:** Industry-standard cryptography using **Curve25519**, **ChaCha20-Poly1305**, and **BLAKE2s**. Maximum line-rate throughput and lowest ping times.
- **Automated NAT Traversal & Hole Punching:** Automatically discovers public WAN endpoints via public STUN servers and punches bidirectional holes through home firewalls and CGNAT.
- **Embedded Wintun Driver:** On Windows, the high-performance `wintun.dll` driver is embedded directly inside the binary. No manual driver installations or installers needed.
- **Self-Healing Connections:** If your laptop changes Wi-Fi networks or drops connection, Wireshell automatically detects the change, re-discovers endpoints, and re-establishes the tunnel.

---

## 🏗️ Architecture

```
  [Device A (Windows / Linux)]                    [Device B (Windows / Linux)]
               │                                               │
               ├────────► 1. STUN Query (Public STUN) ◄────────┤  (Discover WAN IP & NAT Port)
               │                                               │
               │   2. Out-of-band Ticket Exchange (1-step)     │
               │◄─────────────────────────────────────────────►│  (Exchange public keys & endpoints)
               │                                               │
               │═══════════════════════════════════════════════│
               │         3. Bidirectional UDP Hole Punch       │  (Pierce NAT firewalls)
               │                                               │
               │███████████████████████████████████████████████│
               │   4. Direct WireGuard Encrypted Data Plane    │  (ChaCha20-Poly1305 line-rate)
               │███████████████████████████████████████████████│
```

---

## 🚀 Quick Start Guide

### 1. Build the Binary

#### On Windows (PowerShell):
```powershell
.\build.ps1
# or
go build -o wireshell.exe ./cmd/wireshell
```

#### On Linux:
```bash
make build-linux
# or
go build -o wireshell ./cmd/wireshell
```

---

### 2. Connect Two Devices in 4 Steps

#### **Step 1: Initialize Identity (Both Devices)**
Run this on Device A and Device B:
```bash
wireshell init
```
This generates a private/public Curve25519 keypair and assigns your virtual mesh IP (e.g. `10.42.0.1/24`).

#### **Step 2: Generate Ticket on Device A**
On Device A, run:
```bash
wireshell ticket
```
Wireshell probes local network interfaces and queries public STUN servers to detect your WAN endpoint, generating a ticket:
```text
wireshell://ticket/eyJ2IjoxLCJwdWIiOiJZbEVXV2JXekpDVm84bFNPV01VN2RkUUFtRXRH...
```

#### **Step 3: Join on Device B**
On Device B, run:
```bash
wireshell join wireshell://ticket/eyJ2IjoxLCJwdWIiOiJZbEVXV2JXekpDVm84bFNPV01VN2RkUUFtRXRH...
```
*(Device B automatically configures Device A as a peer with assigned mesh routing).*

#### **Step 4: Start the VPN Tunnel**
Because Wireshell creates a virtual network interface adapter (Wintun on Windows / TUN on Linux), run it with Administrator / Root privileges:

**On Windows (Run PowerShell as Administrator):**
```powershell
.\wireshell.exe up
```

**On Linux:**
```bash
sudo ./wireshell up
```

Your high-speed, direct P2P WireGuard tunnel is live! You can now ping, SSH, share files, or route traffic directly across your virtual mesh IPs (e.g. `10.42.0.1` <-> `10.42.0.2`).

---

## 📊 Checking Status

To inspect active peers, transferred bytes, and connection health:
```bash
wireshell status
```
Output:
```text
==================================================
  📡 Wireshell Mesh Status
==================================================
Public Key: R8OXssYkc/eW8IuoNA0lWOBTlXpwtR6a9+Qiaof8/lo=
Mesh IP   : 10.42.0.2/24
Peers     : 1 configured
--------------------------------------------------
[1] Peer: YlEWWbWzJCVo...
    Mesh IP     : 10.42.0.1/24
    AllowedIPs  : [10.42.0.1/32]
    Endpoint    : 49.207.219.88:12157
    Keepalive   : 25s
```

---

## 🐳 Docker Support (Linux Server / Headless)

For headless Linux boxes or home NAS servers:

```bash
docker compose up -d
```

Ensure the container has `NET_ADMIN` capability and access to `/dev/net/tun` (already pre-configured in `docker-compose.yml`).

---

## 🔒 Security & Privacy Model

- **End-to-End Cryptography:** Handled entirely by WireGuard’s Noise Protocol framework.
- **Zero Logging & Zero Telemetry:** Wireshell has no telemetry, no tracking, and no external callbacks.
- **Ephemeral Rendezvous:** STUN queries only use standard public STUN binding requests to learn external NAT port mappings; no identity or private data is ever shared with STUN servers.
- **No Third-Party Relays:** All VPN data packets travel strictly between Peer A and Peer B.

---

## 🛠️ CLI Reference

| Command | Description |
| :--- | :--- |
| `wireshell init` | Initializes identity, generates cryptographic keypair and assigns mesh IP. |
| `wireshell ticket` | Probes LAN/WAN STUN endpoints and generates a shareable invite ticket. |
| `wireshell join <ticket>` | Parses peer ticket, configures routing, and prepares P2P tunnel. |
| `wireshell up` | Launches the Wireshell WireGuard daemon and maintains hole-punched tunnels. |
| `wireshell status` | Displays active peers, endpoints, transfer statistics, and handshake state. |

---

## 📄 License
MIT License. High performance, zero servers, 100% private.
