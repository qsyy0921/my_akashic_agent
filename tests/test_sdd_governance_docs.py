from pathlib import Path
import re


ROOT = Path(__file__).resolve().parents[1]
SDD_DIR = ROOT / "docs" / "sdd"
ATDD_DIR = ROOT / "docs" / "atdd"
TDD_DIR = ROOT / "docs" / "tdd"


def test_sdd_governance_documents_exist_and_define_roles() -> None:
    required = [
        "TODO.md",
        "DONE.md",
        "BACKLOG.md",
        "LIVE_CHECKS.md",
        "OPEN_ISSUES.md",
    ]
    for name in required:
        assert (SDD_DIR / name).exists(), f"missing SDD governance document: {name}"

    open_issues = (SDD_DIR / "OPEN_ISSUES.md").read_text(encoding="utf-8")
    for name in required:
        assert f"`{name}`" in open_issues, f"OPEN_ISSUES.md must explain {name}"
    assert "当前未解决问题" in open_issues
    assert "| ID | 领域 | 问题 | 影响 | 下一步 | 状态 |" in open_issues


def test_atdd_and_tdd_guides_exist() -> None:
    atdd_readme = ATDD_DIR / "README.md"
    tdd_readme = TDD_DIR / "README.md"
    assert atdd_readme.exists(), "missing ATDD guide: docs/atdd/README.md"
    assert tdd_readme.exists(), "missing TDD guide: docs/tdd/README.md"

    atdd_text = atdd_readme.read_text(encoding="utf-8")
    tdd_text = tdd_readme.read_text(encoding="utf-8")
    assert "Acceptance Test-Driven Development" in atdd_text
    assert "Test-Driven Development" in tdd_text


def test_open_issues_table_rows_are_structured() -> None:
    open_issues = (SDD_DIR / "OPEN_ISSUES.md").read_text(encoding="utf-8")
    rows = [
        line
        for line in open_issues.splitlines()
        if line.startswith("| OI-")
    ]
    assert rows, "OPEN_ISSUES.md must contain at least one issue row"

    seen_ids: set[str] = set()
    allowed_statuses = {"Open", "In Progress", "Blocked", "Resolved"}
    for row in rows:
        cells = [cell.strip() for cell in row.strip().strip("|").split("|")]
        assert len(cells) == 6, f"open issue row must have 6 cells: {row}"
        issue_id, area, problem, impact, next_step, status = cells
        assert re.fullmatch(r"OI-\d{3}", issue_id), f"invalid open issue id: {issue_id}"
        assert issue_id not in seen_ids, f"duplicate open issue id: {issue_id}"
        seen_ids.add(issue_id)
        assert area, f"{issue_id} missing area"
        assert problem, f"{issue_id} missing problem"
        assert impact, f"{issue_id} missing impact"
        assert next_step, f"{issue_id} missing next step"
        assert status in allowed_statuses, f"{issue_id} has invalid status: {status}"
