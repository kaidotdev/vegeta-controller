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

$ kubectl logs -l app=attack-sample-job
Requests      [total, rate, throughput]  500, 50.10, 38.51
Duration      [total, attack, wait]      12.984487191s, 9.979884149s, 3.004603042s
Latencies     [mean, 50, 95, 99, max]    2.003985261s, 2.081863241s, 3.005786028s, 3.02320498s, 3.053911426s
Bytes In      [total, mean]              121500, 243.00
Bytes Out     [total, mean]              0, 0.00
Success       [ratio]                    100.00%
Status Codes  [code:count]               200:500
Error Set:
Requests      [total, rate, throughput]  500, 50.10, 38.51
Duration      [total, attack, wait]      12.982401798s, 9.979968589s, 3.002433209s
Latencies     [mean, 50, 95, 99, max]    2.002969191s, 2.068438165s, 3.004653479s, 3.01070406s, 3.032810373s
Bytes In      [total, mean]              121500, 243.00
Bytes Out     [total, mean]              0, 0.00
Success       [ratio]                    100.00%
Status Codes  [code:count]               200:500
Error Set:
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
    {"method":"GET","url":"http://httpbin/delay/1"}
    {"method":"GET","url":"http://httpbin/delay/3"}
  output: text
  option:
    duration: 10s
    rate: 10
    connections: 10000
    timeout: 10s
    workers: 10
    format: json
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

See CRD for other available fields and detailed descriptions: [vegeta.kaidotdev.github.io_attacks.yaml](https://github.com/kaidotdev/vegeta-controller/blob/master/manifests/crd/vegeta.kaidotdev.github.io_attacks.yaml)

## How to develop

### Generate CRD from controller-gen

```sh
$ make gen
```
