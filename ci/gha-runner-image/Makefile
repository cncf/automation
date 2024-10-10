TOPDIR=$(PWD)
WHOAMI=$(shell whoami)

image:
	docker buildx build --platform=linux/amd64,linux/arm64 -t ${WHOAMI}/gha-runner -f Dockerfile .
	docker buildx build --platform=linux/arm64 -t ${WHOAMI}/gha-runner:openeuler -f Dockerfile.openeuler .

image-push: image
	docker push ${WHOAMI}/gha-runner

image-run: image
	docker run -ti --rm ${WHOAMI}/gha-runner
