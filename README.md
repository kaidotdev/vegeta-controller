# VegetaController

VegetaController is Kubernetes Custom Controller that allows distributed execution of [tsenart/vegeta](https://github.com/tsenart/vegeta).

## Installation

```shell
$ kubectl apply -k manifests
```

## Usage

Applying the following manifest enables distributed execution of vegeta.

```shell
$ cat <<EOS | kubectl apply -f -
apiVersion: vegeta.kaidotdev.github.io/v1
kind: Attack
metadata:
  name: attack-sample
spec:
  parallelism: 2
  scenario: |-
    GET http://httpbin/delay/1
    GET http://httpbin/delay/3
  output: text
EOS
$ kubectl get attack attack-sample
NAME            AGE
attack-sample   7s
$ kubectl get job attack-sample-job
NAME                COMPLETIONS   DURATION   AGE
attack-sample-job   0/1 of 2      10s        10s
$ kubectl get pod | grep attack-sample
attack-sample-job-7487s          1/1     Running   0          13s
attack-sample-job-z879t          1/1     Running   0          13s
```

You can also specify vegeta options via manifest,

```yaml
apiVersion: vegeta.kaidotdev.github.io/v1
kind: Attack
metadata:
  name: attack-sample
spec:
  parallelism: 2
  scenario: |-
    GET http://httpbin/delay/1
    GET http://httpbin/delay/3
  output: text
  option:
    duration: 10s
    rate: 10
    connections: 10000
    timeout: 10s
    workers: 10
```

if you are using istio etc., you can control their sidecar through pod annotation.

```yaml
apiVersion: vegeta.kaidotdev.github.io/v1
kind: Attack
metadata:
  name: attack-sample
spec:
  parallelism: 2
  scenario: |-
    GET http://httpbin/delay/1
    GET http://httpbin/delay/3
  output: text
  template:
    metadata:
      labels:
        version: v1
      annotations:
        sidecar.istio.io/inject: "true"
```
