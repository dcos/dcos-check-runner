pipeline {
    agent {
        node {
            label 'mesos'
        }
    }
    stages {
        stage('testing') {
            steps {
                sh 'make test'
            }
        }
    }
}
