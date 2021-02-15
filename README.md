# aws-secret

Have key/value from AWS Secrets Manager

## Install

````bash
git clone
cd admission-controller/
go build
docker build . -t adm-controller

cd ..
docker build . -t aws-sidecar
