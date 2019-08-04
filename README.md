shard
=====

[![GoDoc](https://godoc.org/lukechampine.com/shard?status.svg)](https://godoc.org/lukechampine.com/shard)
[![Go Report Card](https://goreportcard.com/badge/lukechampine.com/shard)](https://goreportcard.com/report/lukechampine.com/shard)

`shard` is a stripped-down Sia node that records host announcements and
basically nothing else. When run as a public service, it provides these
announcements to clients, sparing them the trouble of running a node themselves.

Host announcements are the mechanism by which Sia hosts associate their public
key with their current IP address or domain name. Subsequent announcements for
the same public key replace earlier ones. A `shard` server is thus analogous to
a DNS resolver: it resolves public keys to network addresses.

While this is convenient, it also introduces a degree of trust. The amount of
trust is (nearly) minimized, but still present. The host announcements reported
by `shard` are signed by their public key, so `shard` cannot forge the
announcements themselves. However, it *can* report outdated information. For
example, a host might switch from address A to B, and make a new announcement
declaring so; but a malicious `shard` server can continue providing the old
announcement. `shard` also provides the current blockheight, and it can lie
about that too. Fortunately, the consequences of these lies are minor, and the
lies can be easily detected by comparing the responses of two or more servers.

The API for a `shard` server is as follows:

| Endpoint        | Type        | Description                                                           |
|-----------------|-------------|-----------------------------------------------------------------------|
| `/synced`       | JSON bool   | If false, the server's responses may be outdated.                     |
| `/height`       | JSON number | The current height of the Sia blockchain.                             |
| `/host/:pubkey` | Binary      | The most recent announcement of the host with the request public key. |

Announcements are encoded in binary using the Sia encoding format. An informal
description of the announcement structure is as follows:

| Field       | Size (bytes) | Description                                                        |
|-------------|--------------|--------------------------------------------------------------------|
| Magic       | 16           | The string "HostAnnouncement".                                     |
| Address Len | 8            | The length of the subsequent address.                              |
| Address     | (variable)   | The host's network address, as a string.                           |
| Pubkey Type | 16           | The string "ed25519\0\0\0\0\0\0\0\0\0".                            |
| Pubkey Len  | 8            | The length of the subsequent key; always 32.                       |
| Pubkey      | 32           | The raw bytes of the ed25519 public key.                           |
| Signature   | 64           | The ed25519 signature of the BLAKE-2B hash of the preceding bytes. |
