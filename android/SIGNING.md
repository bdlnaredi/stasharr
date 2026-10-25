# App signing

The Android app is signed with a private key kept in this repository's
GitHub secrets (Settings → Secrets and variables → Actions):

| Secret | What it holds |
|--------|---------------|
| `ANDROID_KEYSTORE_BASE64` | The keystore file (`.jks`), base64 encoded |
| `ANDROID_KEYSTORE_PASSWORD` | The keystore password |
| `ANDROID_KEY_ALIAS` | The key's alias (optional, defaults to `stasharr`) |
| `ANDROID_KEY_PASSWORD` | The key's password (optional, defaults to the keystore password) |

Keep a copy of the keystore and its password somewhere safe. Every update
must be signed with the same key, or Android refuses to install it over the
existing app.

Builds made without these secrets, for example in Android Studio, are signed
with the debug key. They work, but cannot be installed over the released app.
