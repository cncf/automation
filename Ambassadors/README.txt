# Projects list
wget https://landscape.cncf.io/api/projects/all.json
jq '.[] | select(.accepted_at) | .name' all.json | cut -d'"' -f2 > CNCF-Project-list.txt
