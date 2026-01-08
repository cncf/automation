Usage: python AddCouponsToMailingSpreadSheet.py -fl 171 -ll 173

fl firstline
ll lastline (included)



Utilisation google sheet https://erikrood.com/Posts/py_gsheets.html

Reminder needs to authorize in the sheet the Service Account email

-----

For the United States
For Kubestronauts :


#export COUNTRY="United States"
#cat ../Kubestronaut.json | jq -r --arg COUNTRY "$COUNTRY" '.[] | select ((.Country==$COUNTRY) and (.JacketSent=="")) | .Name +" ; "+ .Size +" ; "+ .Email +" ; "+ .Address+" ; "+.JacketSent' > KubestronautToReceiveJackets.csv
./jackets-to-send.sh
python AddJacketsCouponsToMailingSpreadSheet.py



----- 


For the United States
For Golden Kubestronauts :


export COUNTRY="United States"

cat ../Kubestronaut.json | jq -r --arg COUNTRY "$COUNTRY" '.[] | select ((.Country==$COUNTRY) and (.GKBeanie=="") and (.GK=="1")) | .Name +" ; "+ .Email' > GoldenKubestronautToReceiveSwags.csv

python AddGoldenKubestronautsSwagsCouponsToMailingSpreadSheet.py



