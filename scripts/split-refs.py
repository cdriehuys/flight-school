#!/usr/bin/env python3

"""
Parse a reference list from an ACS document and output it as a JSON array.
"""

import json
import sys


def main(refs: str):
    parts = [s.strip() for s in refs.split(";")]
    json.dump(parts, sys.stdout)
    print()

if __name__ == "__main__":
    main(sys.argv[1])
