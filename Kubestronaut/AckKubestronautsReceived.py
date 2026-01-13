import os
import argparse
import pygsheets
from dotenv import load_dotenv


parser = argparse.ArgumentParser(description='Ack the reception of a Kubestronaut in the sheet')

parser.add_argument('-e','--email', help='', required=True)
args = vars(parser.parse_args())

email = args['email']

load_dotenv()
# Store credentials
KUBESTRONAUT_RECEIVERS = os.getenv('KUBESTRONAUT_RECEIVERS')


# Let's open the GoogleSheet to write Kubestronaut info + coupons
#authorization
gc = pygsheets.authorize(service_file='kubestronauts-handling-service-file.json')
#open the google spreadsheet
sh = gc.open_by_key(KUBESTRONAUT_RECEIVERS)
#select the first sheet
wks = sh[0]

NON_acked_Kubestronauts=[]

list_kubestronauts_cells=wks.find(pattern=email, cols=(2,2), matchEntireCell=False)
number_matching_cells = len(list_kubestronauts_cells)

if (number_matching_cells==1):
    email_cell = list_kubestronauts_cells[0]
    wks.update_value("G"+str(email_cell.row),"")
    cell_f2 = wks.cell('F2')
    bg_color_f2 = cell_f2.color

    cell=wks.cell("F"+str(email_cell.row))
    cell.color = bg_color_f2
    print("F"+str(email_cell.row))
    print("Kubestronaut with email "+email+" : ACKed")
elif (number_matching_cells==0):
    print("Kubestronaut with email "+email+" not found !!")
    NON_acked_Kubestronauts.append(email)
else:
    print("Kubestronaut with email "+email+" found multiple times !!")
    NON_acked_Kubestronauts.append(email)

if NON_acked_Kubestronauts:
    print("List of Kubestroauts that were NOT ACKED:")
    for email_address in NON_acked_Kubestronauts:
        print("\t"+email_address)
