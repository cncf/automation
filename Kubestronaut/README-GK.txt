cat Kubestronaut.json | jq -r '.[] | select ((.GK=="1")) | .Email+" - "+.Country'
