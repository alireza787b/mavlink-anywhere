import importlib.util
import sys
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location("mavlink_config", ROOT / "scripts" / "mavlink_config.py")
CONFIG = importlib.util.module_from_spec(SPEC)
assert SPEC.loader is not None
sys.modules[SPEC.name] = CONFIG
SPEC.loader.exec_module(CONFIG)


BASE = """[General]
TcpServerPort=5760
ReportStats=false

[UartEndpoint uart]
Device=/dev/serial0
Baud=57600

[UdpEndpoint gcs_listen]
Mode=server
Address=0.0.0.0
Port=14550

[UdpEndpoint mavsdk]
Mode=normal
Address=127.0.0.1
Port=14540
"""


class ConfigToolTests(unittest.TestCase):
    def test_disabled_endpoint_round_trip(self):
        disabled = CONFIG.disabled_block(
            "mavsdk",
            "[UdpEndpoint mavsdk]\nMode=normal\nAddress=127.0.0.1\nPort=14540\n\n",
        )
        raw = BASE.replace(
            "[UdpEndpoint mavsdk]\nMode=normal\nAddress=127.0.0.1\nPort=14540\n",
            disabled,
        )
        blocks = CONFIG.validate(raw)
        mavsdk = next(block for block in blocks if block.name == "mavsdk")
        self.assertFalse(mavsdk.enabled)
        self.assertEqual(mavsdk.values["Port"], "14540")

    def test_conflicting_server_bind_is_rejected(self):
        raw = BASE + "\n[UdpEndpoint duplicate]\nMode=server\nAddress=127.0.0.1\nPort=14550\n"
        with self.assertRaisesRegex(CONFIG.ConfigError, "both bind"):
            CONFIG.validate(raw)

    def test_mutation_syncs_env_and_keeps_config_valid(self):
        with tempfile.TemporaryDirectory() as directory:
            config_path = Path(directory) / "main.conf"
            env_path = Path(directory) / "router.env"
            config_path.write_text(BASE, encoding="utf-8")

            def add_qgc(text):
                return text.rstrip() + "\n\n" + CONFIG.udp_block("qgc", "192.168.1.50", 14550, "normal")

            CONFIG.mutate_config(config_path, env_path, add_qgc)
            CONFIG.validate(config_path.read_text(encoding="utf-8"))
            self.assertIn("192.168.1.50:14550", env_path.read_text(encoding="utf-8"))

    def test_endpoint_edit_preserves_neighboring_comments(self):
        raw = BASE.replace(
            "[UdpEndpoint mavsdk]",
            "# keep this route note\n[UdpEndpoint mavsdk]",
        ) + "\n# keep this trailing note\n"
        _, blocks = CONFIG.parse_config(raw)
        mavsdk = next(block for block in blocks if block.name == "mavsdk")
        updated = CONFIG.replace_range(
            raw,
            mavsdk,
            CONFIG.udp_block("mavsdk", "127.0.0.1", 14541, "normal"),
        )
        self.assertIn("# keep this route note", updated)
        self.assertIn("# keep this trailing note", updated)


if __name__ == "__main__":
    unittest.main()
