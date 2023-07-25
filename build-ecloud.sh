#!/bin/bash

#init
WORKBASE=/home/jenkins/go/src/infini.sh
WORKDIR=$WORKBASE/$PNAME

#change branch
cd $WORKBASE/framework
git branch |grep -wq ecloud || (git checkout -b ecloud ecloud-0.3.1)
if [ "$(git symbolic-ref --short HEAD)"=="master" ]; then
  git checkout ecloud
  echo "framework checkout ecloud"
fi

cd $WORKBASE/framework-vendor
git branch |grep -wq ecloud || (git checkout -b ecloud ecloud-0.3.1)
if [ "$(git symbolic-ref --short HEAD)"=="master" ]; then
  git checkout ecloud
  echo "framework-vendor checkout ecloud"
fi

cd $WORKDIR
if [ "$(git symbolic-ref --short HEAD)"=="master" ]; then
  git checkout ecloud && git pull
  echo "agent checkout ecloud"
fi

#build
make clean config build-linux
make config build-darwin

#copy-configs
cp -rf $WORKBASE/framework/LICENSE $WORKDIR/bin && cat $WORKBASE/framework/NOTICE $WORKDIR/NOTICE > $WORKDIR/bin/NOTICE

cd $WORKDIR/bin
for t in amd64 ; do
  tar zcf ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-linux-$t.tar.gz "${PNAME}-linux-$t" $PNAME.yml LICENSE NOTICE
done

for t in mac-amd64; do
  [ -f ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-$t.zip ] && cp -rf ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-$t.zip $DEST
done

#git reset
cd $WORKDIR && git reset --hard && git checkout master && git reset --hard
cd $WORKBASE/framework && git reset --hard && git checkout master && git reset --hard
cd $WORKBASE/framework-vendor && git reset --hard && git checkout master && git reset --hard
