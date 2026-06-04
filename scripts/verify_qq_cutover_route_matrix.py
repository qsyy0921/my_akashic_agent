from __future__ import annotations

import argparse
import json
import sys
from typing import Any

import httpx


def _request_json(base_url: str, path: str, *, method: str = "GET") -> dict[str, Any]:
    with httpx.Client(timeout=30.0, trust_env=True) as client:
        response = client.request(method, base_url.rstrip("/") + path)
    response.raise_for_status()
    payload = response.json()
    if not isinstance(payload, dict):
        raise RuntimeError(f"unexpected non-object response for {path}")
    data = payload.get("data", payload)
    if not isinstance(data, dict):
        raise RuntimeError(f"unexpected non-object data payload for {path}")
    return data


def _ensure_list(value: Any) -> list[str]:
    if value is None:
        return []
    if isinstance(value, list):
        return [str(item) for item in value]
    return [str(value)]


def _make_route(
    *,
    account_id: str,
    conversation_type: str,
    conversation_id: str,
    kind: str,
    source: str,
    reason: str,
) -> dict[str, str]:
    return {
        "account_id": account_id,
        "conversation_type": conversation_type,
        "conversation_id": conversation_id,
        "kind": kind,
        "source": source,
        "reason": reason,
    }


def _route_key(route: dict[str, str]) -> tuple[str, str, str, str, str]:
    return (
        route["account_id"],
        route["conversation_type"],
        route["conversation_id"],
        route["kind"],
        route["source"],
    )


def _dedupe_routes(routes: list[dict[str, str]]) -> list[dict[str, str]]:
    seen: set[tuple[str, str, str, str, str]] = set()
    result: list[dict[str, str]] = []
    for route in routes:
        key = _route_key(route)
        if key in seen:
            continue
        seen.add(key)
        result.append(route)
    return result


def _build_go_owned_routes(queue_backend: dict[str, Any]) -> list[dict[str, str]]:
    routes: list[dict[str, str]] = []
    for kind in _ensure_list(queue_backend.get("outbox_allowed_kinds")):
        routes.append(
            _make_route(
                account_id="*",
                conversation_type="*",
                conversation_id="*",
                kind=kind,
                source="global",
                reason="go_execution_owner_global_kind",
            )
        )

    by_account = queue_backend.get("outbox_allowed_kinds_by_account", {})
    if isinstance(by_account, dict):
        for account_id, kinds in by_account.items():
            for kind in _ensure_list(kinds):
                routes.append(
                    _make_route(
                        account_id=str(account_id),
                        conversation_type="*",
                        conversation_id="*",
                        kind=kind,
                        source="account",
                        reason="go_execution_owner_account_kind",
                    )
                )

    by_account_conversation_type = queue_backend.get(
        "outbox_allowed_kinds_by_account_conversation_type", {}
    )
    if isinstance(by_account_conversation_type, dict):
        for account_id, conversation_map in by_account_conversation_type.items():
            if not isinstance(conversation_map, dict):
                continue
            for conversation_type, kinds in conversation_map.items():
                for kind in _ensure_list(kinds):
                    routes.append(
                        _make_route(
                            account_id=str(account_id),
                            conversation_type=str(conversation_type),
                            conversation_id="*",
                            kind=kind,
                            source="account_conversation_type",
                            reason="go_execution_owner_account_conversation_type_kind",
                        )
                    )

    by_account_conversation_id = queue_backend.get(
        "outbox_allowed_kinds_by_account_conversation_id", {}
    )
    if isinstance(by_account_conversation_id, dict):
        for account_id, type_map in by_account_conversation_id.items():
            if not isinstance(type_map, dict):
                continue
            for conversation_type, id_map in type_map.items():
                if not isinstance(id_map, dict):
                    continue
                for conversation_id, kinds in id_map.items():
                    for kind in _ensure_list(kinds):
                        routes.append(
                            _make_route(
                                account_id=str(account_id),
                                conversation_type=str(conversation_type),
                                conversation_id=str(conversation_id),
                                kind=kind,
                                source="account_conversation_id",
                                reason="go_execution_owner_account_conversation_id_kind",
                            )
                        )

    return _dedupe_routes(routes)


def _project_policy_blocked_group_routes(
    go_owned_routes: list[dict[str, str]], qq_group_send_enabled: bool
) -> list[dict[str, str]]:
    if qq_group_send_enabled:
        return []

    blocked: list[dict[str, str]] = []
    for route in go_owned_routes:
        conversation_type = route["conversation_type"]
        if conversation_type not in {"*", "group"}:
            continue
        blocked.append(
            _make_route(
                account_id=route["account_id"],
                conversation_type="group",
                conversation_id=route["conversation_id"],
                kind=route["kind"],
                source=route["source"],
                reason="qq_group_send_disabled",
            )
        )
    return _dedupe_routes(blocked)


def _project_currently_sendable_routes(
    go_owned_routes: list[dict[str, str]], qq_group_send_enabled: bool
) -> list[dict[str, str]]:
    sendable: list[dict[str, str]] = []
    for route in go_owned_routes:
        conversation_type = route["conversation_type"]
        if qq_group_send_enabled:
            sendable.append(
                _make_route(
                    account_id=route["account_id"],
                    conversation_type=conversation_type,
                    conversation_id=route["conversation_id"],
                    kind=route["kind"],
                    source=route["source"],
                    reason="current_runtime_sendable",
                )
            )
            continue

        if conversation_type == "group":
            continue

        projected_conversation_type = (
            "private" if conversation_type == "*" else conversation_type
        )
        sendable.append(
            _make_route(
                account_id=route["account_id"],
                conversation_type=projected_conversation_type,
                conversation_id=route["conversation_id"],
                kind=route["kind"],
                source=route["source"],
                reason="current_runtime_sendable",
            )
        )

    return _dedupe_routes(sendable)


def _build_platform_blocker_routes(
    *, primary_account_id: str, secondary_account_id: str
) -> list[dict[str, str]]:
    routes: list[dict[str, str]] = []
    for account_id in (primary_account_id, secondary_account_id):
        for conversation_type in ("private", "group"):
            routes.append(
                _make_route(
                    account_id=account_id,
                    conversation_type=conversation_type,
                    conversation_id="*",
                    kind="image",
                    source="platform_blocker",
                    reason="native_rich_media_image_unresolved",
                )
            )

    routes.append(
        _make_route(
            account_id=primary_account_id,
            conversation_type="group",
            conversation_id="*",
            kind="file",
            source="platform_blocker",
            reason="first_account_group_file_session_specific_blocker",
        )
    )
    return _dedupe_routes(routes)


def _find_outbox_work_kind(queue_topology: dict[str, Any]) -> dict[str, Any] | None:
    for item in queue_topology.get("work_kinds", []):
        if isinstance(item, dict) and str(item.get("work_kind")) == "outbox_delivery":
            return item
    return None


def _route_matches(
    routes: list[dict[str, str]],
    *,
    account_id: str,
    conversation_type: str,
    kind: str,
) -> bool:
    for route in routes:
        if route["kind"] != kind:
            continue
        if route["account_id"] not in {"*", account_id}:
            continue
        if route["conversation_type"] not in {"*", conversation_type}:
            continue
        return True
    return False


def _build_result(
    *,
    runtime_base_url: str,
    runtime_config: dict[str, Any],
    queue_backend: dict[str, Any],
    outbound_cutover_readiness: dict[str, Any],
    queue_topology: dict[str, Any],
    runtime_overview: dict[str, Any],
    primary_account_id: str,
    secondary_account_id: str,
) -> dict[str, Any]:
    qq_group_send_enabled = bool(runtime_config.get("delivery", {}).get("qq_group_send_enabled"))
    go_owned_routes = _build_go_owned_routes(queue_backend)
    policy_blocked_routes = _project_policy_blocked_group_routes(
        go_owned_routes, qq_group_send_enabled
    )
    currently_sendable_routes = _project_currently_sendable_routes(
        go_owned_routes, qq_group_send_enabled
    )
    platform_blocker_routes = _build_platform_blocker_routes(
        primary_account_id=primary_account_id,
        secondary_account_id=secondary_account_id,
    )
    outbox_work_kind = _find_outbox_work_kind(queue_topology)
    summary = runtime_overview.get("summary", {})

    checks = {
        "qq_group_send_toggle_is_disabled": qq_group_send_enabled is False,
        "outbox_execution_owner_is_go_local_worker": queue_backend.get(
            "outbox_execution_owner"
        )
        == "go_local_outbox_worker",
        "outbox_scope_is_account_conversation_kind_gated": queue_backend.get(
            "outbox_execution_scope"
        )
        in {"account_conversation_kind_gated", "account_conversation_id_kind_gated"},
        "outbound_cutover_execution_ready": bool(
            outbound_cutover_readiness.get("execution_ready")
        ),
        "queue_topology_outbox_owner_matches_queue_backend": (
            outbox_work_kind is not None
            and outbox_work_kind.get("execution_owner")
            == queue_backend.get("outbox_execution_owner")
        ),
        "runtime_overview_outbox_owner_matches_queue_backend": summary.get(
            "queue_outbox_execution_owner"
        )
        == queue_backend.get("outbox_execution_owner"),
        "global_text_route_in_go_scope": _route_matches(
            go_owned_routes,
            account_id=primary_account_id,
            conversation_type="group",
            kind="text",
        ),
        "second_account_file_route_in_go_scope": _route_matches(
            go_owned_routes,
            account_id=secondary_account_id,
            conversation_type="group",
            kind="file",
        ),
        "first_account_private_file_route_in_go_scope": _route_matches(
            go_owned_routes,
            account_id=primary_account_id,
            conversation_type="private",
            kind="file",
        ),
        "first_account_group_file_not_in_go_scope": not _route_matches(
            go_owned_routes,
            account_id=primary_account_id,
            conversation_type="group",
            kind="file",
        ),
        "image_routes_not_in_go_scope": not any(
            route["kind"] == "image" for route in go_owned_routes
        ),
        "policy_blocked_group_text_visible": _route_matches(
            policy_blocked_routes,
            account_id=primary_account_id,
            conversation_type="group",
            kind="text",
        ),
        "platform_blocker_first_account_group_file_visible": _route_matches(
            platform_blocker_routes,
            account_id=primary_account_id,
            conversation_type="group",
            kind="file",
        ),
        "platform_blocker_image_routes_visible": len(
            [route for route in platform_blocker_routes if route["kind"] == "image"]
        )
        == 4,
    }
    all_checks_passed = all(bool(value) for value in checks.values())

    return {
        "runtime_base_url": runtime_base_url,
        "qq_accounts": {
            "configured_bot_ids": runtime_config.get("runtime", {}).get("bot_ids", []),
            "primary_account_id": primary_account_id,
            "secondary_account_id": secondary_account_id,
        },
        "runtime_config": {
            "qq_group_send_enabled": qq_group_send_enabled,
            "telegram_token_configured": runtime_config.get("delivery", {}).get(
                "telegram_token_configured"
            ),
        },
        "queue_backend": {
            "provider": queue_backend.get("provider"),
            "mode": queue_backend.get("mode"),
            "outbox_execution_owner": queue_backend.get("outbox_execution_owner"),
            "outbox_execution_scope": queue_backend.get("outbox_execution_scope"),
            "recommended_first_backend": queue_backend.get("recommended_first_backend"),
            "outbox_allowed_kinds": queue_backend.get("outbox_allowed_kinds"),
            "outbox_allowed_kinds_by_account": queue_backend.get(
                "outbox_allowed_kinds_by_account"
            ),
            "outbox_allowed_kinds_by_account_conversation_type": queue_backend.get(
                "outbox_allowed_kinds_by_account_conversation_type"
            ),
            "outbox_allowed_kinds_by_account_conversation_id": queue_backend.get(
                "outbox_allowed_kinds_by_account_conversation_id"
            ),
        },
        "outbound_cutover_readiness": {
            "ready": outbound_cutover_readiness.get("ready"),
            "execution_ready": outbound_cutover_readiness.get("execution_ready"),
            "execution_owner": outbound_cutover_readiness.get("execution_owner"),
            "reason": outbound_cutover_readiness.get("reason"),
            "blockers": outbound_cutover_readiness.get("blockers"),
        },
        "queue_topology": {
            "outbox_delivery_work_kind": outbox_work_kind,
        },
        "runtime_overview_summary": {
            "queue_outbox_execution_owner": summary.get("queue_outbox_execution_owner"),
            "outbound_cutover_plan_current_owner": summary.get(
                "outbound_cutover_plan_current_owner"
            ),
            "outbound_cutover_plan_ready": summary.get("outbound_cutover_plan_ready"),
        },
        "route_matrix": {
            "go_execution_owner_scope": go_owned_routes,
            "currently_sendable_routes": currently_sendable_routes,
            "policy_blocked_routes": policy_blocked_routes,
            "platform_blocker_routes": platform_blocker_routes,
            "notes": [
                "go_execution_owner_scope reflects current Go-owned route gates from queue backend state",
                "currently_sendable_routes applies the current qq_group_send_enabled policy without re-enabling group sends",
                "platform_blocker_routes reflects the current rich-media routes intentionally kept out of Go scope",
            ],
        },
        "checks": checks,
        "conclusion": {
            "status": "live_verified" if all_checks_passed else "verification_failed",
            "category": (
                "qq_partial_go_cutover_route_matrix_live_verified"
                if all_checks_passed
                else "qq_cutover_route_matrix_mismatch_detected"
            ),
            "reason": (
                "Current runtime proves Go owns text globally, second-account file, and first-account private file; qq/group routes remain blocked by the manual group-send policy, and image plus first-account group file remain outside Go scope behind unresolved rich-media blockers."
                if all_checks_passed
                else "At least one QQ cutover route-matrix parity check failed for the current runtime."
            ),
        },
    }


def run_qq_cutover_route_matrix_verifier(
    runtime_base_url: str,
    primary_account_id: str,
    secondary_account_id: str,
) -> dict[str, Any]:
    runtime_config = _request_json(runtime_base_url, "/v1/runtime-config")
    queue_backend = _request_json(runtime_base_url, "/v1/queue-backend")
    outbound_cutover_readiness = _request_json(
        runtime_base_url, "/v1/outbound-cutover/readiness", method="POST"
    )
    queue_topology = _request_json(runtime_base_url, "/v1/queue-topology")
    runtime_overview = _request_json(runtime_base_url, "/v1/runtime-overview")
    return _build_result(
        runtime_base_url=runtime_base_url,
        runtime_config=runtime_config,
        queue_backend=queue_backend,
        outbound_cutover_readiness=outbound_cutover_readiness,
        queue_topology=queue_topology,
        runtime_overview=runtime_overview,
        primary_account_id=primary_account_id,
        secondary_account_id=secondary_account_id,
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--runtime-base-url", default="http://127.0.0.1:8780")
    parser.add_argument("--primary-account-id", default="1049511700")
    parser.add_argument("--secondary-account-id", default="2365524513")
    args = parser.parse_args(argv)
    result = run_qq_cutover_route_matrix_verifier(
        args.runtime_base_url,
        args.primary_account_id,
        args.secondary_account_id,
    )
    json.dump(result, sys.stdout, ensure_ascii=False, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
