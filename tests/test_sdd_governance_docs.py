from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SDD_DIR = ROOT / "docs" / "sdd"


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
