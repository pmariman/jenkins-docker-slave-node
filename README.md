# jenkins-docker-slave-node


## Application Container

```
$ docker build -t jenkins-slave:latest .
```


## Test

```
$ docker run --rm -ti -v /var/run/docker.sock:/var/run/docker.sock\
            -e SLAVE_IP="172.17.0.1" -e SLAVE_PORT="8080" -e SLAVE_USER="user" \
            -e SLAVE_PASSWD="pass" -e SLAVE_EXECUTORS=2 -e SLAVE_NAME="tiny-client" \
            jenkins-slave:latest
```
