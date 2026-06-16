import pytest

from recommendation_service.config import ConfigError, filter_repo_context, parse_config
from recommendation_service.models import RepoFile


def test_config_filters_paths_and_file_sizes(config_yaml):
    config = parse_config(config_yaml)
    files = [
        RepoFile(path="src/z.py", content="z = 1"),
        RepoFile(path="docs/readme.md", content="ignored"),
        RepoFile(path="src/generated/a.py", content="ignored"),
        RepoFile(path="src/large.py", content="x" * (21 * 1024)),
        RepoFile(path="src/a.py", content="a = 1"),
    ]

    selected = filter_repo_context(files, config)

    assert [file.path for file in selected] == ["src/a.py", "src/z.py"]


def test_config_rejects_model_selection(config_yaml):
    with pytest.raises(ConfigError, match="server-controlled"):
        parse_config(config_yaml + "\nrecommendation_model: qwen2.5-coder:7b\n")


def test_default_include_matches_root_files():
    config = parse_config("version: 1\n")

    selected = filter_repo_context([RepoFile(path="README.md", content="# Repository")], config)

    assert [file.path for file in selected] == ["README.md"]


def test_config_rejects_disabled_recommendations(config_yaml):
    disabled = config_yaml.replace(
        "recommendations:\n  enabled: true",
        "recommendations:\n  enabled: false",
    )

    with pytest.raises(ConfigError, match="recommendations are disabled"):
        parse_config(disabled)
