# Jakapil-cli (Якапиль). Unofficial CLI for Selectel services.

![Jakapil](assets/images/yakapil_nature.jpeg)

## Who is Jakapil

**Jakapil** (or, to be precise, _Jakapil kaniukura_) is a dinosaur whose remains were discovered relatively recently, in 2022, in Argentina.
It was a small herbivorous dinosaur that lived around 100 million years ago.
It was about 1.5 meters long (including its tail) and weighed only 4–7 kg — roughly the size and weight of a domestic cat.
Despite its modest dimensions, it was an armored dinosaur: rows of bony plates and spikes ran along its neck, back, and tail, and it moved on its two hind legs.
In terms of lifestyle, it was more of a "cautious vegetarian" that relied on its protective armor and, possibly, on speed, so among modern pets it can be compared not so much to a cat as to a small dog.

Jakapil is certainly not as popular as Triceratops or Tyrannosaurus Rex, but it is useful and awesome!

But seriously...

**jakapil-cli** is an attempt to create a "universal" CLI in Go for Selectel services. It is inspired by tools from other providers and by the OpenStack CLI. It is intended to use only the existing Selectel Go SDKs. Functionality will be added gradually.
Currently, Managed Kubernetes (MKS) and the container registry are supported. The entire range of the provider's services is on the roadmap.

## Frameworks and libraries used

The following libraries are used at the moment:

- cobra
- viper
- gophercloud
- mks-go
- craas-go

## How to build Jakapil

It is all fairly straightforward:

```
git clone https://repository.example.net/selectel/jakapil-cli
mkdir -p bin/
go build -o bin/ .
```

If you want to use it right away:

```
go install .
```

But first, make sure that $GOPATH/bin is on your PATH environment variable.

## Quick start

Before you start using the utility, you should:

- Create a Selectel account (if you do not have one). [Registration page](https://my.selectel.ru/registration).
- Create a project (if you do not have one) or use the default project. [Project list](https://my.selectel.ru/iam/projects).
- Create a [service user](https://docs.selectel.ru/access-control/user-types/#service-users) with [member](https://docs.selectel.ru/access-control/role-reference/#member) rights in the selected project and in all the projects whose resources you intend to manage.
  Alternatively, you can use a member scoped to the whole account.

Once all the prerequisites are met:

1. Build or install jakapil-cli.
2. Download the generated rc-file for the service user.
3. Source the rc-file.

```
source rc.sh
```

The script will prompt for the service user's password.

4. Issue a token to authenticate against the API.

```
jakapil-cli authenticate token issue
```

5. Start using Jakapil! For example:

```
jakapil-cli mks cluster list --region ru-6
```

will output the list of clusters in the ru-6 cloud pool.
