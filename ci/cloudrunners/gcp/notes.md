
PROJECT_ID=$(gcloud config get project)
PROJECT_NUMBER=$(gcloud projects describe ${PROJECT_ID} --format="value(projectNumber)")

# Upload runner image to gcr
docker buildx build --push -t gcr.io/${PROJECT_ID}/gha-cloudrunners-gcp:latest .

# Create GCE disk image
IMAGE_TAG=$(date +%Y%m%d-%H%M%S)
go run ./tools/gha-imagebuilder/ --project ${PROJECT_ID} --machine-type=n1-standard-4 --disk-name gha-${IMAGE_TAG}-amd64

go run ./tools/gha-imagebuilder/ --project ${PROJECT_ID} --machine-type=n1-standard-4 --disk-name gha-${IMAGE_TAG}-arm64


# TODO: Can we use an image family somehow?
DISK_NAME=projects/${PROJECT_ID}/global/images/gha-${IMAGE_TAG}-amd64

# Patch the ARC config
kubectl patch autoscalingrunnersets.actions.github.com -n arc-runners arc-runner-set --type=merge --patch-file=/dev/stdin <<EOF
spec:
  template:
    spec:
      serviceAccountName: gha-cloudrunners-gcp
      containers:
      - image: gcr.io/${PROJECT_ID}/gha-cloudrunners-gcp:latest
        name: runner
        env:
        - name: RUNNER_IMAGE
          value: ${DISK_NAME}
        - name: RUNNER_MACHINE_TYPE
          value: n1-standard-4
EOF

# Create a serviceaccount to run the Pod (not the VM, the "trampoline" Pod)
kubectl create serviceaccount -n arc-runners gha-cloudrunners-gcp


# Grant permission on the serviceaccount to create VMs
NAMESPACE=arc-runners
KSA_NAME=gha-cloudrunners-gcp
MEMBER=principal://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${PROJECT_ID}.svc.id.goog/subject/ns/${NAMESPACE}/sa/${KSA_NAME}

gcloud projects add-iam-policy-binding projects/${PROJECT_ID} \
    --role=roles/compute.instanceAdmin.v1 \
    --member=${MEMBER} \
    --condition=None
