import pygsheets
import os
from dotenv import load_dotenv
import re

# Authenticate with your Google Sheets API credentials
gc = pygsheets.authorize(service_file='kubestronauts-handling-service-file.json')

# Load environment variables
load_dotenv()
KUBESTRONAUT_RECEIVERS = os.getenv('KUBESTRONAUT_RECEIVERS')
KUBESTRONAUTS_WEEKLY_TEMP = os.getenv('KUBESTRONAUTS_WEEKLY_TEMP')
KUBESTRONAUTS_WELCOME = os.getenv('KUBESTRONAUTS_WELCOME')

# Open the sheets
receivers_sheet = gc.open_by_key(KUBESTRONAUT_RECEIVERS)
weekly_temp_sheet = gc.open_by_key(KUBESTRONAUTS_WEEKLY_TEMP)
welcome_sheet = gc.open_by_key(KUBESTRONAUTS_WELCOME)

# Select the first worksheet in both spreadsheets
weekly_temp_worksheet = weekly_temp_sheet.sheet1
receivers_worksheet = receivers_sheet.sheet1
welcome_worksheet = welcome_sheet.sheet1

# Get emails from column C in the weekly temp sheet
emails_to_check = weekly_temp_worksheet.get_col(3, include_tailing_empty=False)

# Get first names (column A) and last names (column B) from the weekly temp sheet
first_names = weekly_temp_worksheet.get_col(1, include_tailing_empty=False)
last_names = weekly_temp_worksheet.get_col(2, include_tailing_empty=False)

# Get emails from column B in the receivers sheet
existing_emails = receivers_worksheet.get_col(2, include_tailing_empty=False)

# Iterate through emails to check
for idx, email in enumerate(emails_to_check):
    print(email)
    existing=False

    email_indiv = re.findall(r'[\w.+-]+@[\w-]+\.[\w.-]+', email)

    for email_sep in email_indiv:
#        if email in existing_emails:
        if any(email_sep in s for s in existing_emails):
            existing=True

    if not existing:
        print(f"Adding Kubestronaut: {email}")
        # Find the next row index after the last value in column B
        next_row = len(existing_emails) + 1

        # Insert a blank row before the next row
        receivers_worksheet.insert_rows(next_row - 1, number=1)

        # Add the email to column B in the newly inserted row
        receivers_worksheet.update_value(f"B{next_row}", email)

        # Add the first name to column C and last name to column D
        receivers_worksheet.update_value(f"C{next_row}", first_names[idx])
        receivers_worksheet.update_value(f"D{next_row}", last_names[idx])

        # Set the background color of column F to red
        receivers_worksheet.cell(f"F{next_row}").color = (1.0, 0.0, 0.0)  # RGB for red

        # Add value `1` in column G
        receivers_worksheet.update_value(f"G{next_row}", "1")

        # Update the local list to include the newly added email
        existing_emails.append(email)

        # Add this Kubestronaut to the WelcomeEmail Google Sheet Mailer
        welcome_worksheet.insert_rows(1, number=1, values=[email])
        #welcome_worksheet.update_value(f"A2", email)
    else:
        # Mention that the email is already in the file
        print(f"Kubestronaut already exists: {email}")

print("Completed processing emails!")
print("Go to https://docs.google.com/spreadsheets/d/"+KUBESTRONAUTS_WELCOME)
print("And use the mail merger with the draft name \"Welcome to the Kubestronaut program !\"")
