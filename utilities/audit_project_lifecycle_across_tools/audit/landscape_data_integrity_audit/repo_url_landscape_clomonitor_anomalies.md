# Repo URL anomalies for CLOMonitor (Landscape vs CLOMonitor)

Generated from `landscape_source_diff.json` (`field = repo_url`) with `curl` URL checks.

Rule: when both URLs are GitHub and org/owner matches, repo path differences are treated as aligned.
This report includes only non-aligned (anomalous) CLOMonitor vs Landscape rows.

| Project | Maturity | CLOMonitor URL | CLOMonitor | Landscape URL | Landscape | Org match | Same final destination | Result | Note |
|---|---|---|---|---|---|---|---|---|---|
