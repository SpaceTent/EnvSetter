#!/usr/bin/env python3
"""Run a Python module with variables supplied by EnvVault via EnvSetter."""

import os
import subprocess
import sys


def main() -> int:
    command = sys.argv[1:] or [sys.executable, "-m", "myapp"]
    return subprocess.call(["envsetter", "run", "--", *command], env=os.environ.copy())


if __name__ == "__main__":
    raise SystemExit(main())
