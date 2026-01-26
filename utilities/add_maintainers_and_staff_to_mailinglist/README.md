# Add Maintainers and Staff to Mailing List

This utility provides a GitHub Actions workflow to automatically add maintainer and staff email addresses to CNCF mailing lists (Groups.io subgroups).

## What This Workflow Does

The workflow automates the process of adding multiple email addresses to a CNCF mailing list. It can:

- Add maintainer emails with default settings (role: none, delivery: email_delivery_single)
- Optionally add staff emails with owner role
- Validate email formats before adding
- Handle space-separated or newline-separated email lists

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

### Step 3: Run the Workflow

1. Go to your repository's **Actions** tab in GitHub
2. Select **"Update Mailing List"** from the workflow list
3. Click **"Run workflow"**
4. Fill in the workflow inputs:
   - **Mailing list ID**: Paste the SUBGROUP_ID you copied from the mailing list URL
   - **Add staff**: Check this box if you want to add staff members from `staff_emails.txt`
   - **Email addresses to add**: Paste the list of maintainer email addresses
     - You can paste emails separated by spaces or one per line
     - The workflow will automatically convert space-separated emails to newline format
5. Click **"Run workflow"** to start

## Workflow Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `mailing_list_id` | The mailing list ID (SUBGROUP_ID) from the LFX Project Admin URL | Yes | - |
| `add_staff` | Whether to add staff members from `staff_emails.txt` | No | `false` |
| `add_emails` | Email addresses to add (space-separated or one per line) | Yes | - |

## Files in This Utility

- **`maintainer_list_add.sh`**: Script that adds maintainer emails to the mailing list
- **`staff_list_add.sh`**: Script that adds staff emails with owner role
- **`staff_emails.txt`**: Optional file containing staff email addresses (one per line)
- **`.github/workflows/update_mailing_list.yml`**: The GitHub Actions workflow file (located in the repository root)

## How It Works

1. The workflow creates a temporary `config.txt` file with your token and mailing list ID
2. It validates and formats the email addresses you provided
3. It runs `maintainer_list_add.sh` to add the maintainer emails
4. If enabled, it runs `staff_list_add.sh` to add staff emails from `staff_emails.txt`
5. All temporary files are cleaned up after the workflow completes

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

- The workflow uses the `utilities/add_maintainers_and_staff_to_mailinglist` directory as its working directory
- All temporary files (`config.txt`, `maintainers_emails_temp.txt`, etc.) are automatically cleaned up
- The `staff_emails.txt` file is optional and only used if the `add_staff` option is enabled
- Staff members are added with `owner` role, while maintainers are added with `none` role

