pipeline {

    agent none

    environment { 
        CI = 'true'
    }
    stages {
        
        stage('Build Linux Packages') {

            agent {
                label 'linux'
            }

            steps {
                catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE'){
                    sh 'cd /home/jenkins/go/src/infini.sh/agent && git stash && git pull origin master && make clean config build-linux'
                    sh 'cd /home/jenkins/go/src/infini.sh/agent && git stash && git pull origin master && make config build-arm'
                    sh 'cd /home/jenkins/go/src/infini.sh/agent && git stash && git pull origin master && make config build-darwin'
                    sh 'cd /home/jenkins/go/src/infini.sh/agent && git stash && git pull origin master && make config build-win'

                   sh label: 'copy-configs', script: 'cd /home/jenkins/go/src/infini.sh/agent && cp -R  bin && cp ../framework/LICENSE bin && cat ../framework/NOTICE NOTICE > bin/NOTICE'

                   sh label: 'package-linux-amd64', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-amd64.tar.gz agent-linux-amd64 agent.yml LICENSE NOTICE '
                   sh label: 'package-linux-386', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-386.tar.gz agent-linux-386 agent.yml LICENSE NOTICE '
                   sh label: 'package-linux-mips', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-mips.tar.gz agent-linux-mips agent.yml LICENSE NOTICE '
                   sh label: 'package-linux-mipsle', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-mipsle.tar.gz agent-linux-mipsle agent.yml LICENSE NOTICE '
                   sh label: 'package-linux-mips64', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-mips64.tar.gz agent-linux-mips64 agent.yml LICENSE NOTICE '
                   sh label: 'package-linux-mips64le', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-mips64le.tar.gz agent-linux-mips64le agent.yml LICENSE NOTICE '
                   sh label: 'package-linux-arm5', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-arm5.tar.gz agent-linux-armv5 agent.yml LICENSE NOTICE '
                   sh label: 'package-linux-arm6', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-arm6.tar.gz agent-linux-armv6 agent.yml LICENSE NOTICE '
                   sh label: 'package-linux-arm7', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-arm7.tar.gz agent-linux-armv7 agent.yml LICENSE NOTICE '
                   sh label: 'package-linux-arm64', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-arm64.tar.gz agent-linux-arm64 agent.yml LICENSE NOTICE '

                    sh label: 'package-mac-amd64', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && zip -r ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-mac-amd64.zip agent-mac-amd64 agent.yml LICENSE NOTICE '
                    sh label: 'package-mac-arm64', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && zip -r ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-mac-arm64.zip agent-mac-arm64 agent.yml LICENSE NOTICE '

                    sh label: 'package-win-amd64', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && zip -r ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-windows-amd64.zip agent-windows-amd64.exe agent.yml LICENSE NOTICE '
                    sh label: 'package-win-386', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && zip -r ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-windows-386.zip agent-windows-386.exe agent.yml LICENSE NOTICE '


                    fingerprint 'agent-$VERSION-$BUILD_NUMBER-*'
                    archiveArtifacts artifacts: 'agent-$VERSION-$BUILD_NUMBER-*', fingerprint: true, followSymlinks: true, onlyIfSuccessful: true
                }
            }
        }

    }
}
