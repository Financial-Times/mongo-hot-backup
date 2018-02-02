DOCKER_REGISTRY=registry.uw.systems
DOCKER_REGISTRY_NS=system
DOCKER_IMAGE=mongo-hot-backup
DOCKER_TAG=$(DOCKER_REGISTRY)/$(DOCKER_REGISTRY_NS)/$(DOCKER_IMAGE)

docker-image:
	docker build -t $(DOCKER_TAG) .

docker-push:
	docker push $(DOCKER_TAG)