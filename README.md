# Wireshell

Decentralized Peer-to-Peer WireGuard Tunneling Engine for Windows and Linux.

---

## 1. Overview

Wireshell is an implementation of a decentralized, serverless Virtual Private Network (VPN) engine built on top of the WireGuard protocol. It provides peer-to-peer (P2P) layer-3 IP connectivity without relying on centralized coordination servers, control planes, or persistent third-party infrastructure.

NAT traversal is accomplished through standard STUN (RFC 5389 / RFC 8489) binding requests and simultaneous UDP hole punching, followed by direct, encrypted datagram transmission using the WireGuard Noise protocol framework.

---

## 2. Design Principles and Architecture

### 2.1 Separation of Concerns
The system separates the **Signaling/Discovery Plane** from the **Data Plane**:

- **Signaling and Discovery Plane:** Decentralized out-of-band ticket exchange or ephemeral rendezvous. Handles public endpoint discovery, candidate negotiation, and UDP hole punching. Once connectivity is established, this plane transitions to an idle state.
- **Data Plane:** Handled entirely by the userspace WireGuard engine (`golang.zx2c4.com/wireguard`), running over an OS-level TUN device (`Wintun` on Windows, `/dev/net/tun` on Linux). All payload traffic is encrypted and authenticated end-to-end between peer sockets.

```
+-------------------------------------------------------------------------------+
|                               WIRESHELL ENGINE                                |
+-------------------------------------------------------------------------------+
|                                                                               |
|  [Discovery & STUN]          [Hole Puncher]               [Tunnel Manager]   |
|  Query RFC 5389 endpoints    Simultaneous UDP probes      Manage UAPI state   |
|  Generate endpoint vectors   Cookie verification          Configure routing   |
|         |                           |                            |            |
+---------|---------------------------|----------------------------|------------+
          |                           |                            |
          v                           v                            v
  [Public Internet]          [Bidirectional NAT Hole]       [OS Network Stack]
  STUN resolution            Direct peer socket mapping     Wintun / /dev/net/tun
```

### 2.2 Connection Lifecycle

```
[Peer A]                                                            [Peer B]
   |                                                                   |
   |--- 1. Query STUN (RFC 5389) ------------------------------------->|
   |    Resolve: {WAN IP, Mapped Port, Local Interface IPs}            |
   |                                                                   |
   |<-- 2. Out-of-Band Ticket Exchange (Base64 URL-Safe Vector) ------>|
   |    Exchange: Public Key, Virtual Mesh IP, Candidate Endpoints     |
   |                                                                   |
   |--- 3. Concurrent UDP Probe Burst (Hole Punching) ---------------->|
   |<-- 3. Concurrent UDP Probe Burst (Hole Punching) -----------------|
   |    Stateful firewall tables updated on intermediate NATs          |
   |                                                                   |
   |=== 4. WireGuard Handshake (Noise_IKpsk2) =========================|
   |<== 4. WireGuard Handshake (Noise_IKpsk2) =========================|
   |                                                                   |
   |### 5. Encrypted Data Plane (ChaCha20-Poly1305) ###################|
   |<## 5. Encrypted Data Plane (ChaCha20-Poly1305) ###################|
```

---

## 3. Cryptographic Specification

Wireshell strictly implements the standard WireGuard cryptographic suite:

| Component | Primitive | Specification |
| :--- | :--- | :--- |
| Key Exchange | Curve25519 (ECDH) | RFC 7748 |
| Authenticated Encryption | ChaCha20-Poly1305 AEAD | RFC 8439 |
| Cryptographic Hashing | BLAKE2s | RFC 7693 |
| Key Derivation | HKDF | RFC 5869 |
| Table Lookup & Hash Map Keying | SipHash-2-4 | Native WireGuard implementation |
| Handshake Framework | Noise_IKpsk2 | Noise Protocol Framework |

### Key Properties:
- **Perfect Forward Secrecy:** Ephemeral session keys are renegotiated every 120 seconds or after $2^{64} - 2^{13} - 1$ encrypted packets.
- **Identity Hiding:** Initiator and responder public keys are protected against active on-path attackers.
- **Replay Protection:** 64-bit sliding window anti-replay counter.

---

## 4. NAT Traversal and Hole Punching Mechanics

Residential and cellular networks enforce varying degrees of Network Address Translation (Full-Cone, Restricted-Cone, Port-Restricted Cone, and Symmetric NAT). Wireshell implements a deterministic traversal algorithm:

1. **Candidate Gathering:** Local addresses are enumerated from network interface tables. The external reflexive transport address is discovered by querying public STUN servers over UDP (`stun.l.google.com:19302`, `stun.cloudflare.com:3478`).
2. **Prioritization:** Direct LAN IP paths are prioritized over WAN paths to maintain local wire speeds when peers reside within the same broadcast domain.
3. **Simultaneous UDP Hole Punching:** Both peers transmit high-frequency synchronization datagrams prefixed with a verification cookie (`WS_HOLE_PUNCH_V1:<nonce>`). This populates the outbound state table of symmetric/cone NATs.
4. **Endpoint Promotion:** Upon receipt of a valid acknowledgment (`WS_HOLE_PUNCH_ACK:<nonce>`), the verified socket address is committed to the WireGuard device via the UAPI (`update_only=true\nendpoint=<addr>\n`).
5. **Keepalive Cycling:** Persistent keepalives (`persistent-keepalive = 25s`) prevent intermediate NAT state timeouts from dropping the mapping.

---

## 5. Operating System Integration

### 5.1 Windows
- **Adapter Driver:** Embedded `wintun.dll` driver binaries (x86_64 and aarch64) embedded directly in the executable via Go compiler directives (`//go:embed`). The runtime automatically stages the driver if not already present.
- **Interface Configuration:** Configured programmatically via the Windows IP Helper API (`winipcfg`) by resolving the adapter LUID/GUID, assigning IPv4 unicast addresses, and binding routing table entries without spawning external netsh subshells.

### 5.2 Linux
- **Adapter Driver:** Linux kernel Universal TUN/TAP driver (`/dev/net/tun`).
- **Network Stack:** Netlink socket interface commands (`ip addr`, `ip link`, `ip route`) and kernel sysctl parameter controls (`net.ipv4.ip_forward=1`).
- **Container Deployments:** Requires `CAP_NET_ADMIN` capability and pass-through access to `/dev/net/tun`.

---

## 6. Operation Modes

### 6.1 Point-to-Point Mesh Mode (Default)
Peers form an isolated virtual overlay network (default subnet: `10.42.0.0/24`). Intra-mesh packets are routed directly point-to-point without transit overhead. External internet traffic routes through the local host's default gateway.

### 6.2 Exit Node Mode (Default Gateway Routing)
A peer can be instructed to route default outbound traffic (`0.0.0.0/0`, `::/0`) through a designated remote node:
- **Client Configuration:** Sets `AllowedIPs = 0.0.0.0/0`, routing system-wide traffic across the virtual adapter interface.
- **Exit Node Routing:** The designated exit node enables IP forwarding and Network Address Translation (NAT masquerade) on its egress network interface.
- **Edge Routing (WARP Mode):** Wireshell includes automated provisioning for Cloudflare's global WireGuard edge network (`engage.cloudflareclient.com`), allowing immediate public IP masking without provisioning dedicated exit infrastructure.

---

## 7. Compilation and Build Matrix

### Prerequisites
- Go 1.22 or higher
- Windows 10/11 / Windows Server 2019+ (amd64, arm64)
- Linux Kernel 4.19+ (amd64, arm64)

### Native Builds

#### Windows (PowerShell):
```powershell
go build -ldflags="-s -w" -o wireshell.exe ./cmd/wireshell
```

#### Linux:
```bash
CGO_ENABLED=0 go build -ldflags="-s -w" -o wireshell ./cmd/wireshell
```

#### Cross-Compilation Target:
```bash
make build-windows   # Produces wireshell.exe (Windows amd64)
make build-linux     # Produces wireshell-linux-amd64 (Linux amd64)
```

---

## 8. CLI Reference and Workflow

### 8.1 Topology Initialization

#### Step 1: Initialize Local Node Identity
Generates a Curve25519 keypair and assigns a virtual mesh IP:
```bash
wireshell init --ip 10.42.0.1/24 --port 51820
```

#### Step 2: Generate Peer Ticket (Node A)
Queries STUN and gathers candidate addresses:
```bash
wireshell ticket
```
Returns a URL-safe Base64 serialized ticket:
```text
wireshell://ticket/eyJ2IjoxLCJwdWIiOiJZbEVXV2JXekp...
```

#### Step 3: Register Peer (Node B)
Parses Node A's candidate vector and provisions local routing:
```bash
# Standard P2P mesh connection:
wireshell join wireshell://ticket/<base64-payload>

# Full tunnel / Internet exit node mode (routes all traffic via peer):
wireshell join wireshell://ticket/<base64-payload> --exit-node
```

#### Step 4: Start Tunnel Daemon
Brings up the virtual interface and initiates hole punching:
```bash
# Windows (requires elevated Administrator shell):
.\wireshell.exe up

# Linux:
sudo ./wireshell up
```

### 8.2 Edge Connection & Verification Commands

#### Query Active Public IP and Geolocation:
```bash
wireshell ip
```

#### Connect to Encrypted WireGuard Edge Network:
```bash
wireshell connect
```

#### Execute Automated P2P Mesh Simulation:
```bash
wireshell test-mesh
```

#### Display Peer Connection Metrics:
```bash
wireshell status
```

---

## 9. Configuration Schema

Configuration files persist to platform-standard directories:
- **Windows:** `%APPDATA%\Wireshell\config.json`
- **Linux:** `~/.wireshell/config.json` or `/etc/wireshell/config.json`

```json
{
  "private_key": "<base64-encoded-curve25519-private-key>",
  "public_key": "<base64-encoded-curve25519-public-key>",
  "mesh_ip": "10.42.0.1/24",
  "listen_port": 51820,
  "mtu": 1420,
  "peers": [
    {
      "public_key": "<base64-encoded-peer-public-key>",
      "mesh_ip": "10.42.0.2/24",
      "allowed_ips": [
        "10.42.0.2/32"
      ],
      "endpoint": "198.51.100.25:51820",
      "persistent_keepalive": 25
    }
  ]
}
```

---

## 10. Docker Deployment

A minimal multi-stage `Dockerfile` is provided for containerized and headless environments.

```bash
docker compose up -d
```

### Required Linux Container Capabilities:
```yaml
cap_add:
  - NET_ADMIN
  - SYS_MODULE
devices:
  - /dev/net/tun:/dev/net/tun
network_mode: host
```

---

## 11. Security Model and Audit Considerations

- **Attack Surface Minimization:** Wireshell executes with zero network listener exposure outside of the single negotiated UDP socket. Unauthenticated datagrams received on the WireGuard port are dropped unconditionally with zero side-channel response.
- **Zero Third-Party Relays:** User data is never relayed through intermediary proxy nodes. STUN transactions contain zero identification or authentication metadata beyond standard RFC 5389 binding requests.
- **Secret Zeroization:** Private keys are stored in memory and zeroed upon interface destruction. Persistent config files are written with `0600` (read/write owner-only) permission bitmasks.

---

## 12. License

Licensed under the MIT License.
