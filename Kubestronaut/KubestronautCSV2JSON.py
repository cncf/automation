import csv
import json

#Variables

# Decide the two file paths according to your 
# computer system
kubestronautCSV = 'Kubestronaut.tsv'
jsonFile = 'Kubestronaut.json'


def convertToJson(csvFile, jsonFile):
    lineNumber = 0
    fields=("Timestamp", "Name", "Description", "Company", "Pronoun", "Location", "LinkedIn", "Twitter", "Github", "Wechat", "Website", "Youtube", "Email", "SlackID", "Image", "Size", "Address", "JacketSent", "Acked", "MailingList", "SlackOK", "InsertedPeople", "SentCoupons", "Country", "2024Events", "2025Events")
    # create a dictionary
    data = {}
    # Open a csv reader called DictReader
    with open(csvFile, encoding='utf-8') as csvf:
        csvReader = csv.DictReader(csvf, fieldnames=fields, delimiter="\t")
        
        # Convert each row into a dictionary 
        # and add it to data
        for rows in csvReader:
            print(lineNumber)
            lineNumber=lineNumber+1
            if lineNumber == 1:
                continue
            if rows:
                # Assuming a column named 'No' to
                # be the primary key
                key = lineNumber
                data[key] = rows

    # Open a json writer, and use the json.dumps() 
    # function to dump data
    with open(jsonFile, 'w', encoding='utf-8') as jsonf:
        jsonf.write(json.dumps(data, indent=4))

# Call the make_json function
convertToJson(kubestronautCSV, jsonFile)
