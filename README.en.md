# Jakapil-cli. Unofficial CLI for Selectel services

Unofficial CLI for Selectel cloud services.

[Русский](./README.ru.md) | **English**

![Jakapil](assets/images/yakapil_nature.jpeg)

[![License](https://img.shields.io/github/license/readme-SVG/readme-badges?style=flat-square)](LICENSE)
![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=flat-square&logo=go&logoColor=white)
[![Release](https://img.shields.io/github/v/release/kot1grun/jakapil-cli?style=flat-square)](https://github.com/kot1grun/jakapil-cli/releases)

## Table of contents

- [What is Jakapil?](#what-is-it)
- [Installation](#install)
- [Build](#build)
- [Quick start](#quick-start)

<a id="what-is-it"></a>

## Who is Jakapil

**Jakapil** (or, to be precise, _Jakapil kaniukura_) is a dinosaur whose remains were discovered relatively recently, in 2022, in Argentina.
It was a small herbivorous dinosaur that lived around 100 million years ago.
It was about 1.5 meters long (including its tail) and weighed only 4–7 kg, that is, it was roughly the size and weight of a domestic cat.
Despite its modest dimensions, it was an armored dinosaur: rows of bony plates and spikes ran along its neck, back, and tail, and it moved on its two hind legs.
In terms of lifestyle, it was more of a "cautious vegetarian" that relied on its protective armor and, possibly, on speed, so among modern pets it can be compared not so much to a cat as to a small dog.

Jakapil is certainly not as popular as Triceratops or Tyrannosaurus Rex, but it is useful and awesome!

But seriously...

**jakapil-cli** is an attempt to create a "universal" CLI in Go for Selectel services. It is inspired by tools from other providers and by the OpenStack CLI. It is intended to use only the existing Selectel Go SDKs. Functionality will be added gradually.
At the moment, of the provider-specific services, managed Kubernetes (MKS) and the container registry (CRaaS) are supported.

The ultimate goal is to support all the services of the Selectel provider.

IaaS support includes:

- managing the lifecycle of VMs and related entities (images, volumes, flavors, SSH keys)
- managing the lifecycle of network entities (networks, subnets, ports, routers, "floating" addresses)
- managing direct public addresses (public ports, the Public Net service)

**Not yet implemented**, but planned in the near future:

- IAM
- ~~public subnets~~
- ~~direct public addresses~~
- DNS hosting
- private DNS service
- load balancers
- security groups
- basic cloud firewall
- dedicated servers
- global router

The rest will be tracked on the roadmap.

## Frameworks and libraries used

Of the main ones, the following libraries are used:

- cobra
- viper
- gophercloud
- go-selvpcclient
- mks-go
- craas-go
- globalrouter-go
- public-net-api-go

<a id="install"></a>

## How to install Jakapil?

### macOS

The best option is to use the [Homebrew](https://brew.sh/) package manager.

```shell
brew tap kot1grun/tap
brew trust kot1grun/tap
brew update
brew install jakapil-cli
```

Another option is to download a precompiled binary for [Apple silicon](https://github.com/kot1grun/jakapil-cli/releases/latest/download/jakapil-cli_Darwin_arm64.tar.gz) or [Intel x86](https://github.com/kot1grun/jakapil-cli/releases/latest/download/jakapil-cli_Darwin_x86_64.tar.gz), unpack it and copy it into PATH.

### Linux

For rpm-based and deb-based distributions, the corresponding packages are available in the [releases](https://github.com/kot1grun/jakapil-cli/releases).
For other distributions you can use a binary compiled for the required architecture.

### Windows

At the moment, only the precompiled files in a zip archive are available.
For the x86-64 architecture — [via this link](https://github.com/kot1grun/jakapil-cli/releases/latest/download/jakapil-cli_Windows_x86_64.zip), for ARM64 — [here](https://github.com/kot1grun/jakapil-cli/releases/latest/download/jakapil-cli_Windows_arm64.zip).
If someone is still on x86 — download [here](https://github.com/kot1grun/jakapil-cli/releases/latest/download/jakapil-cli_Windows_i386.zip).

<a id="build"></a>

## How to build Jakapil

You need [Go installed](https://go.dev/doc/install).
Further on, it is all fairly straightforward:

```shell
git clone https://github.com/kot1grun/jakapil-cli
mkdir -p bin/
go build -o bin/ .
```

It is more convenient to use the Makefile. In that case, you need GNU Make installed.

```shell
make build
```

<a id="quick-start"></a>

## Quick start

Before you start using the utility, you should:

- Create a Selectel account (if you do not have one). The registration page [is located here](https://my.selectel.ru/registration).
- Create a project (if you do not have one) or use the default project. The project list [is located here](https://my.selectel.ru/iam/projects).
- Create a [service user](https://docs.selectel.ru/access-control/user-types/#service-users) with [member](https://docs.selectel.ru/access-control/role-reference/#member) rights in the selected project and in all the projects whose resources you intend to manage.
  Alternatively, you can use a member scoped to the whole account.

Once all the prerequisites are met:

1. Build or install jakapil-cli.
2. Download the generated rc-file for the service user.
3. Import the rc-file.

```shell
source rc.sh
```

The script will prompt for the service user's password.

4. Issue a token to authenticate against the API.

```shell
jakapil-cli authenticate token issue --account-scope --project-scope --ask-pass
```

The utility will prompt for the service user's password and, if the password is correct, will issue and save tokens scoped to "project" and "account".
Strictly speaking, entering the password here is not required: the environment variable will be used when importing rc.sh. But an extra check does not hurt.

5. Start using Jakapil! For example:

```shell
jakapil-cli mks cluster list --region ru-6
```

will output the list of Kubernetes clusters in the ru-6 cloud pool.

## Jakapil's nice features

Apart from the obvious fact that jakapil-cli is written in Go and everything is statically compiled into a single file, it is worth noting:

- output in three formats: a human-readable table, YAML and JSON
- the _--all-regions_ flag in listings, which shows the service instances (be it a VM, a network or a Kubernetes cluster) across all available regions
- the ability to create highfreq flavors for VMs
- a progressbar when downloading/uploading images, showing the remaining and elapsed time and the speed

## Features (or, if you prefer, the "unpleasant side") of Jakapil

- the utility (for now) does not know names and works only with service UUIDs
