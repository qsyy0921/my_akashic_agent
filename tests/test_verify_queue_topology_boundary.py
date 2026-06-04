from __future__ import annotations

import importlib.util
import sys
from pathlib import Path


SCRIPT_PATH = (
    Path(__file__).resolve().parents[1]
    / "scripts"
    / "verify_queue_topology_boundary.py"
)


def _load_module():
    spec = importlib.util.spec_from_file_location(
        "verify_queue_topology_boundary", SCRIPT_PATH
    )
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


def test_build_result_marks_live_verified_when_owners_match():
    module = _load_module()
    queue_backend = {
        "provider": "local",
        "mode": "local_state_store",
        "migration_phase": "local_only",
        "recommended_first_backend": "nats_jetstream",
        "outbox_execution_owner": "go_local_outbox_worker",
        "outbox_execution_scope": "account_conversation_kind_gated",
        "agent_job_execution_owner": "python_ai_worker_state_store_lease",
    }
    queue_topology = {
        "provider": "local",
        "mode": "local_state_store",
        "migration_phase": "local_only",
        "selected_provider": "local",
        "recommended_provider": "nats_jetstream",
        "external_lease_ready": False,
        "work_kinds": [
            {
                "work_kind": "outbox_delivery",
                "execution_owner": "go_local_outbox_worker",
                "ack_owner": "go_state_store_api",
            },
            {
                "work_kind": "agent_job",
                "execution_owner": "python_ai_worker_state_store_lease",
                "ack_owner": "go_state_store_api",
            },
        ],
    }
    runtime_overview = {
        "summary": {
            "queue_topology_outbox_execution_owner": "go_local_outbox_worker",
            "queue_topology_agent_job_execution_owner": "python_ai_worker_state_store_lease",
            "queue_topology_agent_job_ack_owner": "go_state_store_api",
            "queue_topology_external_lease_ready": False,
        }
    }

    result = module._build_result(
        runtime_base_url="http://127.0.0.1:8780",
        queue_backend=queue_backend,
        queue_topology=queue_topology,
        runtime_overview=runtime_overview,
    )

    assert all(result["checks"].values())
    assert result["conclusion"]["status"] == "live_verified"
    assert (
        result["conclusion"]["category"]
        == "queue_topology_boundary_live_verified_with_runtime_overview"
    )


def test_build_result_marks_failure_when_agent_job_ack_owner_mismatches():
    module = _load_module()
    queue_backend = {
        "provider": "local",
        "mode": "local_state_store",
        "migration_phase": "local_only",
        "recommended_first_backend": "nats_jetstream",
        "outbox_execution_owner": "go_local_outbox_worker",
        "outbox_execution_scope": "account_conversation_kind_gated",
        "agent_job_execution_owner": "python_ai_worker_state_store_lease",
    }
    queue_topology = {
        "provider": "local",
        "mode": "local_state_store",
        "migration_phase": "local_only",
        "selected_provider": "local",
        "recommended_provider": "nats_jetstream",
        "external_lease_ready": False,
        "work_kinds": [
            {
                "work_kind": "outbox_delivery",
                "execution_owner": "go_local_outbox_worker",
                "ack_owner": "go_state_store_api",
            },
            {
                "work_kind": "agent_job",
                "execution_owner": "python_ai_worker_state_store_lease",
                "ack_owner": "nats_external_lease_result_ack",
            },
        ],
    }
    runtime_overview = {
        "summary": {
            "queue_topology_outbox_execution_owner": "go_local_outbox_worker",
            "queue_topology_agent_job_execution_owner": "python_ai_worker_state_store_lease",
            "queue_topology_agent_job_ack_owner": "go_state_store_api",
            "queue_topology_external_lease_ready": False,
        }
    }

    result = module._build_result(
        runtime_base_url="http://127.0.0.1:8780",
        queue_backend=queue_backend,
        queue_topology=queue_topology,
        runtime_overview=runtime_overview,
    )

    assert result["checks"]["agent_job_ack_owner_matches_runtime_overview_summary"] is False
    assert result["conclusion"]["status"] == "verification_failed"
    assert (
        result["conclusion"]["category"]
        == "queue_topology_boundary_mismatch_detected"
    )
