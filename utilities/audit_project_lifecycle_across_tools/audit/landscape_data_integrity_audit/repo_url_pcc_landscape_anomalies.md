# Repo URL anomalies for PCC (Landscape vs PCC)

Generated from `landscape_source_diff.json` (`field = repo_url`) with `curl` URL checks.

Rule: when both URLs are GitHub and org/owner matches, repo path differences are treated as aligned.
This report includes only non-aligned (anomalous) PCC vs Landscape rows.

| Project | Maturity | PCC URL | PCC | Landscape URL | Landscape | Org match | Same final destination | Result | Note |
|---|---|---|---|---|---|---|---|---|---|
| container2wasm | sandbox | https://github.com/ktock/container2wasm | ✅ 200 | https://github.com/container2wasm/container2wasm | ✅ 200 | No | No | Mismatch | Different final destinations: PCC `https://github.com/ktock/container2wasm` vs Landscape `https://github.com/container2wasm/container2wasm`. |
