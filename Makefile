.PHONY: help build run test docker-build docker-run docker-compose-up docker-compose-down deploy clean

# Переменные
BINARY_NAME=stresspulse
IMAGE_NAME=stresspulse
IMAGE_TAG=latest
NAMESPACE=default

# Помощь
help: ## Показать справку
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Сборка
build: ## Собрать Go приложение
	@echo "Сборка Go приложения..."
	go build -o $(BINARY_NAME) .

run: ## Запустить приложение локально
	@echo "Запуск приложения..."
	./$(BINARY_NAME) -cpu 50 -drift 20 -pattern sine -metrics

test: ## Запустить тесты
	@echo "Запуск тестов..."
	go test -v ./...

# Docker
docker-build: ## Собрать Docker образ
	@echo "Сборка Docker образа..."
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .

docker-run: ## Запустить Docker контейнер
	@echo "Запуск Docker контейнера..."
	docker run --rm -p 9090:9090 $(IMAGE_NAME):$(IMAGE_TAG)

# Docker Compose
docker-compose-up: ## Запустить окружение с помощью Docker Compose
	@echo "Запуск Docker Compose..."
	docker-compose up -d

docker-compose-down: ## Остановить Docker Compose
	@echo "Остановка Docker Compose..."
	docker-compose down

docker-compose-logs: ## Просмотр логов Docker Compose
	docker-compose logs -f

# Kubernetes
deploy: ## Развернуть в Kubernetes
	@echo "Развертывание в Kubernetes..."
	./deploy.sh $(IMAGE_TAG) $(NAMESPACE)

k8s-status: ## Показать статус в Kubernetes
	@echo "Статус в Kubernetes:"
	kubectl get pods -n $(NAMESPACE) -l app=stresspulse
	kubectl get svc -n $(NAMESPACE) -l app=stresspulse

k8s-logs: ## Просмотр логов в Kubernetes
	kubectl logs -f -n $(NAMESPACE) -l app=stresspulse

k8s-port-forward: ## Проброс портов для доступа к метрикам
	kubectl port-forward -n $(NAMESPACE) svc/stresspulse 9090:9090

k8s-delete: ## Удалить из Kubernetes
	helm uninstall stresspulse -n $(NAMESPACE)

# Очистка
clean: ## Очистить сборочные артефакты
	@echo "Очистка..."
	rm -f $(BINARY_NAME)
	docker system prune -f

all: clean build docker-build deploy ## Полная сборка и развертывание 