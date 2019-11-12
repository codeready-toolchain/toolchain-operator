.PHONY: docker-image
## Build the docker image locally that can be deployed (only contains bare operator)
docker-image: build
	$(Q)docker build -f build/Dockerfile -t ${IMAGE_NAME} .
