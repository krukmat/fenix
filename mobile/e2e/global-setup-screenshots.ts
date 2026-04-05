// Screenshot suite: custom globalSetup that pre-warms the app before Detox takes control.
// The AndroidX MonitoringInstrumentation has a hardcoded 45s timeout for activity launch.
// On a cold emulator, React Native + Metro bundle load takes 2-3 minutes.
// This setup launches the app via adb BEFORE Detox starts, waits for ReactNativeJS to log
// "Running main", then proceeds — so when Detox calls launchApp(), the app is already warm
// and the relaunch takes < 5s (ART odex cache is hot).

import { execFileSync } from 'node:child_process';

const ADB = process.env.ANDROID_HOME
  ? `${process.env.ANDROID_HOME}/platform-tools/adb`
  : 'adb';

type DetoxGlobalSetup = () => Promise<void>;

function adb(...args: string[]): string {
  try {
    return execFileSync(ADB, args, { encoding: 'utf8', stdio: ['pipe', 'pipe', 'pipe'] });
  } catch {
    return '';
  }
}

function isAppRunning(): boolean {
  const result = adb('shell', 'pidof', 'com.fenixcrm.app');
  return result.trim().length > 0;
}

function isReactNativeReady(): boolean {
  const logs = adb('logcat', '-d', '-s', 'ReactNativeJS');
  return logs.includes('Running "main"');
}

async function waitForAppReady(timeoutMs = 180000): Promise<void> {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    if (isReactNativeReady()) return;
    await new Promise(r => setTimeout(r, 3000));
  }
  throw new Error('App did not reach ready state within timeout');
}

async function runDetoxGlobalSetup(): Promise<void> {
  const { default: detoxGlobalSetup } = await import('detox/runners/jest/globalSetup');
  await (detoxGlobalSetup as DetoxGlobalSetup)();
}

module.exports = async () => {
  // Run Detox own globalSetup first (initializes the Detox server)
  await runDetoxGlobalSetup();

  // Clear stale logcat so we only detect fresh ReactNativeJS logs
  adb('logcat', '-c');

  if (!isAppRunning()) {
    console.log('[pre-warm] Launching app via adb...');
    adb('shell', 'am', 'start', '-n', 'com.fenixcrm.app/.MainActivity');
  } else {
    console.log('[pre-warm] App already running, waiting for ReactNativeJS ready...');
  }

  console.log('[pre-warm] Waiting for React Native bundle to load (up to 3 min)...');
  await waitForAppReady(180000);
  console.log('[pre-warm] App is ready. Handing off to Detox.');
};
