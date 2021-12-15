COMBCore
--------
Prototype Wallet/Node for Haircomb.

Plans
-----
- [x] Mining commits directly from BTC block data.
- [x] Corruption detection and correction (per Block).
- [x] RPC control interface.
- [ ] Mining from remote BTC peers.
- [ ] P2P commit sharing and verification.
- [ ] Automatic signature committing.
- [ ] Opportunistic claiming.
- [ ] Qt GUI (Bitcoin Core rip off).
- [ ] COMB/BTC trading (requires a fork).

Building
--------
```bash
git clone https://github.com/dyoform/combcore
cd combcore
git submodule init
git submodule update
go get
go build
./combcore
```