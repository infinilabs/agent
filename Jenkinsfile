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
                    sh 'cd /home/jenkins/go/src/infini.sh/agent && git stash && git pull origin master && make clean config build-linux build-arm'
                    sh label: 'package-linux-amd64', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-amd64.tar.gz agent-linux-amd64 agent.yml ../sample-configs'
                    sh label: 'package-linux-386', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-386.tar.gz agent-linux-386 agent.yml ../sample-configs'
                    sh label: 'package-linux-mips', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-mips.tar.gz agent-linux-mips agent.yml ../sample-configs'
                    sh label: 'package-linux-mipsle', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-mipsle.tar.gz agent-linux-mipsle agent.yml ../sample-configs'
                    sh label: 'package-linux-mips64', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-mips64.tar.gz agent-linux-mips64 agent.yml ../sample-configs'
                    sh label: 'package-linux-mips64le', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-mips64le.tar.gz agent-linux-mips64le agent.yml ../sample-configs'
                    sh label: 'package-linux-arm5', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-arm5.tar.gz agent-linux-armv5 agent.yml ../sample-configs'
                    sh label: 'package-linux-arm6', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-arm6.tar.gz agent-linux-armv6 agent.yml ../sample-configs'
                    sh label: 'package-linux-arm7', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-arm7.tar.gz agent-linux-armv7 agent.yml ../sample-configs'
                    sh label: 'package-linux-arm64', script: 'cd /home/jenkins/go/src/infini.sh/agent/bin && tar cfz ${WORKSPACE}/agent-$VERSION-$BUILD_NUMBER-linux-arm64.tar.gz agent-linux-arm64 agent.yml ../sample-configs'
                    archiveArtifacts artifacts: 'agent-$VERSION-$BUILD_NUMBER-*.tar.gz', fingerprint: true, followSymlinks: true, onlyIfSuccessful: false
                }
            }
        }

        stage('Build Docker Images') {

                    agent {
                        label 'linux'
                    }

                    steps {
                        catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE'){
                            sh label: 'docker-build', script: 'cd /home/jenkins/go/src/infini.sh/ && docker build -t infini-agent  -f agent/docker/Dockerfile .'
                            sh label: 'docker-tagging', script: 'docker tag infini-agent medcl/infini-agent:latest && docker tag infini-agent medcl/infini-agent:$VERSION-$BUILD_NUMBER'
                            sh label: 'docker-push', script: 'docker push medcl/infini-agent:latest && docker push medcl/infini-agent:$VERSION-$BUILD_NUMBER'
                        }
                    }
                }

    }
}
