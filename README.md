# jenkins-docker-slave-node


## Application Binary

```
$ docker run --rm -ti -v $(pwd)/src/:/go/src/jdsn/ -w /go/src/jdsn/ golang:latest make
```


## Application Container

```
$ docker build -t jenkins-slave:latest .
```


## Test

```
$ docker run --rm -ti -e SLAVE_IP="172.17.0.1" -e SLAVE_PORT="8090" jenkins-slave:latest
```
