pipeline {
  agent any
  
  environment {
    AWS_ACCOUNT_ID = credentials('AWS_ACCOUNT_ID')
    AWS_DEFAULT_REGION = 'us-east-1'
    ECR_FRONTEND_REPO = "${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_DEFAULT_REGION}.amazonaws.com/frontend"
    ECR_BACKEND_REPO = "${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_DEFAULT_REGION}.amazonaws.com/backend"
    DOCKER_BUILDKIT = '1'
    COMPOSE_PROJECT_NAME = "pipeline-${BUILD_NUMBER}"

    DB_USER = credentials('DB_USER')
    DB_PASSWORD = credentials('DB_PASSWORD')
    DB_NAME = credentials('DB_NAME')
    API_URL = 'http://backend:8080'  
  }
  
  stages {
    stage('Clone Repository') {
      steps {
        cleanWs()
        git branch: 'main', url: 'YOUR_REPO_URL'
      }
    }
    
    stage('Start Dependencies') {
      steps {
        script {
          sh """
            docker-compose up -d postgres
            # Wait for PostgreSQL to be healthy
            docker-compose ps postgres | grep -q "database system is ready to accept connections" || {
              for i in `seq 1 30`; do
                if docker-compose ps postgres | grep -q "healthy"; then
                  break
                fi
                echo "Waiting for PostgreSQL to be healthy..."
                sleep 2
              done
            }
          """
        }
      }
    }
    
    stage('Run Migrations') {
      steps {
        script {
          sh 'docker-compose --profile migrate run --rm migration'
        }
      }
    }
    
    stage('AWS ECR Login') {
      steps {
        script {
          sh """
            aws ecr get-login-password --region ${AWS_DEFAULT_REGION} | docker login --username AWS --password-stdin ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_DEFAULT_REGION}.amazonaws.com
          """
        }
      }
    }
    
    stage('Build and Test Services') {
      steps {
        script {
          sh """
            docker-compose build frontend backend
            docker-compose up -d backend frontend
            
            # Wait for backend to be healthy
            docker-compose ps backend | grep -q "Server started on" || {
              for i in `seq 1 30`; do
                if docker-compose ps backend | grep -q "Server started on"; then
                  break
                fi
                echo "Waiting for backend to be healthy..."
                sleep 2
              done
            }
          """
        }
    }
    }
    
    stage('Tag Images for ECR') {
      steps {
        script {
          sh """
            docker tag \${COMPOSE_PROJECT_NAME}_frontend:latest ${ECR_FRONTEND_REPO}:${BUILD_NUMBER}
            docker tag \${COMPOSE_PROJECT_NAME}_frontend:latest ${ECR_FRONTEND_REPO}:latest
            docker tag \${COMPOSE_PROJECT_NAME}_backend:latest ${ECR_BACKEND_REPO}:${BUILD_NUMBER}
            docker tag \${COMPOSE_PROJECT_NAME}_backend:latest ${ECR_BACKEND_REPO}:latest
          """
        }
      }
    }
    
    stage('Push to ECR') {
      steps {
        script {
          sh """
            docker push ${ECR_FRONTEND_REPO}:${BUILD_NUMBER}
            docker push ${ECR_FRONTEND_REPO}:latest
            docker push ${ECR_BACKEND_REPO}:${BUILD_NUMBER}
            docker push ${ECR_BACKEND_REPO}:latest
          """
        }
      }
    }
    
    stage('Cleanup') {
      steps {
        script {
          sh """
            # Stop all services and remove volumes
            docker-compose down -v
            
            # Remove tagged images
            docker rmi ${ECR_FRONTEND_REPO}:${BUILD_NUMBER} ${ECR_FRONTEND_REPO}:latest || true
            docker rmi ${ECR_BACKEND_REPO}:${BUILD_NUMBER} ${ECR_BACKEND_REPO}:latest || true
            docker rmi \${COMPOSE_PROJECT_NAME}_frontend:latest \${COMPOSE_PROJECT_NAME}_backend:latest || true
          """
        }
      }
    }
  }
  
  post {
      always {
          script {
            sh 'docker-compose down -v || true'
          }
      }
  }
}