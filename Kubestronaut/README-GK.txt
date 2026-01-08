cat Kubestronaut.json | jq -r '.[] | select ((.GK=="1")) | .Email+" - "+.Country'

cat Kubestronaut.json | jq -r '.[] | select ((.GK=="1") and (.GKBeanie=="")) | .Name+ " - " +.Email+" - "+.Address'
cat Kubestronaut.json | jq --arg COUNTRY "$COUNTRY" -r '.[] | select ((.GK=="1") and (.GKBeanie=="") and (.Country==$COUNTRY)) | .Name+ " - " +.Email+" - "+.Address'

