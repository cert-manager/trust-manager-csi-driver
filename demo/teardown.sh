#!/bin/bash

set -eu

kubectl delete namespace example 
kubectl delete bundle example.com