import csv
import json
import os
from collections import OrderedDict
import argparse
import shutil
import pygsheets
import pandas as pd
from dotenv import load_dotenv

validity="31st December 2025"


# In the same directory :
# - a file named Kubestronaut.tsv should contains the export of the
# Kubestronauts responses in tsv
# - a file named Coupons.csv should contains coupons to be used
parser = argparse.ArgumentParser(description='Add Kubestronaut info and Coupons to mailing spreadsheet')
parser.add_argument('-fl','--firstLine', help='First row number to be added from the tsv file', required=True)
parser.add_argument('-ll','--lastLine', help='Last row number to be added from the tsv file', required=True)
args = vars(parser.parse_args())

firstLineToBeInserted = int(args['firstLine'])
lastLineToBeInserted = int(args['lastLine'])

load_dotenv("../.env")
KUBESTRONAUTS_MAILING_COUPONS = os.getenv('KUBESTRONAUTS_MAILING_COUPONS')

# Let's open the GoogleSheet to write Kubestronaut info + coupons
#authorization
gc = pygsheets.authorize(service_file='kubestronauts-handling-service-file.json')
#open the google spreadsheet
sh = gc.open_by_key(KUBESTRONAUTS_MAILING_COUPONS)
#select the first sheet
wks = sh[0]


# Read the csv using pandas
coupons_data = pd.read_csv("Coupons.csv")


numberOfKubestronauts = 0

for lineToBeInserted in range(firstLineToBeInserted, lastLineToBeInserted+1, 1):
    # Import CSV that needs to be treated
    with open('Kubestronaut.tsv') as csv_file:
        lineCount = 1
        csv_reader = csv.reader(csv_file, delimiter='\t')

        for row in csv_reader:
            if lineCount == lineToBeInserted:
                if row[1]:
                    print(f'\t{row[1]}')

                    coupon=coupons_data.head(5)
                    print(numberOfKubestronauts*5)
                    print(coupon.at[numberOfKubestronauts*5, 'name'])
                    print(coupon.at[1+numberOfKubestronauts*5, 'name'])
                    print(coupon.at[2+numberOfKubestronauts*5, 'name'])
                    print(coupon.at[3+numberOfKubestronauts*5, 'name'])
                    print(coupon.at[4+numberOfKubestronauts*5, 'name'])

                    values_list=[row[1], row[12], validity, coupon.at[numberOfKubestronauts*5, 'name'], coupon.at[1+numberOfKubestronauts*5, 'name'], coupon.at[2+numberOfKubestronauts*5, 'name'], coupon.at[3+numberOfKubestronauts*5, 'name'], coupon.at[4+numberOfKubestronauts*5, 'name']]
# ,Unnamed: 0.26,Unnamed: 0.25,Unnamed: 0.24,Unnamed: 0.23,Unnamed: 0.22,Unnamed: 0.21,Unnamed: 0.20,Unnamed: 0.19,Unnamed: 0.18,Unnamed: 0.17,Unnamed: 0.16,Unnamed: 0.15,Unnamed: 0.14,Unnamed: 0.13,Unnamed: 0.12,Unnamed: 0.11,Unnamed: 0.10,Unnamed: 0.9,Unnamed: 0.8,Unnamed: 0.7,Unnamed: 0.6,Unnamed: 0.5,Unnamed: 0.4,Unnamed: 0.3,Unnamed: 0.2,Unnamed: 0.1,Unnamed: 0,S.No,id,name,coupon_group_name,active,description,discount,type,max_redemptions,valid_from,expires_at,creator,ti_products,ti_content_types,ti_tags, Unnamed: 0.2, Unnamed: 0.3, Unnamed: 0.4, Unnamed: 0.5, Unnamed: 0.6, Unnamed: 0.7, Unnamed: 0.8, Unnamed: 0.9, Unnamed: 1.0
                    wks.insert_rows(row=1, number=1, values=values_list)
                    coupons_data = coupons_data.drop(coupons_data.index[:5])
                    numberOfKubestronauts += 1
                    break
                else:
                    print("File has an empty line "+str(lineToBeInserted))
                    break
            else:
                lineCount += 1

coupons_data.to_csv("Coupons.csv")


print("\n\n\nThe URL of the merger is \"https://docs.google.com/spreadsheets/d/"+KUBESTRONAUTS_MAILING_COUPONS+"\"")
print("\n\n\nThe name of the email to use in the mail merger is Coupons as a Kubestronaut")
