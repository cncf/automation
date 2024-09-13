import csv
import json
import os
from collections import OrderedDict
import argparse
import shutil
import pygsheets
import pandas as pd

validity="31st December 2024"


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


# Let's open the GoogleSheet to write Kubestronaut info + coupons
#authorization
gc = pygsheets.authorize(service_file='kubestronauts-handling-service-file.json')
#open the google spreadsheet
sh = gc.open_by_key('1SeIntS9PeS07RIPJjM3UBvE3cOeEgnHEKc6l4rx4iSE')
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
                print(f'\t{row[1]}')

                coupon=coupons_data.head(5)
                print(numberOfKubestronauts*5)
                print(coupon.at[numberOfKubestronauts*5, 'name'])
                print(coupon.at[1+numberOfKubestronauts*5, 'name'])
                print(coupon.at[2+numberOfKubestronauts*5, 'name'])
                print(coupon.at[3+numberOfKubestronauts*5, 'name'])
                print(coupon.at[4+numberOfKubestronauts*5, 'name'])

                values_list=[row[1], row[12], validity, coupon.at[numberOfKubestronauts*5, 'name'], coupon.at[1+numberOfKubestronauts*5, 'name'], coupon.at[2+numberOfKubestronauts*5, 'name'], coupon.at[3+numberOfKubestronauts*5, 'name'], coupon.at[4+numberOfKubestronauts*5, 'name']]
                wks.insert_rows(row=1, number=1, values=values_list)
                coupons_data = coupons_data.drop(coupons_data.index[:5])
                numberOfKubestronauts += 1
                break
            else:
                lineCount += 1

coupons_data.to_csv("Coupons.csv")

