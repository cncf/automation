TOPDIR=$(PWD)
WHOAMI=$(shell whoami)

images: image-24 image-22

image-24:
	docker buildx build --platform=linux/amd64 -t ${WHOAMI}/gha-runner:24.04 -f Dockerfile.24.04 .

image-22:
	docker buildx build --platform=linux/amd64 -t ${WHOAMI}/gha-runner:22.04 -f Dockerfile.22.04 .

images-push: images
	docker push ${WHOAMI}/gha-runner

images-run: images
	docker run -ti --rm ${WHOAMI}/gha-runner
