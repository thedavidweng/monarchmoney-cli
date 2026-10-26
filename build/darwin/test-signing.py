import os
from pathlib import Path
import subprocess
import tempfile
import unittest

SCRIPT = Path(__file__).with_name('sign-and-notarize.sh').resolve()


class SigningTest(unittest.TestCase):
    def run_signing(self, snapshot='false', target_os='darwin', **overrides):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            target = root / 'cli'
            target.write_text('release bytes')
            mock = '''#!/bin/bash
printf '%s\\n' "$(basename "$0") $*" >> "$CALL_LOG"
case "$(basename "$0") $1" in
  'codesign --verify')
    if [[ "$*" == *--check-notarization* ]]; then exit "$ASSESS_EXIT"; fi
    exit "$VERIFY_EXIT" ;;
  'codesign --display') printf '%s\\n' "Authority=$AUTHORITY" "TeamIdentifier=$TEAM" 'Timestamp=Oct 16, 2026' 'CodeDirectory flags=0x10000(runtime)' >&2 ;;
  'xcrun notarytool') printf '{"status":"%s"}\\n' "$NOTARY_STATUS" ;;
esac
'''
            for tool in ['codesign', 'xcrun', 'ditto']:
                path = root / tool
                path.write_text(mock)
                path.chmod(0o755)
            env = dict(os.environ, PATH=f'{root}:{os.environ["PATH"]}',
                       CALL_LOG=str(root / 'calls'), APPLE_TEAM_ID='TESTTEAM',
                       APPLE_SIGNING_IDENTITY='Developer ID Application: Test (TESTTEAM)',
                       APPLE_SIGNING_KEYCHAIN=str(root / 'signing.keychain-db'),
                       APPLE_API_KEY_ID='TESTKEY', APPLE_API_ISSUER='test-issuer',
                       APPLE_API_KEY_PATH=str(root / 'key.p8'), TEAM='TESTTEAM',
                       AUTHORITY='Developer ID Application: Test (TESTTEAM)',
                       VERIFY_EXIT='0', ASSESS_EXIT='0',
                       NOTARY_STATUS='Accepted')
            env.update(overrides)
            result = subprocess.run(['bash', str(SCRIPT), str(target), snapshot, target_os],
                                    env=env, text=True, capture_output=True)
            calls = root / 'calls'
            return result, calls.read_text() if calls.exists() else ''

    def test_release_artifact(self):
        result, calls = self.run_signing()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('--timestamp', calls)
        self.assertIn('--options runtime', calls)
        self.assertIn('--keychain', calls)
        self.assertLess(calls.index('codesign --verify'), calls.index('notarytool submit'))
        self.assertIn('--check-notarization -R =notarized', calls)

    def test_invalid_signatures_or_credentials_stop_submission(self):
        for invalid in [dict(TEAM='WRONGTEAM'), dict(AUTHORITY='Apple Distribution: Test'),
                        dict(VERIFY_EXIT='1'), dict(APPLE_API_KEY_ID='')]:
            with self.subTest(invalid=invalid):
                result, calls = self.run_signing(**invalid)
                self.assertNotEqual(result.returncode, 0)
                self.assertNotIn('notarytool submit', calls)

    def test_rejected_submission_fails_release(self):
        result, calls = self.run_signing(NOTARY_STATUS='Invalid')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('notarytool submit', calls)
        self.assertNotIn('--check-notarization', calls)

    def test_failed_assessment_fails_release(self):
        result, calls = self.run_signing(ASSESS_EXIT='1')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('notarytool submit', calls)

    def test_other_platforms_do_not_use_signing_credentials(self):
        for target_os in ['linux', 'windows']:
            result, calls = self.run_signing(target_os=target_os, APPLE_API_KEY_ID='')
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(calls, '')

    def test_snapshot_does_not_use_signing_credentials(self):
        result, calls = self.run_signing(snapshot='true', APPLE_API_KEY_ID='')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(calls, '')


if __name__ == '__main__':
    unittest.main()
