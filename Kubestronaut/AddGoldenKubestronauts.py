import pygsheets
import os
from dotenv import load_dotenv
import re
import json
from contextlib import contextmanager
import time
import secrets

# Authenticate
gc = pygsheets.authorize(service_file='kubestronauts-handling-service-file.json')

# Load environment variables
load_dotenv()
GOLDEN_KUBESTRONAUTS_WEEKLY_TEMP = os.getenv('GOLDEN_KUBESTRONAUTS_WEEKLY_TEMP')
GOLDEN_KUBESTRONAUTS_WELCOME = os.getenv('GOLDEN_KUBESTRONAUTS_WELCOME')
KUBESTRONAUTS_INFOS = os.getenv('KUBESTRONAUTS_INFOS')

# Open sheets
golden_weekly_temp_sheet = gc.open_by_key(GOLDEN_KUBESTRONAUTS_WEEKLY_TEMP)
golden_welcome_sheet = gc.open_by_key(GOLDEN_KUBESTRONAUTS_WELCOME)
infos_sheet = gc.open_by_key(KUBESTRONAUTS_INFOS)

weekly_temp_worksheet = golden_weekly_temp_sheet.sheet1
welcome_worksheet = golden_welcome_sheet.sheet1
infos_worksheet = infos_sheet.sheet1

# Get emails and welcome emails
emails_to_check = weekly_temp_worksheet.get_col(4, include_tailing_empty=False)
already_welcome_emails = welcome_worksheet.get_col(3, include_tailing_empty=False)

# Load people.json
with open('../../people/people.json', "r+", encoding='utf-8') as jsonfile:
    data = json.load(jsonfile)

# Step 1 - Validate
valid_kubestronauts = []
invalid_kubestronauts = []
already_welcomed = []

print(f"\n🔍 Validating {len(emails_to_check)} Kubestronauts...\n")

for idx, email in enumerate(emails_to_check, start=1):
    email_indiv = re.findall(r'[\w.+-]+@[\w-]+\.[\w.-]+', email)
    found = False

    for email_sep in email_indiv:
        if email_sep in already_welcome_emails:
            already_welcomed.append(email_sep)
            found = True
            print(f"[{idx:2}/{len(emails_to_check)}] {email_sep:<40} ✅ already welcomed")
            break

        cells = infos_worksheet.find(pattern=email_sep, cols=(13, 13), matchEntireCell=True)
        if len(cells) == 1:
            row = cells[0].row
            full_name = infos_worksheet.cell("B" + str(row)).value.strip()
            valid_kubestronauts.append({
                "email": email_sep,
                "row": row,
                "full_name": full_name,
                "lfid": weekly_temp_worksheet.cell("A" + str(idx)).value.strip(),
                "token": secrets.token_hex(16)
            })
            found = True
            print(f"[{idx:2}/{len(emails_to_check)}] {email_sep:<40} ✅ OK")
            break

    if not found:
        first_name = weekly_temp_worksheet.cell((idx, 1)).value.strip()
        last_name = weekly_temp_worksheet.cell((idx, 2)).value.strip()
        invalid_kubestronauts.append((email, first_name, last_name))
        print(f"[{idx:2}/{len(emails_to_check)}] {email:<40} ❌ not found ({first_name} {last_name})")

# Stop if any are invalid
if invalid_kubestronauts:
    print("\n❌ Some Kubestronauts could not be matched in infos sheet:")
    for email, first, last in invalid_kubestronauts:
        print(f" - {email:<35} ({first} {last})")
    print("\n❗ Please fix the above email(s) before re-running the script.")
    exit(1)

# Context manager for rollback
@contextmanager
def rollback_guard(spreadsheet, main_worksheet_title, temp_worksheet_title='weekly_temp'):
    try:
        main_ws = spreadsheet.worksheet_by_title(main_worksheet_title)
        try:
            temp_ws = spreadsheet.worksheet_by_title(temp_worksheet_title)
            spreadsheet.del_worksheet(temp_ws)
        except pygsheets.WorksheetNotFound:
            pass
        temp_ws = main_ws.copy_to(spreadsheet.id)
        temp_ws.title = temp_worksheet_title
        time.sleep(2)
        yield main_ws
        try:
            temp_ws = spreadsheet.worksheet_by_title(temp_worksheet_title)
            spreadsheet.del_worksheet(temp_ws)
        except pygsheets.WorksheetNotFound:
            pass
    except Exception as e:
        print("❌ An error occurred, rolling back changes...")
        spreadsheet.del_worksheet(main_ws)
        temp_ws.title = main_worksheet_title
        print("✅ Rollback completed.")
        raise e

# Step 2 - Apply changes
NON_managed_Golden_Kubestronauts = {}

with rollback_guard(golden_welcome_sheet, main_worksheet_title='Sheet1', temp_worksheet_title='weekly_temp') as welcome_worksheet:
    for k in valid_kubestronauts:
        print(f"✨ Welcoming Kubestronaut: {k['email']}")

        # Mark as GK in infos sheet
        infos_worksheet.update_value("Z" + str(k["row"]), "1")

        # Format names with capitalized first letter
        name_parts = k["full_name"].strip().split()
        if len(name_parts) >= 2:
            first = name_parts[0].capitalize()
            last = " ".join([p.capitalize() for p in name_parts[1:]])
        else:
            first = name_parts[0].capitalize()
            last = ""

        welcome_worksheet.insert_rows(1, number=1, values=["", k["lfid"], k["email"], first, last, "", k["token"]])

        # Tag in people.json
        tagged = False
        for person in data:
            if person["name"].lower() == k["full_name"].lower():
                if "Golden-Kubestronaut" not in person["category"]:
                    person["category"].append("Golden-Kubestronaut")
                tagged = True
                break
        if not tagged:
            NON_managed_Golden_Kubestronauts[k["email"]] = "Not found in people.json"

# Save updated people.json
with open('../../people/people.json', "w", encoding='utf-8') as jsonfile:
    jsonfile.write(json.dumps(data, indent=4, ensure_ascii=False))

# Final summary
if NON_managed_Golden_Kubestronauts:
    print("\n❗ Kubestronauts not added to people.json:")
    for email, reason in NON_managed_Golden_Kubestronauts.items():
        print(f" - {email}: {reason}")

if already_welcomed:
    print("\nℹ️ Kubestronauts already welcomed (skipped):")
    for email in already_welcomed:
        print(f" - {email}")

print("\n✅ All valid Kubestronauts have been welcomed.")
print("👉 Go to https://docs.google.com/spreadsheets/d/" + GOLDEN_KUBESTRONAUTS_WELCOME)
print("📩 Use the mail merger with the draft: \"Welcome to the Golden Kubestronaut program !\"")

