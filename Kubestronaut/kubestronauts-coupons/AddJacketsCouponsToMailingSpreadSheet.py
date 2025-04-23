import csv
import json
import os
from collections import OrderedDict
import argparse
import shutil
import pygsheets
import pandas as pd
from dotenv import load_dotenv
import datetime


# In the same directory :
# - a file named Jackets-Coupons.csv should contains coupons provided by Pinnacle
# - a file name KubestronautToReceiveJackets.csv that contains the infos of Kubestronauts to send the coupon
parser = argparse.ArgumentParser(description='Add Kubestronaut info and Jacket Coupons to mailing spreadsheet')
#parser.add_argument('-ll','--lastLine', help='Last row number to be added from the tsv file', required=True)
#args = vars(parser.parse_args())
#lastLineToBeInserted = int(args['lastLine'])

load_dotenv("../.env")
KUBESTRONAUTS_MAILING_JACKET_COUPONS = os.getenv("KUBESTRONAUTS_MAILING_JACKET_COUPONS")
KUBESTRONAUTS_INFOS = os.getenv('KUBESTRONAUTS_INFOS')


# Let's open the GoogleSheet to write Kubestronaut info + coupons
#authorization
gc = pygsheets.authorize(service_file='kubestronauts-handling-service-file.json')
#open the google spreadsheet with coupons
sh = gc.open_by_key(KUBESTRONAUTS_MAILING_JACKET_COUPONS)
#select the first sheet
wks = sh[0]

#open the google spreadsheet with infos
sh_infos = gc.open_by_key(KUBESTRONAUTS_INFOS)
#select the first sheet
wks_infos = sh_infos[0]

today = datetime.datetime.now()
annotation="Individual-"+str(today.year)+str(today.month)+str(today.day)

# Read the csv using pandas
coupons_data = pd.read_csv("Jackets-Coupons.csv")

numberOfKubestronauts = 0

# Import CSV that needs to be treated
with open('KubestronautToReceiveJackets.csv') as csv_file:
    csv_reader = csv.reader(csv_file, delimiter=';')

    for row in csv_reader:
        print(f'\t{row[0]} - {row[2]}')
        email=row[2].strip()
        coupon=coupons_data.iloc[0]
        coupon_code = coupon['Code']
        numberOfKubestronauts += 1
        values_list=[row[0], email, coupon_code]
        wks.insert_rows(row=1, number=1, values=values_list)
        coupons_data = coupons_data.drop(coupons_data.index[:1])

        list_kubestronauts_cells=wks_infos.find(pattern=email, cols=(13,13), matchEntireCell=False)
        number_matching_cells = len(list_kubestronauts_cells)

        if (number_matching_cells==1):
            email_cell = list_kubestronauts_cells[0]
            wks_infos.update_value("S"+str(email_cell.row),annotation)
            print(email+" : OK")
        elif (number_matching_cells==0):
            print("Kubestronaut with email "+row[2]+" not found !!")
        else:
            print("Kubestronaut with email "+row[2]+" found multiple times !!")

coupons_data.to_csv("Jackets-Coupons.csv")


print("\n\n\nThe URL of the mail merger is \"https://docs.google.com/spreadsheets/d/"+KUBESTRONAUTS_MAILING_JACKET_COUPONS+"\"")
print("\n\n\nThe name of the email to use in the mail merger is \"Your Kubestronaut jacket !\"")
