**[Installation](#setup-instructions)** |
**[How to use](#how-to-use)** |
**[How to write tests](#how-to-include-benchmarks)**|

# kubectl-mtb
> kubectl plugin to validate the the multi-tenancy in the K8s cluster.
> This tool automates behavioral and configuration checks on existing clusters which will help K8s users validate whether their
clusters are set up correctly for multi-tenancy.

## Demo
[![asciicast](https://asciinema.org/a/5J0bA6AIIk8Y0mH8w3UYSRkxK.svg)](https://asciinema.org/a/5J0bA6AIIk8Y0mH8w3UYSRkxK)


## Setup Instructions

**Prerequisites** : Make sure you have working GO environment in your system.

kubectl-mtb can be installed by running

```bash
$ go get sigs.k8s.io/multi-tenancy/benchmarks/kubectl-mtb
```
or by cloning this repository, and running

```bash 
$ make kubectl-mtb
```

## How to use

To list available benchmarks :

```bash
$ kubectl-mtb get benchmarks
```

To run the available benchmarks:

```bash
$ kubectl-mtb test benchmarks -n "name of tenant-admin namespace" -t "name of tenant service account"
```
Example: 

```bash
$ kubectl-mtb test benchmarks -n tenant0admin -t system:serviceaccount:tenant0admin:t0-admin0
```
 
## How to include bechmarks

You can use mtb-builder to include/write other benchmarks.

Run the following command to build mtb-builder. 

```
$ make builder
```
The generated binary will create the relevant templates, needed to write the bechmark as well as associated unittest.

**Example :**

```
$ ./mtb-builder create block multitenant resources -p 1
```
Here,  `create block multitenant resources` is name of the benchmark and `-p` flag is used here to mention the profile level. The above command will generate a directory named `create_block_multitenant_resources` under which following files would be present.

- config.yaml
- create_block_multitenant_resources_test.go
- create_block_multitenant_resources.go





### Make commands

- **make kubectl-mtb** : To build the project and copy the binary file to the PATH.
- **make generate** : To convert benchmarks config yaml files into static assets.
- **make readme** : To generate each benchmark README files from their respective config yaml files to serve as a docs for benchmark.
