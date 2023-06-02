 #!/bin/bash

#init
PNAME=agent
WORKBASE=/home/jenkins/go/src/infini.sh
WORKDIR=$WORKBASE/$PNAME
export DOCKER_CLI_EXPERIMENTAL=enabled

#pull code
cd $WORKDIR && git clean -fxd
git stash && git pull origin master

 #build
make clean config build-linux
make config build-arm
make config build-darwin
make config build-win
GOROOT="/infini/go-pkgs/go-loongarch" GOPATH="/home/jenkins/go" make build-linux-loong64

#copy-configs
cd $WORKDIR && cp $WORKBASE/framework/LICENSE bin && cat $WORKBASE/framework/NOTICE NOTICE > bin/NOTICE

cd $WORKDIR/bin
for t in amd64 386 mips mipsle mips64 mips64le arm5 arm6 arm7 arm64 amd64 loong64 riscv64 ; do
  echo "package-linux-$t"
  tar cfz ${WORKSPACE}/$PANME-$VERSION-$BUILD_NUMBER-linux-$t.tar.gz $PANME-linux-$t $PANME.yml LICENSE NOTICE 
done

for t in mac-amd64 mac-arm64 windows-amd64 windows-386 ; do
  echo "package-$t"
  cd $WORKDIR/bin && zip -r ${WORKSPACE}/$PANME-$VERSION-$BUILD_NUMBER-$t.zip $PANME-$t $PANME.yml LICENSE NOTICE
done

#build image & push
for t in amd64 arm64 ; do

  cat <<EOF>Dockerfile
MAINTANIER "hardy <luohoufu@gmail.com>"
FROM --platform=linux/$t alpine:3.16.5
WORKDIR /opt/$PANME

COPY ["$PANME-linux-$t", "$PANME.yml", "./"]

CMD ["/opt/$PANME/$PANME-linux-$t"]
EOF

  docker buildx build -t infinilabs/$PANME-$t:latest --platform=linux/$t -o type=docker .

  docker tag infinilabs/$PANME-$t:latest infinilabs/$PANME-t:$VERSION-$BUILD_NUMBE
  docker push infinilabs/$PANME-$t:latest
  docker push infinilabs/$PANME-$t:$VERSION-$BUILD_NUMBE
done

#composite tag
docker buildx imagetools create -t infinilabs/$PANME:latest \
    infinilabs/$PANME-arm64:latest \
    infinilabs/$PANME-amd64:latest

docker buildx imagetools create -t infinilabs/$PANME:$VERSION-$BUILD_NUMBE \
    infinilabs/$PANME-arm64:$VERSION-$BUILD_NUMBE \
    infinilabs/$PANME-amd64:$VERSION-$BUILD_NUMBE

#git reset
git reset --hard

#clen weeks ago image
docker images |grep $PANME |grep "weeks ago" |awk '{print $3}' |xargs docker rmi