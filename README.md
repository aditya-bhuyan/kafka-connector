## Kafka-connector

[![Go Report Card](https://goreportcard.com/badge/github.com/openfaas/kafka-connector)](https://goreportcard.com/report/github.com/openfaas/kafka-connector) [![Build
Status](https://travis-ci.org/openfaas/kafka-connector.svg?branch=master)](https://travis-ci.org/openfaas/kafka-connector) [![GoDoc](https://godoc.org/github.com/openfaas/kafka-connector?status.svg)](https://godoc.org/github.com/openfaas/kafka-connector) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![OpenFaaS](https://img.shields.io/badge/openfaas-serverless-blue.svg)](https://www.openfaas.com)

The Kafka connector connects OpenFaaS functions to Kafka topics.

Goals:

* Allow functions to subscribe to Kafka topics
* Ingest data from Kafka and execute functions
* Work with the OpenFaaS REST API / Gateway
* Formulate and validate a generic "connector-pattern" to be used for various event sources like Kafka, AWS SNS, RabbitMQ etc

## Conceptual design

![Conceptual diagram](./images/overview.svg)

This diagram shows the Kafka connector on the left hand side. It is responsible for querying the API Gateway for a list of functions. It will then build up a map or table of which functions have advertised an interested in which topics.

When the connector hears a message on an advertised topic it will look that up in the reference table and find out which functions it needs to invoke. Functions are invoked only once and there is no re-try mechanism. The result is printed to the logs of the Kafka connector process.

The cache or list of functions <-> topics is refreshed on a periodic basis.

## Building

```
export TAG=0.3.3
make build push
```

## Try it out

### Deploy on Kubernetes

The following instructions show how to run `kafka-connector` on Kubernetes.

Deploy a function with a `topic` annotation:

```bash
$ faas store deploy figlet --annotation topic="faas-request" --gateway <faas-netes-gateway-url>
```

If deploying via a `stack.yml` then add an annotation key called `topic` with a value of `faas-request`.

```yaml
  my-function:
...
    annotations:
      topic: payment
```

For further details, please see the [YAML Reference](https://docs.openfaas.com/reference/yaml/#function-annotations) section of the [OpenFaaS Docs](https://docs.openfaas.com/)

Deploy Kafka:

You can run the zookeeper, kafka-broker and kafka-connector pods with:

```bash
kubectl apply -f ./yaml/kubernetes/
```

If you already have Kafka then update `./yaml/kubernetes/connector-dep.yml` with your Kafka broker address and then deploy only that file:

```bash
kubectl apply -f ./yaml/kubernetes/connector-dep.yml
```

Alternatively you can use the [kafka-connector helm chart](https://github.com/openfaas/faas-netes/tree/master/chart/kafka-connector)

Then use the broker to send messages to the topic:

```bash
BROKER=$(kubectl get pods -n openfaas -l component=kafka-broker -o name|cut -d'/' -f2)
kubectl exec -n openfaas -t -i $BROKER -- /opt/kafka_2.12-0.11.0.1/bin/kafka-console-producer.sh --broker-list kafka:9092 --topic faas-request

hello world
```
Once you have connected, each new line will be a message published.

If you have an error such as the below, then wait a little longer and try again.

```
error: unable to upgrade connection: container not found ("kafka")
```

You can verify the proper path of the publisher script by getting the shell of the running broker:
```
$ kubectl exec -n openfaas -t -i $BROKER -- sh 
/ # find | grep producer

./opt/kafka_2.12-0.11.0.1/config/producer.properties
./opt/kafka_2.12-0.11.0.1/bin/kafka-verifiable-producer.sh
./opt/kafka_2.12-0.11.0.1/bin/kafka-producer-perf-test.sh
./opt/kafka_2.12-0.11.0.1/bin/kafka-console-producer.sh
```

With the `$BROKER` variable still set, view the list of messages on a given topic:

```
$ kubectl exec -n openfaas -t -i $BROKER -- /opt/kafka_2.12-0.11.0.1/bin/kafka-console-consumer.sh --bootstrap-server kafka:9092 --topic faas-request --from-beginning
```

Now check the connector logs to see the figlet function was invoked:

```bash
CONNECTOR=$(kubectl get pods -n openfaas -o name|grep kafka-connector|cut -d'/' -f2)
kubectl logs -n openfaas -f --tail 100 $CONNECTOR

2018/08/08 16:54:35 Binding to topics: [faas-request]
2018/08/08 16:54:38 Syncing topic map
Rebalanced: &{Type:rebalance start Claimed:map[] Released:map[] Current:map[]}
Rebalanced: &{Type:rebalance OK Claimed:map[faas-request:[0]] Released:map[] Current:map[faas-request:[0]]}

2018/08/08 16:54:41 Syncing topic map
2018/08/08 16:54:44 Syncing topic map
2018/08/08 16:54:47 Syncing topic map

[#53753] Received on [faas-request,0]: 'hello world.'
2018/08/08 16:57:41 Invoke function: figlet
2018/08/08 16:57:42 Response [200] from figlet  
 _          _ _                            _     _ 
| |__   ___| | | ___   __      _____  _ __| | __| |
| '_ \ / _ \ | |/ _ \  \ \ /\ / / _ \| '__| |/ _` |
| | | |  __/ | | (_) |  \ V  V / (_) | |  | | (_| |
|_| |_|\___|_|_|\___/    \_/\_/ \___/|_|  |_|\__,_|
                                                   
```

### Deploy on Swarm

Deploy the stack which contains Kafka and the connector:

```bash
docker stack deploy kafka -c ./yaml/connector-swarm.yml
```

* Deploy or update a function so it has an annotation `topic=faas-request` or some other topic

As an example:

```shell
$ faas store deploy figlet --annotation topic="faas-request"
```

The function can advertise more than one topic by using a comma-separated list i.e. `topic=topic1,topic2,topic3`

* Publish some messages to the topic in question i.e. `faas-request`

Instructions are below for publishing messages

* Watch the logs of the kafka-connector

## Trigger a function via a topic with `exec`

You can use the kafka container to send a message to the topic.

```
SERVICE_NAME=kafka_kafka
TASK_ID=$(docker service ps --filter 'desired-state=running' $SERVICE_NAME -q)
CONTAINER_ID=$(docker inspect --format '{{ .Status.ContainerStatus.ContainerID }}' $TASK_ID)
docker exec -it $CONTAINER_ID kafka-console-producer --broker-list kafka:9092 --topic faas-request

hello world
```

## Generate load on the topic

You can use the sample application in the producer folder to generate load for a topic.

Make sure you have some functions advertising an interest in that topic so that they receive the data.

> Note: the producer *must* run inside the Kubernetes or Swarm cluster in order to be able to access the broker(s).

## Monitor the results

Once you have generated some requests or start a load-test you can watch the function invocation rate increasing in Prometheus or watch the logs of the container.

### Prometheus

You can open the Prometheus metrics or Grafana dashboard for OpenFaaS to see the functions being invoked.

### Watch the logs

```
docker service logs kafka_connector -f

Topics
[__consumer_offsets faas-request] <nil>
2018/03/24 12:42:58 Binding to topics: [faas-request]
Rebalanced: &{Type:rebalance start Claimed:map[] Released:map[] Current:map[]}
Rebalanced: &{Type:rebalance OK Claimed:map[faas-request:[0]] Released:map[] Current:map[faas-request:[0]]}

2018/03/24 17:04:40 Syncing topic map
[#53694] Received on [faas-request,0]: 'Test the function.'
2018/03/24 17:04:41 Invoke function: figlet
2018/03/24 17:04:42 Response [200] from figlet  
|_   _|__  ___| |_  | |_| |__   ___   / _|_   _ _ __   ___| |_(_) ___  _ __    
  | |/ _ \/ __| __| | __| '_ \ / _ \ | |_| | | | '_ \ / __| __| |/ _ \| '_ \   
  | |  __/\__ \ |_  | |_| | | |  __/ |  _| |_| | | | | (__| |_| | (_) | | | |_ 
  |_|\___||___/\__|  \__|_| |_|\___| |_|  \__,_|_| |_|\___|\__|_|\___/|_| |_(_)
                  
```


> Note: If the broker has a different name from `kafka` you can pass the `broker_host` environmental variable. This exclude the port.

## Configuration

Configure the software as required with these environment variables.

> Note: the helm chart options are written differently and exist in the upstream chart (see above for a link).

| Environment variable      | description                                                 |
| ------------------------- |----------------------------------------------------------   |
| `upstream_timeout`        | Go duration - maximum timeout for upstream function call    |
| `asynchronous_invocation` | For long running or slow functions, offload to asychronous function invocations and carry on processing the stream |
| `rebuild_interval`        | Go duration - interval for rebuilding function to topic map |
| `topics`                  | Topics to which the connector will bind                     |
| `gateway_url`             | The URL for the API gateway i.e. http://gateway:8080 or http://gateway.openfaas:8080 for Kubernetes       |
| `broker_host`             | Default is `kafka:9092`. You can configure this with our own host with port |
| `print_response`          | Default is `true` - this will output information about the response of calling a function in the logs, including the HTTP status, topic that triggered invocation, the function name, and the length of the response body in bytes |
| `print_response_body`     | Default is `true` - this will print the body of the response of calling a function to stdout |
| `topic_delimiter`         | Default is `,` - Specifies character upon which to split the `topic` annotation when subscribing a function to mulitple topics |
