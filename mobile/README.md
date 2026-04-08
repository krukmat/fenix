# Mobile (FenixCRM)

Aplicación mobile basada en Expo/React Native para consumir el BFF de FenixCRM.

## Arquitectura de navegación (wedge-first)

La app expone exactamente 5 tabs visibles:

| Tab | Ruta | Descripción |
|-----|------|-------------|
| Inbox | `/inbox` | Approvals pendientes, handoffs, signals |
| Support | `/support` | Casos → detalle → copilot → trigger agent |
| Sales | `/sales` | Accounts/Deals segmentados → detalle → sales brief |
| Activity | `/activity` | Log de runs con filter chips por public status |
| Governance | `/governance` | Usage events + quota states |

Las rutas legacy (`/home`, `/accounts`, `/deals`, `/cases`, `/copilot`) permanecen como redirects ocultos hacia sus destinos wedge.

## Requisitos

- Node.js 18+
- npm 9+
- Expo CLI: `npm install -g expo-cli eas-cli`
- Android Studio + Android SDK (API 35) — para Maestro visual audit

## Desarrollo local

```bash
npm install
npm run start
```

## Quality Gates (local)

```bash
npm run typecheck
npm run lint
npm run quality:arch
npm run test:coverage
```

Pipeline completo:

```bash
npm run quality
```

## Visual Audit (Maestro)

El audit visual usa Maestro sobre un emulador Android real con datos deterministas. El detalle del baseline visual activo se documenta en el `README` raíz del repositorio.

### Prerrequisitos

1. Android Studio con AVD configurado (Pixel 6 API 35 recomendado)
2. Emulador Android corriendo (`adb devices` debe mostrar un dispositivo)
3. Maestro CLI instalado: `curl -Ls "https://get.maestro.mobile.dev" | bash`
4. Backend Go corriendo en `localhost:8080`
5. BFF corriendo en `localhost:3000`

### Ejecutar

```bash
# Construir APK debug
npm run e2e:build

# Sembrar datos + correr flows Maestro
bash maestro/seed-and-run.sh
```

El script:
1. Registra/loguea el usuario de prueba (`e2e@fenixcrm.test`)
2. Crea fixtures via Go directo a SQLite
3. Instala la APK en el emulador
4. Corre `maestro/visual-audit.yaml` y guarda screenshots en `artifacts/screenshots/`

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

## Notas

- `quality:arch` ejecuta `mobile/scripts/quality-check.mjs`.
- Los checks arquitectónicos están orientados a prevenir regresiones tempranas en PRs.
- El directorio `e2e/` contiene los helpers de seed para el audit Maestro.
