# App signing key

`stasharr.jks` signs the Android app so each new version installs over the
previous one. It is kept here so GitHub can build the app without extra
setup.

- Key alias: `stasharr`
- Password: `jL_a9amq7rzz8EHkg0-fFL72Isug6oTJ`

Because this repository is public, anyone could sign an app with this key.
That only matters if someone gets you to install an APK from somewhere else:
only install Stasharr from this repository's Releases page.

To use a private key instead, add the repository secrets
`ANDROID_KEYSTORE_BASE64`, `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS`
and `ANDROID_KEY_PASSWORD`. The build uses them when they are set. You will
need to uninstall the app once before the first build signed with the new key
will install.
