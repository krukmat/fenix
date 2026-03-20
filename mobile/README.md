# Mobile (FenixCRM)

Aplicación mobile basada en Expo/React Native para consumir el BFF de FenixCRM.

## Requisitos

- Node.js 18+
- npm 9+
- Android Studio + Android SDK (API 35, NDK 26+) — para E2E tests
- Expo CLI: `npm install -g expo-cli eas-cli`
- Detox CLI: `npm install -g detox-cli` — para E2E tests

## Desarrollo local

Instalar dependencias:

```bash
npm install
```

Arrancar app:

```bash
npm run start
```

## Quality Gates (local)

Comandos principales:

```bash
npm run typecheck
npm run lint
npm run quality:arch
npm run test:coverage
```

Pipeline agregado:

```bash
npm run quality
```

## E2E Tests (Detox)

Los tests E2E validan flujos críticos de usuario en un emulador Android real.

### Prerrequisitos

1. Android Studio con AVD configurado (Pixel 6 API 35 recomendado)
2. Emulador Android corriendo (`adb devices` debe mostrar un dispositivo)
3. Backend Go corriendo en `localhost:8080`
4. BFF corriendo en `localhost:3000`

### Ejecutar tests E2E

```bash
# 1. Construir APK de prueba
npm run e2e:build

# 2. Ejecutar tests E2E
npm run e2e:test
```

### Suites E2E disponibles

| Archivo | Flujo validado |
|---------|----------------|
| `e2e/auth.e2e.ts` | Login → Register → Accounts list → Logout |
| `e2e/accounts.e2e.ts` | Accounts list → Detail → Timeline → Agent Activity |
| `e2e/deals.e2e.ts` | Deal detail → Agent Activity section → navegación a run detail |
| `e2e/cases.e2e.ts` | Case detail → Agent Activity section → navegación a run detail |
| `e2e/workflows.e2e.ts` | Workflows list → Create draft → Edit → Version actions |
| `e2e/copilot.e2e.ts` | Cases list → Case detail → Copilot panel → SSE response → Evidence cards |
| `e2e/agent-runs.e2e.ts` | Activity Log → Trigger agent → Run detail → Rejected run smoke |

### testIDs requeridos

Los tests E2E dependen de `testID` props en los componentes. Ver `docs/tasks/task_4.8.md` para la lista completa de testIDs usados.

## Definición de Done para PR (DoD)

Todo PR en `mobile/` debe cumplir:

1. `npm run quality` en verde.
2. No introducir regresiones de calidad:
   - nuevas violaciones de lint/arquitectura,
   - nuevas URLs hardcodeadas fuera de config/env,
   - nuevos anti-patrones de listas o aislamiento de query keys.
3. Agregar/ajustar tests para código nuevo o modificado en hooks/stores/services/componentes críticos.

## Política de subida de thresholds

Los thresholds de cobertura se elevan gradualmente por sprint cuando el pipeline se mantiene estable.

Estado actual de referencia (logic coverage global):

- statements: **35**
- branches: **25**
- lines: **35**
- functions: **20**

Subida sugerida:

1. subir 10–15 puntos por sprint en statements/lines si no hay flakiness;
2. subir branches/functions de forma más conservadora;
3. nunca bajar thresholds salvo incidente documentado.

## Notas

- `quality:arch` ejecuta `mobile/scripts/quality-check.mjs`.
- Los checks arquitectónicos están orientados a prevenir regresiones tempranas en PRs.
