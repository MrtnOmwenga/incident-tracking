pipeline {
    agent any
    triggers {
        githubPush()
        // pollSCM('* * * * *')  // For general Git repositories
    }
    environment {
        AWS_REGION = 'us-east-1'
        AWS_CREDENTIALS = credentials('aws-access-key')
        ECR_FRONTEND = "000000000000.dkr.ecr.${AWS_REGION}.amazonaws.com/incident-tracking/frontend"
        ECR_BACKEND = "000000000000.dkr.ecr.${AWS_REGION}.amazonaws.com/incident-tracking/backend"

        DB_USER = credentials('db-user')
        DB_PASSWORD = credentials('db-password')
        DB_PORT = credentials('db-port')
        DB_NAME = credentials('db-name')
        API_URL = credentials('api-url')

        DOCKER_COMPOSE_FILE = 'docker-compose.yml'
        WORKSPACE_DIR = "${env.WORKSPACE}"
    }
    stages {
        stage('Checkout Code') {
            steps {
                checkout scm
            }
        }
        stage('Setup Environment') {
            steps {
                script {
                    sh '''
                        docker-compose down

                        echo "DB_USER=${DB_USER}" > .env
                        echo "DB_PASSWORD=${DB_PASSWORD}" >> .env
                        echo "DB_PORT=${DB_PORT}" >> .env
                        echo "DB_NAME=${DB_NAME}" >> .env
                        echo "API_URL=${API_URL}" >> .env
                        echo "WORKSPACE_DIR=${WORKSPACE_DIR}" >> .env
                    '''
                }
            }
        }
        stage('Login to ECR') {
            steps {
                script {
                    withCredentials([[$class: 'AmazonWebServicesCredentialsBinding', credentialsId: 'aws-access-key']]) {
                        sh '''#!/bin/bash
                        export AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID
                        export AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY
                        aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin $ECR_FRONTEND
                        aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin $ECR_BACKEND
                        '''
                    }
                }
            }
        }
        stage('Run Migrations') {
            steps {
                script {
                    sh '''
                        ls -la ${WORKSPACE_DIR}/backend/migrations

                        docker-compose run --rm --entrypoint sh migration -c "ls -la /migrations"

                        /usr/local/bin/docker-compose --profile migrate up migration --force-recreate
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
                    sh "docker tag workspace-frontend ${frontendImageTag}"
                    sh "docker tag workspace-backend ${backendImageTag}"
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