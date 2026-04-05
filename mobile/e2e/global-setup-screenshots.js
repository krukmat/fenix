// Screenshot suite: custom globalSetup that pre-warms the app before Detox takes control.
// The AndroidX MonitoringInstrumentation has a hardcoded 45s timeout for activity launch.
// On a cold emulator, React Native + Metro bundle load takes 2-3 minutes.
// This setup launches the app via adb BEFORE Detox starts, waits for ReactNativeJS to log
// "Running main", then hands off to Detox — so the ART odex cache is hot and relaunch is < 5s.

const { execFileSync } = require('node:child_process');

const ADB = process.env.ANDROID_HOME
  ? `${process.env.ANDROID_HOME}/platform-tools/adb`
  : 'adb';

function adb(...args) {
  try {
    return execFileSync(ADB, args, { encoding: 'utf8', stdio: ['pipe', 'pipe', 'pipe'] });
  } catch {
    return '';
  }
}

function isAppRunning() {
  return adb('shell', 'pidof', 'com.fenixcrm.app').trim().length > 0;
}

function isReactNativeReady() {
  return adb('logcat', '-d', '-s', 'ReactNativeJS').includes('Running "main"');
}

async function waitForAppReady(timeoutMs = 180000) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (isReactNativeReady()) return;
    await new Promise(r => setTimeout(r, 3000));
  }
  throw new Error('[pre-warm] App did not reach ready state within timeout');
}

module.exports = async () => {
  // Run Detox own globalSetup first (initializes the Detox server + installs APKs)
  // This may kill the app during APK install — we handle that below
  await require('detox/runners/jest/globalSetup')();

  // After Detox installs APKs (which kills the running app), relaunch and wait for RN ready.
  // This ensures the ART odex cache is hot when Detox calls launchApp() in the first test.
  adb('logcat', '-c');

  if (!isAppRunning()) {
    console.log('[pre-warm] Launching app after APK install...');
    adb('shell', 'am', 'start', '-n', 'com.fenixcrm.app/.MainActivity');
  } else {
    console.log('[pre-warm] App still running after install.');
  }

  console.log('[pre-warm] Waiting for React Native bundle to load (up to 3 min)...');
  await waitForAppReady(180000);
  console.log('[pre-warm] App is ready. Handing off to Detox.');
};
