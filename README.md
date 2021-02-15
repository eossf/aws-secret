# aws-secret

Have key/value from AWS Secrets Manager

## Install

````bash
git clone
git pull
cd admission-controller/
go build
docker build . -t adm-controller

cd ..
docker build . -t aws-sidecar
````

the images are in docker hub smetairie
````bash
docker tag adm-controller smetairie/adm-controller
docker push smetairie/adm-controller

docker tag aws-sidecar smetairie/aws-sidecar
docker push smetairie/aws-sidecar
````
