// Task 4.8 — E2E Detox configuration for Android emulator
/** @type {Detox.DetoxConfig} */
module.exports = {
  testRunner: {
    args: {
      $0: 'jest',
      config: 'e2e/jest.config.ts',
    },
    jest: {
      setupTimeout: 300000,
    },
  },
  behavior: {
    launchApp: 'auto',
  },
  apps: {
    'android.debug': {
      type: 'android.apk',
      binaryPath: 'android/app/build/outputs/apk/debug/app-debug.apk',
      build: 'cp .env.e2e .env && npx expo prebuild --platform android --clean && cd android && ./gradlew assembleDebug assembleAndroidTest -DtestBuildType=debug --init-script ../init.gradle',
      reversePorts: [3000, 8080], // BFF on 3000, Go backend on 8080
    },
  },
  devices: {
    simulator: {
      type: 'android.emulator',
      device: {
        avdName: 'Pixel_7_API_33', // Task 4.8 — AVD disponible en esta máquina
      },
    },
  },
  configurations: {
    'android.emu.debug': {
      device: 'simulator',
      app: 'android.debug',
    },
    'android.emu.debug.screenshots': {
      device: 'simulator',
      app: 'android.debug',
      testRunner: {
        args: {
          config: 'e2e/jest.screenshots.config.ts',
        },
      },
      artifacts: {
        rootDir: './artifacts/screenshots',
        plugins: {
          screenshot: {
            shouldTakeAutomaticScreenshots: false,
            keepOnlyFailedTestsArtifacts: false,
          },
          log: 'none',
          video: 'none',
          timeline: 'none',
        },
      },
    },
  },
};
