#Insert Kubestronaut in the people repo
# Start line 739, last line (inlucded 771)
python CNCFInsertKubestronautInPeople_json.py -fl 739 -ll 771


# Annotation when doing a grouped shipping
export COUNTRY="XXX"
cat Kubestronaut.json | jq -r --arg COUNTRY "$COUNTRY" '.[] | select ((.Country==$COUNTRY) and (.JacketSent=="")) | .Email'  > Kubestronauts_ToBe_Annotated.txt
for i in $( cat Kubestronauts_ToBe_Annotated.txt ); do python AnnotateKubestronautAsJacketSent.py -a "Grouped-$COUNTRY-25" -e $i; done


#### Helpers
# List every Kubestronaut from a country
export COUNTRY="Japan"
cat Kubestronaut.json | jq --arg COUNTRY "$COUNTRY" '.[] | select (.Country==$COUNTRY) | .Name +" - "+ .Email'

# List every Kubestronaut from a country who have not their jacket sent
cat Kubestronaut.json | jq --arg COUNTRY "$COUNTRY" '.[] | select ((.Country==$COUNTRY) and (.JacketSent=="")) | .Name +" - "+ .Email


# Find out who got his jacket delivered at "OSS EU 24"
cat Kubestronaut.json | jq '.[] | select (.JacketSent=="OSS EU 24") | .Name +" - "+ .Email'

# Repartition of sizes since the beginning of the program
# cat Kubestronaut.json | jq -r '.[] | .Size' | sort | uniq -c

# Repartition of sizes which are pending shipment
# cat Kubestronaut.json | jq -r '.[] | select ((.JacketSent=="")) | .Size' | sort | uniq -c

# Locations since the beginning of the program
# cat Kubestronaut.json | jq -r '.[] | .Country' | sort | uniq -c

# Locations which are pending shipment
# cat Kubestronaut.json | jq -r '.[] | select ((.JacketSent=="")) | .Country' | sort | uniq -c

