 #!/bin/bash

#init
WORKBASE=/home/jenkins/go/src/infini.sh
WORKDIR=$WORKBASE/agent
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

for t in amd64 386 mips mipsle mips64 mips64le arm5 arm6 arm7 arm64 amd64 loong64 riscv64 ; do
  echo "package-linux-$t"
  tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-$t.tar.gz agent-linux-$t agent.yml LICENSE NOTICE 
done

for t in mac-amd64 mac-arm64 windows-amd64 windows-386 ; do
  echo "package-$t"
  cd $WORKDIR/bin && zip -r ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-$t.zip agent-$t agent.yml LICENSE NOTICE
done

#docker build
cd $WORKDIR/bin
cat <<"EOF">Dockerfile
FROM --platform=$TARGETPLATFORM alpine:3.16.5

COPY ["agent-linux-amd64", "agent.yml", "./"]
CMD ["/agent-linux-amd64"]
EOF
docker buildx build -t infinilabs/agent-amd64:latest --platform=linux/amd64 -o type=docker .
cat <<"EOF">Dockerfile
FROM --platform=$TARGETPLATFORM alpine:3.16.5

COPY ["agent-linux-arm64", "agent.yml", "./"]
CMD ["/agent-linux-arm64"]
EOF
docker buildx build -t infinilabs/agent-arm64:latest --platform=linux/arm64 -o type=docker .

#推送镜像
docker push infinilabs/agent-amd64:latest
docker push infinilabs/agent-arm64:latest

docker tag infinilabs/agent-amd64:latest infinilabs/agent-amd64:$VERSION-$BUILD_NUMBE
docker tag infinilabs/agent-amd64:latest infinilabs/agent-arm64:$VERSION-$BUILD_NUMBE
docker push infinilabs/agent-amd64:$VERSION-$BUILD_NUMBE
docker push infinilabs/agent-arm64:$VERSION-$BUILD_NUMBE

docker buildx imagetools create -t infinilabs/agent:latest \
    infinilabs/agent-arm64:latest \
    infinilabs/agent-amd64:latest

docker buildx imagetools create -t infinilabs/agent:$VERSION-$BUILD_NUMBE \
    infinilabs/agent-arm64:$VERSION-$BUILD_NUMBE \
    infinilabs/agent-amd64:$VERSION-$BUILD_NUMBE

#清理git
git reset --hard

#清理2周之前的镜像
docker images |grep agent |grep "weeks ago" |awk '{print $3}' |xargs docker rmi