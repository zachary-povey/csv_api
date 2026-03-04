import os
import subprocess
import pytest
from pathlib import Path

from tests.utils import get_build_path


@pytest.fixture(scope="session")
def build_path() -> Path:
    """Returns the path to the csv_api binary.

    If the BUILD_PATH env var is set, returns that path directly. Otherwise,
    runs build.sh to produce a fresh build and returns the default artifact path.
    """
    if env_path := os.environ.get("BUILD_PATH"):
        return Path(env_path)

    build_script = Path(__file__).parent.parent / "scripts" / "build.sh"
    result = subprocess.run([str(build_script)], capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(f"Build failed:\n{result.stderr}")

    return get_build_path()
