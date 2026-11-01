import subprocess
from pathlib import Path


def test_go_unit_tests():
    """Run the Go unit tests (go test ./...) as part of the suite, so ACs
    proven by Go tests are covered by `python3 -m pytest tests`."""
    repo = Path(__file__).parent.parent
    result = subprocess.run(
        ["go", "test", "./..."], cwd=repo, capture_output=True, text=True, timeout=300
    )
    assert result.returncode == 0, result.stdout + result.stderr
