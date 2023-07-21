 #!/bin/bash

#init
WORKBASE=/home/jenkins/go/src/infini.sh
WORKDIR=$WORKBASE/$PNAME
DEST=/infini/Sync/Release/$PNAME/stable

if [[ $VERSION =~ NIGHTLY ]]; then
  BUILD_NUMBER=$BUILD_DAY
  DEST=/infini/Sync/Release/$PNAME/snapshot
fi
export DOCKER_CLI_EXPERIMENTAL=enabled

#clean all
cd $WORKSPACE && git clean -fxd

#change framework branch
cd $WORKBASE/framework
git switch -c ecloud-0.3.1 && git pull

#pull code
cd $WORKDIR && git clean -fxd
git stash && git pull origin master

 #build
make clean config build-linux

#copy-configs
cp -rf $WORKBASE/framework/LICENSE $WORKDIR/bin && cat $WORKBASE/framework/NOTICE $WORKDIR/NOTICE > $WORKDIR/bin/NOTICE

cd $WORKDIR/bin
for t in amd64 ; do
  tar zcf ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-linux-$t.tar.gz "${PNAME}-linux-$t" $PNAME.yml LICENSE NOTICE 
done

#git reset
cd $WORKSPACE && git reset --hard
cd $WORKBASE/framework && git checkout master && git reset --hard

