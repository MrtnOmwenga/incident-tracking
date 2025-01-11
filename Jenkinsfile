pipeline {
    agent any
    environment {
        AWS_REGION = 'us-east-1' 
        AWS_CREDENTIALS = credentials('aws-access-key')
        ECR_FRONTEND = "000000000000.dkr.ecr.${AWS_REGION}.amazonaws.com/incident-tracking/frontend"
        ECR_BACKEND = "000000000000.dkr.ecr.${AWS_REGION}.amazonaws.com/incident-tracking/backend"
        DOCKER_COMPOSE_FILE = 'docker-compose.yml'
        WORKSPACE_DIR = "${env.WORKSPACE}"
    }
    stages {
        stage('Checkout Code') {
            steps {
                checkout scm
            }
        }
        stage('Login to ECR') {
            steps {
                script {
                    withCredentials([[$class: 'AmazonWebServicesCredentialsBinding', credentialsId: 'aws-access-key']]) {
                        sh '''#!/bin/bash
                        export AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID
                        export AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY
                        aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $ECR_FRONTEND
                        aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $ECR_BACKEND
                        '''
                    }
                }
            }
        }
        stage('Run Migrations') {
            steps {
                script {
                    sh '''
                    cd $WORKSPACE_DIR
                    ls -l ./backend/migrations
                    docker-compose up migration
                    '''
                }
            }
        }
        stage('Build Docker Images') {
            steps {
                script {
                    sh 'docker-compose build'
                }
            }
        }
        stage('Tag Docker Images') {
            steps {
                script {
                    def frontendImageTag = "${ECR_FRONTEND}:latest"
                    def backendImageTag = "${ECR_BACKEND}:latest"
                    sh "docker tag workspace-frontend-1 ${frontendImageTag}"
                    sh "docker tag workspace-backend-1 ${backendImageTag}"
                }
            }
        }
        stage('Push Docker Images') {
            steps {
                script {
                    def frontendImageTag = "${ECR_FRONTEND}:latest"
                    def backendImageTag = "${ECR_BACKEND}:latest"
                    sh "docker push ${frontendImageTag}"
                    sh "docker push ${backendImageTag}"
                }
            }
        }
    }
    post {
        success {
            echo 'Docker images successfully pushed to ECR.'
        }
        failure {
            echo 'Deployment failed. Check the logs.'
        }
    }
}
