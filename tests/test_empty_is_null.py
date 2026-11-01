"""Tests for empty cells on nullable fields: empty_is_null and
quoted_empty_is_null (TASK-23)."""

import datetime
import subprocess

import pytest
import yaml

from tests.utils import FIXTURES_DIR, run_fixture, run_validate_config


def test_unquoted_and_quoted_empty_cells_are_null_by_default(build_path):
    """TASK-23 AC#1: with the default flags, an unquoted empty cell (,,) of a
    nullable field is null. TASK-23 AC#2: so is a quoted empty cell (,"",).
    Covers several types, and the last field of a row."""
    result = run_fixture(build_path, "empty_is_null_default")
    assert result.returncode == 0, result.stderr
    empty = {"label": None, "count": None, "day": None, "active": None}
    assert result.records == [
        {"id": 1, **empty},  # all unquoted
        {"id": 2, **empty},  # all quoted
        {"id": 3, "label": "x", "count": 5, "day": datetime.date(2024, 1, 2), "active": True},
        {"id": 4, **empty},  # mixed
    ]


def test_quoted_empty_is_null_false_postgres_style(build_path):
    """TASK-23 AC#3: Postgres COPY ... CSV style. With quoted_empty_is_null:
    false and a ^$ representation, ,, is null and ,"", is an empty string."""
    result = run_fixture(build_path, "quoted_empty_is_not_null")
    assert result.returncode == 0, result.stderr
    assert [r["note"] for r in result.records] == [None, "", "hi"]


def test_empty_is_null_false_without_matching_representation_fails(build_path):
    """TASK-23 AC#4: with empty_is_null: false and no representation matching
    an empty value, ,, is a data failure (the quoted ,"", is still null, since
    quoted_empty_is_null keeps its default)."""
    result = run_fixture(build_path, "empty_is_not_null_fails", ["--fail-fast=false"])
    assert result.returncode != 0, "Command should have failed"
    assert not result.output_exists
    assert "row 3, column 'count': value '' did not match any pattern" in result.stderr
    assert "row 4" not in result.stderr

    dropped = run_fixture(build_path, "empty_is_not_null_fails", ["--error-on-data-failures=false"])
    assert dropped.returncode == 0, dropped.stderr
    assert dropped.records == [{"id": 1, "count": 5}, {"id": 3, "count": None}]


def test_is_null_representation_does_not_change_empty_cells(build_path):
    """TASK-23 AC#5: a nullable field with an is_null representation (^-$)
    that doesn't match empty values still gets null for empty cells with the
    default flags. Updated from TASK-7's decision-3 test (superseded by
    TASK-23), where the empty cell was a data failure."""
    result = run_fixture(build_path, "nullable_empty_cell_unmatched")
    assert result.returncode == 0, result.stderr
    assert result.records == [
        {"id": 1, "count": 5},
        {"id": 2, "count": None},
        {"id": 3, "count": None},
    ]


@pytest.mark.parametrize("flag", ["empty_is_null", "quoted_empty_is_null"])
def test_flag_on_non_nullable_field_fails_config_load(build_path, flag):
    """TASK-23 AC#6: either flag set explicitly on a field that is not nullable
    fails the config load, naming the field and the flag (validate_config and
    parse)."""
    expected = f"field 'label': {flag} is only allowed on a nullable field"
    fixture = f"config_{flag}_not_nullable"
    proc = run_validate_config(build_path, fixture)
    assert proc.returncode != 0
    assert expected in proc.stderr
    result = run_fixture(build_path, fixture)
    assert result.returncode != 0
    assert expected in result.stderr
    assert not result.output_exists


def test_empty_cells_on_non_nullable_fields_are_unchanged(build_path):
    """TASK-23 AC#7: on a non-nullable field an empty cell, quoted or not, is
    handled by the representations as before: an empty string for a string
    field with ^(.*)$, a data failure for an integer field."""
    result = run_fixture(build_path, "empty_cell_not_nullable", ["--error-on-data-failures=false"])
    assert result.returncode == 0, result.stderr
    assert result.records == [
        {"id": 1, "label": "", "count": 5},
        {"id": 2, "label": "", "count": 6},
    ]
    assert "dropped 2 row(s) with invalid data" in result.stderr

    failed = run_fixture(build_path, "empty_cell_not_nullable", ["--fail-fast=false"])
    assert failed.returncode != 0
    assert "row 4, column 'count': value '' did not match any pattern" in failed.stderr
    assert "row 5, column 'count': value '' did not match any pattern" in failed.stderr


def test_top_level_flags_are_defaults_and_fields_override_them(build_path):
    """TASK-23 AC#8: top-level empty_is_null / quoted_empty_is_null apply to a
    nullable field that doesn't set them, a per-field value overrides the
    top-level one, and the top-level settings don't make the non-nullable id
    field fail the load."""
    assert run_validate_config(build_path, "empty_is_null_top_level").returncode == 0
    result = run_fixture(build_path, "empty_is_null_top_level")
    assert result.returncode == 0, result.stderr
    assert result.records == [
        {"id": 1, "inherits": "", "overrides": None},
        {"id": 2, "inherits": "", "overrides": ""},
        {"id": 3, "inherits": "a", "overrides": "b"},
    ]


def test_layered_config_overrides_flags(build_path):
    """TASK-23 AC#9: an extending config's top-level quoted_empty_is_null and
    per-field empty_is_null are used in the merged config (the root alone makes
    every empty cell null), and --print-merged shows them."""
    result = run_fixture(build_path, "layered_empty_is_null")
    assert result.returncode == 0, result.stderr
    assert result.records == [
        {"id": 1, "a": None, "b": ""},
        {"id": 2, "a": "", "b": ""},
    ]

    config_path = FIXTURES_DIR / "layered_empty_is_null" / "config.yaml"
    proc = subprocess.run(
        [str(build_path), "validate_config", "-c", str(config_path), "--print-merged"],
        capture_output=True,
        text=True,
        timeout=60,
    )
    assert proc.returncode == 0, proc.stderr
    merged = yaml.safe_load(proc.stdout)
    assert merged["quoted_empty_is_null"] is False
    assert "empty_is_null" not in merged
    fields = {f["name"]: f for f in merged["fields"]}
    assert fields["b"]["empty_is_null"] is False
    assert "empty_is_null" not in fields["a"]
    assert "quoted_empty_is_null" not in fields["id"]


def test_layered_flag_on_non_nullable_field_fails(build_path):
    """TASK-23 AC#6, AC#9: the nullable check runs on the merged config, so an
    extending file setting a flag on a non-nullable root field fails."""
    proc = run_validate_config(build_path, "layered_empty_is_null_not_nullable")
    assert proc.returncode != 0
    assert "field 'id': quoted_empty_is_null is only allowed on a nullable field" in proc.stderr


def test_quoted_and_unquoted_empty_cells_told_apart_in_any_position(build_path):
    """TASK-23 AC#10: with CRLF line endings, no trailing newline at end of
    file, and after a quoted field containing a newline, quoted and unquoted
    empty cells (including as the last field of a row) are told apart. The
    detection itself is unit tested in internal/reader (TestQuotedEmptyDetection)."""
    data = (FIXTURES_DIR / "empty_is_null_line_endings" / "data.csv").read_bytes()
    assert b"\r\n" in data and not data.endswith(b"\n")
    result = run_fixture(build_path, "empty_is_null_line_endings")
    assert result.returncode == 0, result.stderr
    assert result.records == [
        {"id": 1, "text": None, "last": ""},
        {"id": 2, "text": "", "last": None},
        {"id": 3, "text": "multi\nline", "last": ""},
        {"id": 4, "text": "multi\nline", "last": None},
        {"id": 5, "text": None, "last": ""},
    ]
