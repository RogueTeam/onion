# onion

Tor and I2P inspired, powered by [libp2p](https://libp2p.io/) hidden network service. Entirely written from scratch (**This is not a port of any of those**) to prove a smaller code base can archieve the same.

![logo](images/logo.webp)

## IMPORTANT: Onion needs new hands

If you have experience working with Go, or maintaining I2P or Tor, OPSEC experience, a look at the source code could allow us to check if the privacy logic is being preserved. Remember, this isn't the final solution, **just a proposal to the community**. So feel free to **critique** my approach and **argue** for a better way of doing things. Maybe I've missed something you haven't!

## Abstract

Tor and I2P are robust solutions for anonymous networking.

## Features

- [x] Circuits

- [x] Exit nodes

- [ ] Hidden services

## Licensing

Fork this and do whatever you want. All of it is under UNLICENSED so feel free to copy, redistribute, sell or whatever you want but most important share with others.

## How it works

### Truth of source

Uses DHT as peer sharing and discovery. Leveraging this important feature in IPFS's DHT protocol.

### Nodes

Every node is a relay. Even those without a public IP. This is possible thanks to the libp2p's feature of circuits. Allowing nodes behind a NAT use a temporary address to communicate with the outside world. Of course Public nodes has more bandwidth advantages but even so, this allow anyone to be part of the routing process.

### SPAN prevention

To prevent span of requests to nodes. Each nodes implements a configurable PoW algorithm. Allowing the server admin to set the difficulty needed to forward traffic using his node

### Circuits

Chain multiple network participants into a single circuit. Only the first connected peer knows your LIBP2P Unique ID and IP the rest can only see the last peer of the chain and identify you by a Hidden identity that your node generates.

Each connection to each peer is connected from your point to that peer. Meaning no middle peer can see what both points are talking about.

### Exit nodes

**Feature disabled by default**. Special nodes that allow connected peers to access host ourside the network.

## Why not Tor?

**Technically**, developing services that have Tor embedded is almost impossible outside the C world. Developers always need to ship the Tor binary as a separate file instead of just importing what they actually need from a possible library. Also, including in this same binary a lot of features that the project actually doesn't need. The new Rust port is allowing this but again, mostly in the Rust world.

**Documentation:** Tor's protocol documentation is not great. If it were, we could see other fully working implementations just like I2P. But porting the original code or trying to write the protocol using the official docs as a reference is **hard**, not because it's not complete, but rather because of the spaghetti documentation. Direct source code porting could be the other option, but the way it was written makes it even harder than the protocol docs. The best attempt could be trying directly against the new Rust port made by the official team, since this implementation is cleaner. But sadly, it's not an option just yet because its APIs are not currently stable, as the project has not fully released to production levels.

**Politically**, here comes the sad part. While Tor is the industry standard, it is moving away from what I thought was a secure development place. The maintainers recently pushed changes into the Tor Browser that allow adversaries to fingerprint users.

**Other comments:** Apart from this, while Rust is the "wannabe" language for almost all projects, it reduces the amount of people who are able to contribute to the project's heart. (Please, Rust guys, don't attack me!) But let's be honest: Rust is harder than any other language. While it for sure could make development more secure, avoiding a lot of unsafe operations that are common in C and C++, making the compiler happy and the limited amount of devs make community-based contributions harder, forcing the project to be reviewed by even fewer eyes (at least at the moment until Rust replaces in the future the Cs).

## Status

This project is developed by a single person and is provided as-is, without any warranty or responsibility for its use.
