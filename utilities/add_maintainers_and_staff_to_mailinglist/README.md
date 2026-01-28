# Add Maintainers and Staff to Mailing List

This utility provides a GitHub Actions workflow to automatically add maintainer and staff email addresses to CNCF mailing lists (Groups.io subgroups).

## What This Workflow Does

The workflow automates the process of adding multiple email addresses to a CNCF mailing list. It can:

- Add maintainer emails from a committed file with default settings (role: none, delivery: email_delivery_single)
- Optionally add staff emails with owner role
- Perform email format validation before attempting to add addresses
- Skip comment lines (starting with #) and empty lines in email files
- Read email addresses from a committed file path to protect privacy

**Privacy and Security**: Email addresses are read from committed files rather than workflow inputs. This prevents email addresses from appearing in GitHub Actions UI or workflow run logs. The workflow validates email format without logging the actual addresses, and the processing scripts redact emails in logs by default.
## Prerequisites

1. **LFX Authentication Token**: You need a valid token from [Open Profile Developer Settings](https://openprofile.dev/developer-settings)
   - The token is short-lived (~3 hours)
   - You'll need to update it regularly when it expires

2. **Mailing List ID (SUBGROUP_ID)**: The mailing list ID from the LFX Project Admin URL
   - Found in the CNCF mailing list management URL
   - Example: If the URL is `https://projectadmin.lfx.linuxfoundation.org/project/a092M00001LkNgVQAV/collaboration/mailing-lists/manage-members/117989`, the ID is `117989`

3. **GitHub Repository Secret**: The `LFX_TOKEN` must be configured in your repository secrets
   - Go to: Repository Settings → Secrets and variables → Actions
   - Add a new repository secret named `LFX_TOKEN`
   - Paste your token from Open Profile Developer Settings

## How to Run the Workflow

### Step 1: Get Your LFX Token

1. Visit [Open Profile Developer Settings](https://openprofile.dev/developer-settings)
2. Copy your authentication token
3. **Note**: The token expires after ~3 hours, so you'll need to update the `LFX_TOKEN` secret when it expires

### Step 2: Get the Mailing List ID

1. Navigate to your CNCF mailing list in Groups.io
2. Look at the URL in your browser
3. The mailing list ID (SUBGROUP_ID) is the number at the end of the URL
4. Copy this ID

### Step 3: Prepare Email File

1. Create or update a file with maintainer email addresses (one per line)
2. The default file path is `utilities/add_maintainers_and_staff_to_mailinglist/maintainers_emails.txt`
3. You can use a different file path if needed
4. Comment lines (starting with #) and empty lines are ignored
5. Commit this file to your branch

Example `maintainers_emails.txt`:
```
# Maintainer emails for project X
maintainer1@example.com
maintainer2@example.com
# Add more emails below
```

### Step 4: Run the Workflow

1. Go to your repository's **Actions** tab in GitHub
2. Select **"Update Mailing List"** from the workflow list
3. Click **"Run workflow"**
4. Fill in the workflow inputs:
   - **Mailing list ID**: Paste the SUBGROUP_ID you copied from the mailing list URL
   - **Add staff**: Check this box if you want to add staff members from `staff_emails.txt`
   - **Email file path**: Leave default or specify a custom path to your email file (default: `utilities/add_maintainers_and_staff_to_mailinglist/maintainers_emails.txt`)
5. Click **"Run workflow"** to start

## Workflow Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `mailing_list_id` | The mailing list ID (SUBGROUP_ID) from the LFX Project Admin URL | Yes | - |
| `add_staff` | Whether to add staff members from `staff_emails.txt` | No | `false` |
| `email_file` | Path to file containing email addresses (one per line) | No | `utilities/add_maintainers_and_staff_to_mailinglist/maintainers_emails.txt` |

## Files in This Utility

- **`maintainer_list_add.sh`**: Script that adds maintainer emails to the mailing list from a file
- **`staff_list_add.sh`**: Script that adds staff emails with owner role
- **`maintainers_emails.txt`**: Template file for maintainer email addresses (one per line, supports comments)
- **`staff_emails.txt`**: Optional file containing staff email addresses (one per line)
- **`.github/workflows/update_mailing_list.yml`**: The GitHub Actions workflow file (located in the repository root)

## How It Works

1. The workflow validates the mailing list ID and checks that the email file exists
2. It validates the email format for all addresses in the file (skipping comments and empty lines)
3. It creates a temporary `config.txt` file with your token and mailing list ID
4. It runs `maintainer_list_add.sh` to add the maintainer emails from the specified file
5. If enabled, it runs `staff_list_add.sh` to add staff emails from `staff_emails.txt`
6. All temporary files are cleaned up after the workflow completes

**Note**: The email file you commit should contain one email address per line. Comment lines (starting with #) and empty lines are ignored.

## Troubleshooting

### Token Expired
- If you get authentication errors, your token may have expired
- Get a new token from [Open Profile Developer Settings](https://openprofile.dev/developer-settings)
- Update the `LFX_TOKEN` secret in your repository settings

### Invalid Email Format
- The workflow validates email formats before adding
- Ensure emails are in the format: `user@example.com`
- Check the workflow logs for specific error messages

### Mailing List ID Not Found
- Double-check the URL of your mailing list in LFX Project Admin
- The ID should be a number at the end of the URL (after `/manage-members/`)
- Example: `https://projectadmin.lfx.linuxfoundation.org/project/a092M00001LkNgVQAV/collaboration/mailing-lists/manage-members/117989`
- The ID is the number at the very end: `117989`

## Notes

- The workflow reads email addresses from a committed file path to protect privacy
- Email addresses are never exposed in GitHub Actions workflow run logs or UI
- The email file should contain one email address per line
- Comment lines starting with # are ignored
- Empty lines are ignored
- All temporary files (`config.txt`, etc.) are automatically cleaned up
- The `staff_emails.txt` file is optional and only used if the `add_staff` option is enabled
- Staff members are added with `owner` role, while maintainers are added with `none` role

### Privacy and Logging

To protect privacy, email addresses are **redacted in logs by default**:
- Emails are shown as `***@domain.com` instead of the full address
- The workflow provides a summary count of processed, succeeded, and failed additions
- This prevents email addresses from being exposed in GitHub Actions workflow logs

For local debugging only, you can set `VERBOSE=true` to log full email addresses:
```bash
VERBOSE=true ./maintainer_list_add.sh
```
**Note**: Never enable `VERBOSE=true` in CI/CD environments, as it will expose email addresses in logs.

