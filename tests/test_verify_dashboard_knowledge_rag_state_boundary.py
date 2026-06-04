from __future__ import annotations

from scripts.verify_dashboard_knowledge_rag_state_boundary import (
    build_dashboard_knowledge_rag_state_boundary_verification,
)


def _runtime_direct_payload() -> dict:
    return {
        "data": {
            "totals": {
                "targets": 2,
                "enabled": 2,
                "ready": 1,
                "warning": 1,
                "blocked": 0,
                "lagging": 0,
                "high_pressure": 0,
                "receiver_connected": 2,
                "configured_rag_datasets": 1,
                "configured_rag_dataset_not_started": 0,
                "rag_datasets": 1,
                "rag_dataset_blocked": 0,
                "rag_dataset_warning": 1,
                "rag_dataset_index_ready": 0,
                "rag_dataset_index_missing_snapshot": 1,
                "rag_dataset_index_empty": 0,
                "rag_dataset_index_lagging": 0,
                "rag_dataset_ingest_snapshots": 0,
                "rag_checkpoints": 0,
                "memory_checkpoints": 0,
                "group_memory_pending": 0,
                "rag_ingest_pending": 0,
                "stale_checkpoints": 0,
                "stale_active_leases": 0,
                "expired_active_leases": 0,
                "stalled": 0,
                "stagnant": 0,
                "muted": 1,
            },
            "pipelines": [
                {"target_id": "qq:1049511700:group:27234224"},
                {"target_id": "qq:1049511700:group:3219982"},
            ],
            "notes": [
                "side_effect=none",
                "group_level_knowledge_control_plane_view",
            ],
            "side_effect": "none",
        }
    }


def _runtime_overview_payload() -> dict:
    return {
        "data": {
            "cards": [
                {
                    "id": "knowledge_pipelines",
                    "label": "Knowledge Pipelines",
                    "value": "1/2",
                    "status": "warn",
                    "detail": {
                        "knowledge_pipelines": _runtime_direct_payload()["data"],
                    },
                }
            ]
        }
    }


def _dashboard_payload() -> dict:
    return {
        "cards": [
            {
                "id": "knowledge_pipelines",
                "label": "Knowledge Pipelines",
                "value": "1/2",
                "status": "warn",
                "detail": {
                    "knowledge_pipelines": _runtime_direct_payload()["data"],
                },
            }
        ],
        "knowledge_pipelines": _runtime_direct_payload()["data"],
    }


def test_dashboard_knowledge_rag_state_boundary_live_verified_shape() -> None:
    result = build_dashboard_knowledge_rag_state_boundary_verification(
        runtime_knowledge_pipelines=_runtime_direct_payload(),
        runtime_overview=_runtime_overview_payload(),
        dashboard_overview=_dashboard_payload(),
    )

    assert result["checks"]["runtime_endpoint_exposes_knowledge_pipeline_targets"] is True
    assert result["checks"]["runtime_endpoint_exposes_rag_boundary_fields"] is True
    assert result["checks"]["runtime_overview_exposes_knowledge_pipelines_card"] is True
    assert (
        result["checks"]["runtime_overview_card_matches_direct_knowledge_pipeline_state"]
        is True
    )
    assert result["checks"]["dashboard_overview_exposes_knowledge_pipelines_card"] is True
    assert (
        result["checks"]["dashboard_card_matches_runtime_knowledge_pipeline_state"]
        is True
    )
    assert (
        result["checks"]["dashboard_top_level_knowledge_pipelines_matches_runtime"]
        is True
    )
    assert result["checks"]["knowledge_pipeline_boundary_is_read_only"] is True


def test_dashboard_knowledge_rag_state_boundary_detects_missing_dashboard_card() -> None:
    dashboard_payload = {"cards": [], "knowledge_pipelines": {}}

    result = build_dashboard_knowledge_rag_state_boundary_verification(
        runtime_knowledge_pipelines=_runtime_direct_payload(),
        runtime_overview=_runtime_overview_payload(),
        dashboard_overview=dashboard_payload,
    )

    assert result["checks"]["dashboard_overview_exposes_knowledge_pipelines_card"] is False
    assert (
        result["checks"]["dashboard_card_matches_runtime_knowledge_pipeline_state"]
        is False
    )
    assert (
        result["checks"]["dashboard_top_level_knowledge_pipelines_matches_runtime"]
        is False
    )
