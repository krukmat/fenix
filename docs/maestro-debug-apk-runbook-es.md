---
doc_type: summary
title: "Maestro debug APK runbook (ES)"
status: active
created: 2026-05-03
tags:
  - maestro
  - mobile
  - screenshots
  - android
  - runbook
task_refs:
  - SCR-FIX1
---

# Runbook: levantar Maestro con el APK debug real

## Objetivo

Dejar documentado el procedimiento operativo que funciono para correr `Maestro` contra el `app-debug.apk` real y diagnosticar el caso en que la app se queda en splash.

Comando canonico:

```bash
cd mobile && npm run screenshots
```

## Prerrequisitos

Antes de correr Maestro, el entorno local debe tener:

- emulador Android o device conectado por `adb`
- backend Go levantado en `localhost:8080`
- BFF levantado en `localhost:3000`
- APK debug compilado e instalado, o al menos compilable desde `mobile/android`
- Metro disponible en `localhost:8081`

## Sintoma principal observado

La app quedaba clavada en el splash al lanzar el APK debug desde el runner de screenshots.

El problema no era login, seed ni BFF. La causa raiz fue que el build debug intentaba cargar el bundle JS desde Metro y Metro no estaba levantado en `:8081`.

## Evidencia util de logs

Cuando el splash se debe a Metro caido, `adb logcat` muestra lineas de este estilo:

```text
Couldn't connect to "ws://10.0.2.2:8081/message..."
The packager does not seem to be running
Unable to load script
Make sure you're running Metro...
```

Tambien se puede confirmar que Android sigue reteniendo la splash window:

```bash
adb shell dumpsys activity activities | rg "com.fenixcrm.app|Splash"
adb shell dumpsys window windows | rg "Splash Screen com.fenixcrm.app|firstWindowDrawn"
```

## Pasos operativos que funcionaron

### 1. Verificar device Android

```bash
adb devices
```

Esperado: un emulador o device en estado `device`.

### 2. Levantar backend Go

Desde la raiz del repo:

```bash
JWT_SECRET='test-secret-32-chars-minimum!!!' go run ./cmd/fenix serve --port 8080
```

Check rapido:

```bash
curl -fsS http://127.0.0.1:8080/health
```

### 3. Levantar BFF

Levantar el BFF para que responda en `localhost:3000`.

Check rapido:

```bash
curl -sS --max-time 3 http://127.0.0.1:3000
```

Si el BFF no responde, `seed-and-run.sh` corta antes del seed.

### 4. Compilar el APK debug si hace falta

Si el APK debug no existe o quedo viejo, recompilarlo:

```bash
cd mobile
npx expo run:android
```

Alternativa directa sobre Gradle:

```bash
cd mobile/android
./gradlew assembleDebug
```

APK esperado:

```text
mobile/android/app/build/outputs/apk/debug/app-debug.apk
```

### 5. Instalar o reinstalar el APK debug

Si hace falta reinstalar manualmente:

```bash
adb install -r mobile/android/app/build/outputs/apk/debug/app-debug.apk
```

El runner tambien intenta instalarlo automaticamente si el archivo existe.

### 6. Levantar Metro en `8081`

Desde `mobile/`:

```bash
cd mobile
npx expo start --port 8081 --host localhost
```

Nota: `expo` aviso que `--non-interactive` no aplica aqui y recomendo `CI=1` si se necesita modo no interactivo.

### 7. Confirmar que Metro responde

```bash
curl -sS http://127.0.0.1:8081/status
```

Esperado:

```text
packager-status:running
```

Si devuelve `Connection refused`, el splash es esperable en el APK debug.

### 8. Preparar networking Android

El runner ya deja configurado:

```bash
adb reverse tcp:3000 tcp:3000
adb reverse tcp:8080 tcp:8080
adb reverse tcp:8081 tcp:8081
```

Aunque en emulador React Native suele resolver `10.0.2.2:8081`, mantener `adb reverse tcp:8081 tcp:8081` sigue siendo parte del setup correcto y evita problemas en device fisico.

### 9. Ejecutar el runner

```bash
cd mobile && npm run screenshots
```

En la corrida valida observada:

- el runner detecto Metro con `Using existing Metro server at http://127.0.0.1:8081/status`
- hizo seed correctamente
- avanzo hasta `Phase 1/2: capturing auth surface...`
- dejo de fallar por splash/packager

## Diagnostico rapido si vuelve a quedar en splash

### Check 1. Metro local

```bash
curl -sS --max-time 3 http://127.0.0.1:8081/status
```

Si falla, levantar Metro primero.

### Check 1b. Backend y BFF

```bash
curl -fsS http://127.0.0.1:8080/health
curl -sS --max-time 3 http://127.0.0.1:3000
```

Si alguno falla, el problema todavia es de infraestructura local, no del flow Maestro.

### Check 2. Logcat filtrado del proceso app

```bash
adb shell pidof com.fenixcrm.app
adb logcat --pid <PID> -d | rg "ReconnectingWebSocket|Unable to load script|packager|ReactNative|10.0.2.2:8081"
```

Si aparecen `Unable to load script` o `The packager does not seem to be running`, el problema sigue siendo bundler.

### Check 3. Estado visual Android

```bash
adb shell dumpsys activity activities | rg "com.fenixcrm.app|Splash"
adb shell dumpsys window windows | rg "Splash Screen com.fenixcrm.app|mTopFullscreenOpaqueWindowState"
```

Si la splash sigue como top window, React Native no logro dibujar la primera pantalla.

## Ajuste aplicado al runner

`mobile/maestro/seed-and-run.sh` quedo reforzado para:

- detectar si Metro ya esta vivo en `http://127.0.0.1:8081/status`
- levantar Metro si no existe
- esperar activamente a `packager-status:running`
- mantener `adb reverse tcp:8081 tcp:8081`

Eso reduce el riesgo de repetir el fallo al usar el APK debug real.

## Observacion posterior al fix

Una vez resuelto Metro, el siguiente sintoma observado ya no fue splash sino un ANR Android intermitente:

```text
Process system isn't responding
```

El flow `auth-surface.yaml` ya contiene un bloque para intentar cerrarlo con `Wait`. Si reaparece, el problema ya no es bundler sino estabilidad del emulador o carga de arranque.

## Comandos de referencia

```bash
adb devices
curl -fsS http://127.0.0.1:8080/health
curl -sS --max-time 3 http://127.0.0.1:3000
curl -sS http://127.0.0.1:8081/status
adb shell pidof com.fenixcrm.app
adb logcat --pid <PID> -d | rg "Unable to load script|packager|10.0.2.2:8081"
adb shell dumpsys activity activities | rg "com.fenixcrm.app|Splash"
JWT_SECRET='test-secret-32-chars-minimum!!!' go run ./cmd/fenix serve --port 8080
cd mobile && npx expo run:android
cd mobile/android && ./gradlew assembleDebug
cd mobile && npx expo start --port 8081 --host localhost
cd mobile && npm run screenshots
```

## Resultado esperado

Con Metro activo y el device accesible:

- la app debe salir del splash
- `auth-surface.yaml` debe poder esperar `login-screen`
- `npm run screenshots` debe avanzar al menos hasta la fase 1 sin fallar por packager
