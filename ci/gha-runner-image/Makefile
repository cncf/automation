TOPDIR=$(PWD)
WHOAMI=$(shell whoami)

images: image-noble image-jammy

image-noble:
	docker buildx build --build-arg BASE_IMAGE=noble --platform=linux/amd64 -t ${WHOAMI}/gha-runner:noble -f Dockerfile .

image-jammy:
	docker buildx build --build-arg BASE_IMAGE=jammy --platform=linux/amd64 -t ${WHOAMI}/gha-runner:jammy -f Dockerfile .

images-push: images
	docker push ${WHOAMI}/gha-runner

images-run: images
	docker run -ti --rm ${WHOAMI}/gha-runner:noble
