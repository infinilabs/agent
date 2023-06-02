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
cp -rf $WORKBASE/framework/LICENSE $WORKDIR/bin && cat $WORKBASE/framework/NOTICE $WORKDIR/NOTICE > $WORKDIR/bin/NOTICE

cd $WORKDIR/bin && ls -lrt .
for t in amd64 386 mips mipsle mips64 mips64le arm5 arm6 arm7 arm64 amd64 loong64 riscv64 ; do
  echo "package-linux-$t"
  tar -zcvf ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-linux-$t.tar.gz "${PNAME}-linux-$t" $PNAME.yml LICENSE NOTICE 
done

for t in mac-amd64 mac-arm64 windows-amd64 windows-386 ; do
  echo "package-$t"
  cd $WORKDIR/bin && zip -r ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-$t.zip $PNAME-$t $PNAME.yml LICENSE NOTICE
done

#build image & push
for t in amd64 arm64 ; do

  cat <<EOF>Dockerfile
MAINTANIER "hardy <luohoufu@gmail.com>"
FROM --platform=linux/$t alpine:3.16.5
WORKDIR /opt/$PNAME

COPY ["$PNAME-linux-$t", "$PNAME.yml", "./"]

CMD ["/opt/$PNAME/$PNAME-linux-$t"]
EOF

  docker buildx build -t infinilabs/$PNAME-$t:latest --platform=linux/$t -o type=docker .

  docker tag infinilabs/$PNAME-$t:latest infinilabs/$PNAME-t:$VERSION-$BUILD_NUMBE
  docker push infinilabs/$PNAME-$t:latest
  docker push infinilabs/$PNAME-$t:$VERSION-$BUILD_NUMBE
done

#composite tag
docker buildx imagetools create -t infinilabs/$PNAME:latest \
    infinilabs/$PNAME-arm64:latest \
    infinilabs/$PNAME-amd64:latest

docker buildx imagetools create -t infinilabs/$PNAME:$VERSION-$BUILD_NUMBE \
    infinilabs/$PNAME-arm64:$VERSION-$BUILD_NUMBE \
    infinilabs/$PNAME-amd64:$VERSION-$BUILD_NUMBE

#git reset
git reset --hard

#clen weeks ago image
NEEDCLEN=$(docker images |grep "$PNAME" |grep "weeks ago")
if [ ! -z "$NEEDCLEN" ]; then
  docker images |grep "$PNAME" |grep "weeks ago" |awk '{print $3}' |xargs docker rmi
fi