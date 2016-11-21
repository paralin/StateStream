node {
  stage ("node v6") {
    sh 'init-node-ci 6'
  }

  stage ("scm") {
    checkout scm
    sh 'init-jenkins-node-scripts'
  }

  env.CACHE_CONTEXT='state-stream'
  wrap([$class: 'AnsiColorBuildWrapper', 'colorMapName': 'XTerm']) {
    stage ("cache-download") {
      sh '''
        #!/bin/bash
        source ./jenkins_scripts/jenkins_env.bash
        ./jenkins_scripts/init_cache.bash
      '''
    }

    stage ("install") {
      sh '''
        #!/bin/bash
        source ./jenkins_scripts/jenkins_env.bash
        enable-npm-proxy
        npm install -g yarn
        yarn install
        ./scripts/jenkins_setup_deps.bash
      '''
    }

    stage ("cache-upload") {
      sh '''
        #!/bin/bash
        source ./jenkins_scripts/jenkins_env.bash
        ./jenkins_scripts/finalize_cache.bash
      '''
    }

    stage ("test") {
      sh '''
        #!/bin/bash
        source ./jenkins_scripts/jenkins_env.bash
        npm run ci
      '''
    }

    stage ("gotest") {
      sh '''
        #!/bin/bash
        source ./jenkins_scripts/jenkins_env.bash
        go test .
      '''
    }

    stage ("release") {
      sh '''
        #!/bin/bash
        source ./jenkins_scripts/jenkins_env.bash
        ./jenkins_scripts/jenkins_release.bash
      '''
    }
  }
}
