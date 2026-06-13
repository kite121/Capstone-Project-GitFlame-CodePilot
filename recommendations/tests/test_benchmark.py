from experiments.run_benchmark import match_findings


def test_match_findings_uses_category_file_and_line_tolerance():
    predicted = [
        {"category": "security", "file": "a.py", "line": 12},
        {"category": "performance", "file": "a.py", "line": 30},
    ]
    expected = [
        {"category": "security", "file": "a.py", "line": 10},
        {"category": "architecture", "file": "b.py", "line": 1},
    ]

    score = match_findings(predicted, expected)

    assert score["true_positives"] == 1
    assert score["false_positives"] == 1
    assert score["false_negatives"] == 1
    assert score["f1"] == 0.5

