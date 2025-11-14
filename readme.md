# Tabi


Tabi is the fastest general purpose L1 blockchain and the first parallelized EVM. 


# Documentation
For the most up to date documentation please visit https://docs.tabichain.com/

# Testnet V3
## Get started
**How to validate on the Tabi Testnet-v3**


## Hardware Requirements
**Minimum**
* 16 GB RAM
* 2 TB NVME SSD
* 8 Cores (modern CPU's)

## Operating System 

> Linux (x86_64) or Linux (amd64) Recommended Arch Linux

**Dependencies**
> Prerequisite: go1.18+ required.
* Arch Linux: `pacman -S go`
* Ubuntu: `sudo snap install go --classic`

> Prerequisite: git. 
* Arch Linux: `pacman -S git`
* Ubuntu: `sudo apt-get install git`

> Optional requirement: GNU make. 
* Arch Linux: `pacman -S make`
* Ubuntu: `sudo apt-get install make`

## Tabid Installation Steps

**Clone git repository**

```bash
git clone https://github.com/tabilabs/tabi-v3
cd tabiv-v3
git checkout $VERSION
make build
```
**Generate keys**

* `tabid keys add [key_name]`

* `tabid keys add [key_name] --recover` to regenerate keys with your mnemonic

* `tabid keys add [key_name] --ledger` to generate keys with ledger device

## Start the node

**Start tabid on Linux**

* cd build
* Start the node sudo: `./tabid start`


### Create Validator Transaction
```bash
tabid tx staking create-validator \
--from {{KEY_NAME}} \
--chain-id  \
--moniker="<VALIDATOR_NAME>" \
--commission-max-change-rate=0.01 \
--commission-max-rate=1.0 \
--commission-rate=0.05 \
--details="<description>" \
--security-contact="<contact_information>" \
--website="<your_website>" \
--pubkey $(tabid tendermint show-validator) \
--min-self-delegation="1" \
--amount <token delegation>atabi \
--node localhost:26657
```
