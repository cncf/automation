Usage: python AddCouponsToMailingSpreadSheet.py -fl 171 -ll 173

fl firstline
ll lastline (included)



Utilisation google sheet https://erikrood.com/Posts/py_gsheets.html

Reminder needs to authorize in the sheet the Service Account email

-----

For the United States
export COUNTRY="United States"

cat ../Kubestronaut.json | jq -r --arg COUNTRY "$COUNTRY" '.[] | select ((.Country==$COUNTRY) and (.JacketSent=="")) | .Name +" ; "+ .Size +" ; "+ .Email +" ; "+ .Address+" ; "+.JacketSent' > KubestronautToReceiveJackets.csv

python AddJacketsCouponsToMailingSpreadSheet.py


----- 


