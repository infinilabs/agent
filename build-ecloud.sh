#!/bin/bash

#init
WORKBASE=/home/jenkins/go/src/infini.sh
WORKDIR=$WORKBASE/$PNAME
DEST=/infini/Sync/Release/$PNAME/stable

#change branch
cd $WORKBASE/framework
git branch |grep -wq "ecloud-0.3.1" || (git switch -c ecloud-0.3.1 && git pull)
git branch |grep -wq "ecloud-0.3.1" && (git checkout ecloud-0.3.1 && git pull)

cd $WORKSPACE
git branch |grep -wq "ecloud" || (git switch -c ecloud && git pull)
git branch |grep -wq "ecloud" && (git checkout ecloud && git pull)

#build
make clean config build-linux

#copy-configs
cp -rf $WORKBASE/framework/LICENSE $WORKDIR/bin && cat $WORKBASE/framework/NOTICE $WORKDIR/NOTICE > $WORKDIR/bin/NOTICE

cd $WORKDIR/bin
for t in amd64 ; do
  tar zcf ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-linux-$t.tar.gz "${PNAME}-linux-$t" $PNAME.yml LICENSE NOTICE
done

#git reset
cd $WORKSPACE && git checkout master && git reset --hard
cd $WORKBASE/framework && git checkout master && git reset --hard
