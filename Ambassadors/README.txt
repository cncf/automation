# Projects list
wget https://landscape.cncf.io/api/projects/all.json
jq '.[] | select(.accepted_at) | .name' all.json | cut -d'"' -f2 > CNCF-Project-list.txt


# Get the secrets from https://console.cloud.google.com/auth/clients/, store it under credentials.json and run the script
python3  CNCFInsertAmbassadorInPeople_json.py -fl 2 -ll 45
# You will be prompted to login
