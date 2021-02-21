#!/bin/bash
sudo kubectl apply -f ~/Desktop/pi/pi.yaml
sudo kubectl logs -n kube-system kube-scheduler-master-vm
