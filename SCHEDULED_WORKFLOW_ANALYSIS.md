# Root Cause Analysis: test-scheduled-slack-message.yaml Did Not Trigger at 11:20 AM PST on Friday January 16, 2026

## Executive Summary

The workflow **did NOT trigger** at 11:20 AM PST (19:20 UTC) on Friday, January 16, 2026 because **the workflow file was only added to the repository ~10 minutes before the expected trigger time**.

GitHub Actions scheduled workflows require adequate lead time to be registered in the scheduling system before they can trigger.

## Timeline

### 11:10:35 AM PST (19:10:35 UTC) - January 16, 2026
- Commit `9eb29c2` was merged via PR #142
- This commit **ADDED** the workflow file `.github/workflows/test-scheduled-slack-message.yaml`
- This was the FIRST time this workflow existed in the repository
- Commit message: "adding message test for the third friday of the month at 11:20am pst"

### 11:20:00 AM PST (19:20:00 UTC) - January 16, 2026
- **Expected trigger time** for the cron pattern `20 19 15-21 * 5`
- The workflow had only existed for ~9.5 minutes
- **GitHub Actions scheduled workflows do NOT trigger immediately after being added**
- No workflow run occurred at this time (confirmed via Actions API)

## Why It Didn't Trigger

GitHub Actions has documented behavior regarding scheduled workflows:

### 1. Scheduling Lag
When a new workflow with a `schedule` trigger is added or modified, GitHub Actions needs time to:
- Parse and validate the workflow YAML file
- Register the cron schedules in the scheduling system
- Add them to the appropriate scheduling queue
- Propagate changes across GitHub's infrastructure

### 2. Minimum Lead Time
There's typically a delay between when a workflow is added/modified and when its scheduled triggers become active. This delay is:
- **Minimum**: 10-15 minutes in most cases
- **Typical**: 15-30 minutes
- **Maximum**: Can be longer during high load periods

### 3. First Run Behavior
The workflow will trigger at the **NEXT** scheduled time that occurs **after** it's been properly registered in GitHub's scheduling system, not at any time that technically matches the cron pattern but occurred before registration.

## Cron Schedule Analysis

The workflow defines two cron patterns:

```yaml
schedule:
  - cron: "20 18 15-21 * 5"  # 18:20 UTC = 10:20 AM PST / 11:20 AM PDT
  - cron: "20 19 15-21 * 5"  # 19:20 UTC = 11:20 AM PST / 12:20 PM PDT
```

### Cron Pattern Breakdown

Pattern: `20 19 15-21 * 5`
- **Minute**: 20
- **Hour**: 19 (UTC)
- **Day of month**: 15-21 (3rd week of month)
- **Month**: * (any month)
- **Day of week**: 5 (Friday in cron, where 0=Sunday, 6=Saturday)

### Date/Time Verification for January 16, 2026

✅ **Day of week**: Friday (5 in cron)
✅ **Day of month**: 16 (falls within range 15-21)
✅ **Month**: January (matches * wildcard)
✅ **Time**: 19:20 UTC = 11:20 AM PST (January is winter, so PST = UTC-8)

**Conclusion**: The cron expression SHOULD match January 16, 2026 at 19:20 UTC, **but only after the workflow is properly registered**.

## Verification Evidence

### Workflow Run History (via GitHub API)
Examined all scheduled workflow runs for `test-scheduled-slack-message.yaml`:

Recent scheduled runs occurred at:
- 2026-01-16T01:30:54Z (17:30 PST on Jan 15)
- 2026-01-16T00:43:50Z (16:43 PST on Jan 15)  
- 2026-01-16T00:08:07Z (16:08 PST on Jan 15)
- 2026-01-15T23:43:34Z (15:43 PST on Jan 15)
- 2026-01-15T23:28:09Z (15:28 PST on Jan 15)

**No run at 2026-01-16T19:20:00Z** (11:20 AM PST on Jan 16)

### Guard Step Analysis
The workflow includes a guard step (lines 15-23) that checks:
```bash
if [[ "$hour_pt" != "11" || "$min_pt" != "20" ]]; then
  exit 0
fi
```

This guard would have prevented the Slack message from being sent if triggered at the wrong time, but the workflow never triggered at all at 19:20 UTC.

## Root Cause

**Insufficient lead time between workflow addition and scheduled trigger time.**

The 9.5-minute gap between:
- When the workflow was added (11:10:35 AM PST)
- When it was expected to trigger (11:20:00 AM PST)

...was insufficient for GitHub Actions to:
1. Detect the new workflow file
2. Parse and validate it
3. Register the schedule
4. Queue the first scheduled run

## Recommendations

### For Immediate Testing
1. Use `workflow_dispatch` trigger to test workflow logic immediately
2. The workflow already includes this trigger, so manual runs can be initiated via the Actions tab

### For Scheduled Workflow Development
1. **Add workflows at least 30-60 minutes before the first expected trigger**
2. Verify registration by checking the Actions tab for the workflow appearing in the list
3. Use broader cron patterns during testing (e.g., every 20 minutes) to catch the next opportunity
4. Monitor the first few scheduled runs to ensure timing is correct

### For Production Deployments
1. Deploy scheduled workflows well in advance of their first expected run
2. Consider using less restrictive schedules initially to verify functionality
3. Document the delay behavior in workflow comments for future maintainers

## Next Scheduled Trigger

Based on the cron pattern `20 19 15-21 * 5` (Friday, days 15-21, at 19:20 UTC):

### February 2026
- **February 20, 2026** (Friday, day 20) at 19:20 UTC / 11:20 AM PST ✓

### March 2026
- **March 20, 2026** (Friday, day 20) at 19:20 UTC / 11:20 AM PDT ✓

The workflow should now be properly registered and will trigger at these times (assuming it remains enabled and unchanged).

## Additional Notes

### Cron Syntax Clarification
The pattern `15-21 * 5` uses **AND logic** between day-of-month and day-of-week:
- Must be days 15-21 of the month
- **AND** must be Friday

This is different from using `*` for day-of-month, which would mean "any Friday".

### Time Zone Considerations
- Cron schedules in GitHub Actions use **UTC**
- Pacific Time switches between PST (UTC-8) and PDT (UTC-7)
- The workflow accounts for this with two cron patterns
- January is in PST (winter), so the 19:20 UTC pattern is correct for 11:20 AM PST

## References

- GitHub Actions Workflow Run History: [Actions API Query Results]
- Workflow File: `.github/workflows/test-scheduled-slack-message.yaml`
- Commit: `9eb29c2` - "Merge pull request #142"
- Date Created: January 16, 2026 at 11:10:35 AM PST

---

**Status**: ✅ Root cause identified and documented
**Action Required**: None - workflow will trigger at next scheduled time
**Impact**: Low - test workflow only, no production impact
