import pygsheets
import os
from dotenv import load_dotenv
import re

# ---------------------------
# Helpers
# ---------------------------
EMAIL_RE = re.compile(r'[\w.+-]+@[\w-]+\.[\w.-]+')

def norm(s: str) -> str:
    return (s or "").strip().lower()

def extract_emails(s: str):
    return [norm(e) for e in EMAIL_RE.findall(s or "")]

def force_refresh(spreadsheet, worksheet, label=""):
    """
    Best-effort refresh. Depending on pygsheets versions/objects,
    some methods may not exist -> we just warn and continue.
    """
    print(f"\n[REFRESH] Forcing refresh {label} ...")
    try:
        spreadsheet.fetch_properties(fetch_sheets=True)
        print("[REFRESH] spreadsheet.fetch_properties(fetch_sheets=True) OK")
    except Exception as e:
        print(f"[REFRESH] WARN spreadsheet.fetch_properties failed: {e}")

    try:
        worksheet.refresh()
        print("[REFRESH] worksheet.refresh() OK")
    except Exception as e:
        print(f"[REFRESH] WARN worksheet.refresh failed: {e}")

# ---------------------------
# Auth + env
# ---------------------------
gc = pygsheets.authorize(service_file='kubestronauts-handling-service-file.json')

load_dotenv()
KUBESTRONAUT_RECEIVERS = os.getenv('KUBESTRONAUT_RECEIVERS')
KUBESTRONAUTS_WEEKLY_TEMP = os.getenv('KUBESTRONAUTS_WEEKLY_TEMP')
KUBESTRONAUTS_WELCOME = os.getenv('KUBESTRONAUTS_WELCOME')

if not KUBESTRONAUT_RECEIVERS or not KUBESTRONAUTS_WEEKLY_TEMP or not KUBESTRONAUTS_WELCOME:
    raise RuntimeError("Missing env var(s): KUBESTRONAUT_RECEIVERS / KUBESTRONAUTS_WEEKLY_TEMP / KUBESTRONAUTS_WELCOME")

# ---------------------------
# Open sheets + worksheets
# ---------------------------
receivers_sheet = gc.open_by_key(KUBESTRONAUT_RECEIVERS)
weekly_temp_sheet = gc.open_by_key(KUBESTRONAUTS_WEEKLY_TEMP)
welcome_sheet = gc.open_by_key(KUBESTRONAUTS_WELCOME)

weekly_temp_worksheet = weekly_temp_sheet.sheet1
receivers_worksheet = receivers_sheet.worksheet_by_title("Invited")
welcome_worksheet = welcome_sheet.sheet1

# Debug URLs (to ensure you're looking at the same doc/tab)
print("Receivers spreadsheet URL:", receivers_sheet.url)
try:
    print("Receivers worksheet:", receivers_worksheet.title, "|", receivers_worksheet.url)
except Exception:
    print("Receivers worksheet:", receivers_worksheet.title)

print("Weekly temp spreadsheet URL:", weekly_temp_sheet.url)
print("Welcome spreadsheet URL:", welcome_sheet.url)

# Force refresh (best-effort)
force_refresh(receivers_sheet, receivers_worksheet, label="(receivers/Invited)")
force_refresh(weekly_temp_sheet, weekly_temp_worksheet, label="(weekly temp)")
force_refresh(welcome_sheet, welcome_worksheet, label="(welcome)")

# ---------------------------
# Read data
# ---------------------------
emails_to_check = weekly_temp_worksheet.get_col(3, include_tailing_empty=False)
first_names = weekly_temp_worksheet.get_col(1, include_tailing_empty=False)
last_names = weekly_temp_worksheet.get_col(2, include_tailing_empty=False)

# Column B in receivers sheet
existing_cells = receivers_worksheet.get_col(2, include_tailing_empty=False)

# Build a normalized set of existing emails (exact email matches, even if a cell contains multiple)
existing_set = set()
for cell in existing_cells:
    for e in extract_emails(cell):
        existing_set.add(e)

print(f"\nLoaded {len(emails_to_check)} emails from weekly temp.")
print(f"Loaded {len(existing_cells)} non-empty cells from receivers column B.")
print(f"Extracted {len(existing_set)} unique emails from receivers column B.")

# ---------------------------
# Process
# ---------------------------
for idx, raw_email_cell in enumerate(emails_to_check):
    print("\n----------------------------------------")
    print("[WEEKLY] Raw:", raw_email_cell)

    email_indiv = extract_emails(raw_email_cell)
    if not email_indiv:
        print("[WEEKLY] No valid email found in this cell -> SKIP")
        continue

    existing = False

    # For each email found in the weekly cell, check if it exists in receivers.
    for email_sep in email_indiv:
        if email_sep in existing_set:
            # Also print where it matches (row + exact cell content) for debugging visibility/filter issues
            for row, cell in enumerate(existing_cells, start=1):
                if email_sep in extract_emails(cell):
                    print(f"[MATCH] Found '{email_sep}' in receivers: row={row}, col=B, cell={cell!r}")
                    break
            existing = True
            break

    if existing:
        print(f"[SKIP] Kubestronaut already exists (per API read): {email_indiv}")
        continue

    # Not existing -> add
    # NOTE: next_row is computed from the *API-read* list, so if rows are hidden/filtered,
    # they still count and your row number may look "far" in the UI.
    next_row = len(existing_cells) + 1

    print(f"[ADD] Adding Kubestronaut(s): {email_indiv} at row {next_row}")

    # Insert a blank row before the next_row (kept same behavior as your original script)
    receivers_worksheet.insert_rows(next_row - 1, number=1)

    # Write values
    receivers_worksheet.update_value(f"B{next_row}", raw_email_cell)
    receivers_worksheet.update_value(f"C{next_row}", first_names[idx] if idx < len(first_names) else "")
    receivers_worksheet.update_value(f"D{next_row}", last_names[idx] if idx < len(last_names) else "")

    # Background color of column F to red
    receivers_worksheet.cell(f"F{next_row}").color = (1.0, 0.0, 0.0)

    # Value `1` in column G
    receivers_worksheet.update_value(f"G{next_row}", "1")

    # Update local caches (so subsequent checks in the same run see it)
    existing_cells.append(raw_email_cell)
    for e in email_indiv:
        existing_set.add(e)

    # Add this Kubestronaut to the WelcomeEmail Google Sheet Mailer
    welcome_worksheet.insert_rows(1, number=1, values=[raw_email_cell])

print("\nCompleted processing emails!")
print("Go to https://docs.google.com/spreadsheets/d/" + KUBESTRONAUTS_WELCOME)

cmd = "echo 'Welcome to the Kubestronaut Program – Next Steps, Jacket Info & More!' | pbcopy"
os.system(cmd)

print('And use the mail merger with the draft name "Welcome to the Kubestronaut Program – Next Steps, Jacket Info & More!"')

