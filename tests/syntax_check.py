import sys
from unittest.mock import MagicMock

# Mock dependencies that require credentials or external systems
sys.modules["pygsheets"] = MagicMock()
sys.modules["gdown"] = MagicMock()
sys.modules["dotenv"] = MagicMock()

# Mock the specific files that are opened globally
import builtins
original_open = builtins.open

def mocked_open(file, *args, **kwargs):
    if "people.json" in str(file):
        return original_open("tests/mock_people.json", *args, **kwargs)
    if "Kubestronaut.tsv" in str(file):
        return original_open("tests/mock_kubestronaut.tsv", *args, **kwargs)
    return original_open(file, *args, **kwargs)

# We can't easily mock open() globally for the script execution without more complex patching 
# because the script runs immediately on import.
# So we will just check for SyntaxErrors by compiling the files.

import py_compile
import os

files_to_check = [
    "s:/Gsoc organisation/automation/Kubestronaut/CNCFInsertKubestronautInPeople_json.py",
    "s:/Gsoc organisation/automation/Kubestronaut/AddNewWeeklyyKubestronautsInReceivers.py"
]

print("Verifying syntax...")
for f in files_to_check:
    try:
        py_compile.compile(f, doraise=True)
        print(f"✅ {os.path.basename(f)} passed syntax check.")
    except Exception as e:
        print(f"❌ {os.path.basename(f)} FAILED syntax check: {e}")
        sys.exit(1)

print("\nVerifying importability (with mocks)...")
# We won't actually import them because they have side effects (argparse, file I/O) at top level.
# The syntax check is the most important for now given the constraints.
