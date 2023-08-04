#!/bin/bash

#!/bin/bash

#NOTICE: must change framework Makefile remove git pulll from master, we just need code from ecloud branch

#init
WORKBASE=/home/jenkins/go/src/infini.sh
WORKDIR=$WORKBASE/$PNAME

#change branch 
cd $WORKBASE/framework
git branch |grep -wq ecloud && git branch -D ecloud
if [ "$(git symbolic-ref --short HEAD)" == "master" ]; then
  git pull && git checkout ecloud
  echo "framework checkout ecloud"
fi

cd $WORKBASE/vendor
git branch |grep -wq ecloud && git branch -D ecloud
if [ "$(git symbolic-ref --short HEAD)" == "master" ]; then
  git pull && git checkout ecloud
  echo "vendor checkout ecloud"
fi

cd $WORKDIR
git branch |grep -wq ecloud && git branch -D ecloud
if [ "$(git symbolic-ref --short HEAD)" == "master" ]; then
  git pull && git checkout ecloud
  echo "agent checkout ecloud"
fi

#build
make clean config build-linux-amd64
make config build-darwin

#copy-configs
cp -rf $WORKBASE/framework/LICENSE $WORKDIR/bin && cat $WORKBASE/framework/NOTICE $WORKDIR/NOTICE > $WORKDIR/bin/NOTICE
if [ ! -f $WORKDIR/bin/$PNAME.yml ]; then
  cp -rf $PNAME.yml $WORKDIR/bin
fi

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
