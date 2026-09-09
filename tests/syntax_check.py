"""Syntax gate: py_compile every tracked Python file in the automation directories.

Files are compiled, never imported, so credential files and network access are not needed.
"""

import py_compile
import subprocess
import sys

CHECK_ROOTS = ("Ambassadors", "Kubestronaut", "utilities", ".github/scripts", ".github/actions", "tests")


def tracked_python_files() -> list[str]:
    out = subprocess.run(
        ["git", "ls-files", "--", *(f"{root}/*.py" for root in CHECK_ROOTS)],
        check=True,
        capture_output=True,
        text=True,
    ).stdout
    return sorted(line for line in out.splitlines() if line)


def main() -> int:
    files = tracked_python_files()
    if not files:
        print("No Python files found to check.", file=sys.stderr)
        return 1

    failures = 0
    for path in files:
        try:
            py_compile.compile(path, doraise=True)
            print(f"OK   {path}")
        except py_compile.PyCompileError as exc:
            failures += 1
            print(f"FAIL {path}: {exc.msg}")

    print(f"\n{len(files) - failures}/{len(files)} files passed syntax check.")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
