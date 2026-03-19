import subprocess
import tempfile
import fastavro
from dataclasses import dataclass, field
from pathlib import Path


FIXTURES_DIR = Path(__file__).parent / "fixtures"


@dataclass
class FixtureResult:
    """Result of running csv_api against a named fixture.

    Attributes:
        returncode: Exit code of the csv_api process.
        stdout: Standard output from the process.
        stderr: Standard error from the process.
        records: Parsed Avro records, populated only when returncode is 0.
    """

    returncode: int
    stdout: str
    stderr: str
    records: list[dict] = field(default_factory=list)


def get_build_path() -> Path:
    """Returns the path to the csv_api binary."""
    return Path(__file__).parent.parent / "build" / "csv-api"


def run_csv_api(
    build_path: Path, config_path, data_path, output_path
) -> subprocess.CompletedProcess:
    """Run the csv_api binary with given parameters.

    Args:
        build_path: Path to the csv_api binary.
        config_path: Path to the YAML config file.
        data_path: Path to the CSV input file.
        output_path: Path to write the Avro output file.

    Returns:
        The completed process result.
    """
    cmd = [
        str(build_path),
        "parse",
        "--config_path",
        str(config_path),
        "--data_path",
        str(data_path),
        "--output_path",
        str(output_path),
    ]
    return subprocess.run(cmd, capture_output=True, text=True)


def run_fixture(build_path: Path, fixture_name: str) -> FixtureResult:
    """Run csv_api against a named fixture and return the result.

    Looks up config.yaml and data.csv from tests/fixtures/<fixture_name>/,
    runs the binary, and automatically reads the Avro output on success.

    Args:
        build_path: Path to the csv_api binary.
        fixture_name: Name of the subdirectory under tests/fixtures/.

    Returns:
        FixtureResult with returncode, stdout, stderr, and records (if successful).

    Raises:
        FileNotFoundError: If the fixture directory or required files are missing.
    """
    fixture_dir = FIXTURES_DIR / fixture_name
    config_path = fixture_dir / "config.yaml"
    data_path = fixture_dir / "data.csv"

    if not fixture_dir.is_dir():
        raise FileNotFoundError(f"Fixture directory not found: {fixture_dir}")
    if not config_path.exists():
        raise FileNotFoundError(f"Fixture config not found: {config_path}")
    if not data_path.exists():
        raise FileNotFoundError(f"Fixture data not found: {data_path}")

    with tempfile.TemporaryDirectory() as tmpdir:
        output_path = Path(tmpdir) / "output.avro"
        proc = run_csv_api(build_path, config_path, data_path, output_path)
        records = (
            read_avro_file(output_path)
            if proc.returncode == 0 and output_path.exists()
            else []
        )

    return FixtureResult(
        returncode=proc.returncode,
        stdout=proc.stdout,
        stderr=proc.stderr,
        records=records,
    )


def read_avro_file(file_path) -> list[dict]:
    """Read and return records from an Avro file.

    Args:
        file_path: Path to the Avro file.

    Returns:
        List of records as dicts.
    """
    records = []
    with open(file_path, "rb") as f:
        reader = fastavro.reader(f)
        for record in reader:
            records.append(record)
    return records
