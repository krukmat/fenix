# Mobile (FenixCRM)

Aplicación mobile basada en Expo/React Native para consumir el BFF de FenixCRM.

## Requisitos

- Node.js 18+
- npm 9+

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
