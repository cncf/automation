import pygsheets
import os
from dotenv import load_dotenv
import re
import json
from contextlib import contextmanager
import time
import secrets
from typing import Dict, List, Tuple, Optional
from datetime import datetime


# ----------------------------
# Manual matching cache
# ----------------------------
MANUAL_CACHE_FILE = "Kubestronaut_manual_matching.json"

def load_manual_cache(cache_file: str) -> Dict[str, str]:
    """
    Load manual matching cache from JSON file.
    Format: {"source_email": "target_email"}
    Returns empty dict if file doesn't exist.
    """
    if not os.path.exists(cache_file):
        return {}
    try:
        with open(cache_file, "r", encoding="utf-8") as f:
            cache = json.load(f)
        print(f"[INFO] Loaded {len(cache)} manual matches from cache: {cache_file}")
        return cache
    except Exception as e:
        print(f"[WARN] Could not load manual cache from {cache_file}: {e}")
        return {}

def save_manual_cache(cache: Dict[str, str], cache_file: str):
    """
    Save manual matching cache to JSON file.
    """
    try:
        with open(cache_file, "w", encoding="utf-8") as f:
            json.dump(cache, f, indent=2, ensure_ascii=False)
        print(f"[INFO] Saved {len(cache)} manual matches to cache: {cache_file}")
    except Exception as e:
        print(f"[WARN] Could not save manual cache to {cache_file}: {e}")

def norm_email(s: str) -> str:
    """Normalize email: strip and lowercase."""
    return (s or "").strip().lower()

def prompt_manual_match(email: str, first_name: str, last_name: str,
                       infos_worksheet) -> Optional[str]:
    """
    Prompt user to manually match an email by searching in the infos sheet.
    Returns the matched email or None if user skips.
    """
    print("\n" + "=" * 78)
    print(f"❌ Email not found: {email}")
    print(f"   Name from Golden Kubestronaut sheet: {first_name} {last_name}")
    print("-" * 78)
    print("Please help match this person by providing their email from KUBESTRONAUTS_INFOS.")
    print("You can:")
    print("  1. Search in the KUBESTRONAUTS_INFOS sheet for this person")
    print("  2. Copy their email (column M) and paste it here")
    print("  3. Type 'skip' to skip this person for now")
    print("-" * 78)

    while True:
        ans = input("Enter the correct email (or 'skip'): ").strip()
        ans_lower = ans.lower()

        if ans_lower in ("skip", "s", ""):
            print("⏭️  Skipping this person.")
            return None

        # Check if it looks like an email
        if "@" in ans and "." in ans:
            # Verify this email exists in infos sheet
            cells = infos_worksheet.find(pattern=ans, cols=(13, 13), matchEntireCell=True)
            if len(cells) == 1:
                row = cells[0].row
                full_name = infos_worksheet.cell("B" + str(row)).value.strip()
                print(f"✅ Found in KUBESTRONAUTS_INFOS: {full_name}")
                confirm = input(f"   Confirm this is the right person? (y/n): ").strip().lower()
                if confirm in ("y", "yes", ""):
                    return ans
                else:
                    print("   Let's try again.")
                    continue
            elif len(cells) == 0:
                print(f"⚠️  Email '{ans}' not found in KUBESTRONAUTS_INFOS (column M).")
                retry = input("   Try another email? (y/n): ").strip().lower()
                if retry not in ("y", "yes", ""):
                    return None
            else:
                print(f"⚠️  Found {len(cells)} matches for '{ans}' in KUBESTRONAUTS_INFOS.")
                print("   This shouldn't happen. Please verify the email.")
                continue
        else:
            print("⚠️  This doesn't look like a valid email address. Please try again or type 'skip'.")

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



now = datetime.now()
date_string_yyyymmdd = now.strftime("%Y%m%d")



# Load people.json
with open('../../people/people.json', "r+", encoding='utf-8') as jsonfile:
    data = json.load(jsonfile)

# Load manual matching cache
manual_cache = load_manual_cache(MANUAL_CACHE_FILE)
cache_modified = False

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

        # Try direct lookup first
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

        # Not found directly - check cache
        email_norm = norm_email(email_sep)
        if email_norm in manual_cache:
            cached_email = manual_cache[email_norm]
            cells = infos_worksheet.find(pattern=cached_email, cols=(13, 13), matchEntireCell=True)
            if len(cells) == 1:
                row = cells[0].row
                full_name = infos_worksheet.cell("B" + str(row)).value.strip()
                valid_kubestronauts.append({
                    "email": cached_email,
                    "row": row,
                    "full_name": full_name,
                    "lfid": weekly_temp_worksheet.cell("A" + str(idx)).value.strip(),
                    "token": secrets.token_hex(16)
                })
                found = True
                print(f"[{idx:2}/{len(emails_to_check)}] {email_sep:<40} ✅ OK (cached: {cached_email})")
                break
            else:
                print(f"[{idx:2}/{len(emails_to_check)}] {email_sep:<40} ⚠️  Cached email '{cached_email}' is invalid")
                # Continue to manual matching

    if not found:
        first_name = weekly_temp_worksheet.cell((idx, 1)).value.strip()
        last_name = weekly_temp_worksheet.cell((idx, 2)).value.strip()

        # Try manual matching
        print(f"[{idx:2}/{len(emails_to_check)}] {email:<40} ❌ not found ({first_name} {last_name})")
        matched_email = prompt_manual_match(email, first_name, last_name, infos_worksheet)

        if matched_email:
            # Found via manual matching
            cells = infos_worksheet.find(pattern=matched_email, cols=(13, 13), matchEntireCell=True)
            if len(cells) == 1:
                row = cells[0].row
                full_name = infos_worksheet.cell("B" + str(row)).value.strip()
                valid_kubestronauts.append({
                    "email": matched_email,
                    "row": row,
                    "full_name": full_name,
                    "lfid": weekly_temp_worksheet.cell("A" + str(idx)).value.strip(),
                    "token": secrets.token_hex(16)
                })
                # Save to cache
                for email_sep in email_indiv:
                    manual_cache[norm_email(email_sep)] = matched_email
                cache_modified = True
                print(f"[{idx:2}/{len(emails_to_check)}] {email:<40} ✅ OK (manual: {matched_email})")
                found = True

        if not found:
            invalid_kubestronauts.append((email, first_name, last_name))

# Save manual cache if modified
if cache_modified:
    save_manual_cache(manual_cache, MANUAL_CACHE_FILE)

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
        infos_worksheet.update_value("Z" + str(k["row"]), date_string_yyyymmdd)

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

