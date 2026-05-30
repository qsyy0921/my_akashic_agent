from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from eval.group_memory.runner import run_group_memory_fixture_eval


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--fixture",
        default="tests/fixtures/group_memory_open_strategy_dataset.json",
    )
    parser.add_argument("--workspace", default=".tmp/group_memory_eval")
    parser.add_argument("--min-top1-accuracy", type=float, default=1.0)
    parser.add_argument("--min-evidence-coverage", type=float, default=1.0)
    args = parser.parse_args()

    report = run_group_memory_fixture_eval(
        fixture_path=args.fixture,
        workspace=args.workspace,
        min_top1_accuracy=args.min_top1_accuracy,
        min_evidence_coverage=args.min_evidence_coverage,
    )
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0 if bool(report.get("passed")) else 1


if __name__ == "__main__":
    sys.exit(main())
