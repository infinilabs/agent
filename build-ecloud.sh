#!/bin/bash

#init
WORKBASE=/home/jenkins/go/src/infini.sh
WORKDIR=$WORKBASE/$PNAME

#change branch
cd $WORKBASE/framework
git branch |grep -wq ecloud && git branch -D ecloud
if [ "$(git symbolic-ref --short HEAD)"=="master" ]; then
  git pull && git checkout ecloud
  echo "framework checkout ecloud"
fi

cd $WORKBASE/vendor
git branch |grep -wq ecloud && git branch -D ecloud
if [ "$(git symbolic-ref --short HEAD)"=="master" ]; then
  git pull && git checkout ecloud
  echo "vendor checkout ecloud"
fi

cd $WORKDIR
git branch |grep -wq ecloud && git branch -D ecloud
if [ "$(git symbolic-ref --short HEAD)"=="master" ]; then
  git pull && git checkout ecloud
  echo "agent checkout ecloud"
fi

#build
#make clean config build-linux
#make config build-darwin

sed -i "s/build-linux-amd64: config/build-linux-amd64:/" $WORKBASE/framework/Makefile 
sed -i "s/build-darwin: config/build-darwin:/" $WORKBASE/framework/Makefile 

make clean update-vfs update-generated-file update-plugins build-linux-amd64 build-darwin

sed -i "s/build-linux-amd64:/build-linux-amd64: config/" $WORKBASE/framework/Makefile
sed -i "s/build-darwin:/build-darwin: config/" $WORKBASE/framework/Makefile

#copy-configs
cp -rf $WORKBASE/framework/LICENSE $WORKDIR/bin && cat $WORKBASE/framework/NOTICE $WORKDIR/NOTICE > $WORKDIR/bin/NOTICE
cp -rf $PNAME.yml $WORKDIR/bin

cd $WORKDIR/bin
for t in amd64 ; do
  tar zcf ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-linux-$t.tar.gz "${PNAME}-linux-$t" $PNAME.yml LICENSE NOTICE
done

for t in mac-amd64; do
  zip -qr ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-$t.zip $PNAME-$t $PNAME.yml LICENSE NOTICE config
done

#git reset
cd $WORKDIR && git reset --hard && git checkout master && git reset --hard
cd $WORKBASE/framework && git reset --hard && git checkout master && git reset --hard
cd $WORKBASE/vendor && git reset --hard && git checkout master && git reset --hard
