
# ICAP Adaptation Service

ICAP Adaptation Service is an in-cluster process to monitor a RabbitMQ instance and to spin up the Request Processing pod to facilitate the Rebuild functionality of the Glasswall ICAP offering.

## Built With
- Go
- Helm
- Docker

# Getting Started
To get a local instance up and running follow these steps:

## Prerequisites
- Kubernetes cluster running locally
- Helm installed
- kubectl installed

## Installation
1. Clone the repo

```
git clone https://github.com/filetrust/icap-adaptation-service.git
```

2. cd to root
```
cd .\icap-adaptation-service\
```

3. Create namespace on cluster

```
kubectl create ns icap-adaptation
```

4. Create Docker Registry Secret on cluster to allow pulling the private icap-request-processing image. 

``` 
kubectl create -n icap-adaptation secret docker-registry regcred --docker-server=https://index.docker.io/v1/ --docker-username=<DOCKER HUB USERNAME> --docker-password='<DOCKER HUB PASSWORD>'--docker-email=<DOCKER HUB EMAIL> 
```

5. Update glasswallsourcevolume & glasswalltargetvolume to point to local test file directory
```
    path: "/run/desktop/mnt/host/<drive>/<folder>"
```

6. Run Helm Install
```
helm install . --namespace icap-adaptation --generate-name
```

# Usage

To start the adaptation process send a message to the RabbitMQ with the following:

```
Exchange: adaptation-exchange
Routing Key: adaptation-request
Body: '{"file-id": "<FILE NAME>", "request-mode": "respmod", "source-file-location": "/input/<FILE NAME>", "rebuilt-file-location": "/output/<FILE NAME>"}'
```
