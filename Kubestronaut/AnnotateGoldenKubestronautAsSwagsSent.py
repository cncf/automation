import csv
import json
import os
from collections import OrderedDict
import argparse
import shutil
import pygsheets
import pandas as pd
from dotenv import load_dotenv

parser = argparse.ArgumentParser(description='Annotate the Kubestronaut sheet to reflect shipping associated to a Golden Kubestronauts email')

parser.add_argument('-a','--annotation', help='', required=True)
parser.add_argument('-e','--email', help='', required=True)
args = vars(parser.parse_args())

annotation = args['annotation']
email = args['email']

load_dotenv()
# Store credentials
KUBESTRONAUTS_INFOS = os.getenv('KUBESTRONAUTS_INFOS')

# Let's open the GoogleSheet to write Kubestronaut info + coupons
#authorization
gc = pygsheets.authorize(service_file='kubestronauts-handling-service-file.json')
#open the google spreadsheet
sh = gc.open_by_key(KUBESTRONAUTS_INFOS)
#select the first sheet
wks = sh[0]

list_kubestronauts_cells=wks.find(pattern=email, cols=(13,13), matchEntireCell=True)
number_matching_cells = len(list_kubestronauts_cells)

if (number_matching_cells==1):
        email_cell = list_kubestronauts_cells[0]
        wks.update_value("AA"+str(email_cell.row),annotation)
        wks.update_value("AB"+str(email_cell.row),annotation)
        print(email+" : OK")
elif (number_matching_cells==0):
        print("Kubestronaut with email "+email+" not found !!")
else:
        print("Kubestronaut with email "+email+" found multiple times !!")
