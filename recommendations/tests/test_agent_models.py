from agent_engine.models import parse_configuration


def test_nested_gitflame_yaml_is_adapted_to_plan_configuration():
    config = parse_configuration(
        """
analysis:
  include: [src/**]
  exclude: [src/generated/**]
rag:
  max_files: 12
  max_snippets_per_file: 4
"""
    )

    assert config.include == ["src/**"]
    assert config.exclude == ["src/generated/**"]
    assert config.max_files == 12
    assert config.max_snippets_per_file == 4
