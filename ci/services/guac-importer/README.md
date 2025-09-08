syft scan "dir:/var/home/mfahlandt/GolandProjects/automation/ci/services/guac-importer/local-dev/tmp/guac_ingest_ws/run.BwMFaP/source_repo" -o cyclonedx-json=1.5=test.json
source .env && ./ingest-github-data.sh 

