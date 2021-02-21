#!/bin/bash
sudo echo "FROM busybox
ADD ./local/bin/linux/amd64/kube-scheduler /usr/local/bin/kube-scheduler" >./_output/Dockerfile
sudo docker build -t 0yukali0/bab-scheduler:v1.0 ./_output
sudo docker push 0yukali0/bab-scheduler
