import pytest

from agent_engine.errors import InvalidPlanError
from agent_engine.plan_validator import PlanValidator
from tests.test_agent_api import valid_plan


def test_plan_validator_accepts_exact_contract_and_existing_paths():
    result = PlanValidator().validate(valid_plan(), {"src/auth.py"})

    assert result.markdown.startswith("# Implementation Plan")
    assert result.relevant_files[0].path == "src/auth.py"


def test_plan_validator_allows_explicit_new_file():
    plan = valid_plan().replace(
        "- `src/auth.py`: Contains token validation behavior.",
        "- `src/auth.py`: Contains token validation behavior.\n"
        "- `tests/test_auth.py` (create): Adds regression coverage.",
    )
    result = PlanValidator().validate(plan, {"src/auth.py"})

    assert result.relevant_files[-1].create is True


def test_plan_validator_rejects_hallucinated_existing_file():
    with pytest.raises(InvalidPlanError, match="unavailable repository file"):
        PlanValidator().validate(valid_plan("src/missing.py"), {"src/auth.py"})


def test_plan_validator_rejects_wrong_heading_order():
    plan = valid_plan().replace("## Goal", "## Unexpected")
    with pytest.raises(InvalidPlanError, match="headings"):
        PlanValidator().validate(plan, {"src/auth.py"})


def test_plan_validator_rejects_unsafe_new_file():
    plan = valid_plan().replace(
        "- `src/auth.py`: Contains token validation behavior.",
        "- `../secrets.txt` (create): Unsafe path.",
    )
    with pytest.raises(InvalidPlanError, match="unsafe repository path"):
        PlanValidator().validate(plan, {"src/auth.py"})
