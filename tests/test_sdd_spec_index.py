from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
AGENT_GATEWAY_SPEC_DIR = ROOT / "docs" / "sdd" / "specs" / "agent-gateway"
INDEX_PATH = AGENT_GATEWAY_SPEC_DIR / "000-index.md"


def test_agent_gateway_specs_are_indexed_by_filename() -> None:
    index = INDEX_PATH.read_text(encoding="utf-8")
    specs = sorted(
        path.name
        for path in AGENT_GATEWAY_SPEC_DIR.glob("[0-9][0-9][0-9]-*.md")
        if path.name != INDEX_PATH.name
    )

    missing = [name for name in specs if f"`{name}`" not in index]

    assert not missing, "Missing agent-gateway specs in 000-index.md: " + ", ".join(missing)
