#!/bin/zsh

VERSION=0.5

docker tag mash-auth-base-v${VERSION} nexus:5001/mash-auth/mash-auth-base:latest
docker tag mash-auth-base-v${VERSION} nexus:5001/mash-auth/mash-auth-base:v${VERSION}
docker push --all-tags nexus:5001/mash-auth/mash-auth-base