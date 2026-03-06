# Task - FR-202 Gap Closure (Tool Registry)

**Status**: Closed
**Depends on**: FR-060, FR-071

## Objetivo
Completar el Tool Registry con un lifecycle admin gestionado, validacion fuerte de schema y enforcement obligatorio de `requiredPermissions` e `is_active` antes de cualquier ejecucion de tools.

## Entregables
- Lifecycle admin completo para tool definitions: `list`, `create`, `update`, `activate`, `deactivate`, `delete`.
- Validacion fuerte de `inputSchema` al alta/edicion y validacion runtime del payload contra la definicion persistida.
- Enforcement obligatorio de `requiredPermissions` e `is_active` en todos los puntos actuales de ejecucion antes de invocar un executor.

## Pasos
1. Completar handlers, rutas y persistencia necesarios para `update`, `activate`, `deactivate` y `delete`, manteniendo el scope por workspace y el comportamiento admin existente de `list` y `create`.
2. Endurecer la validacion de `inputSchema` con un subset minimo aceptado y rechazos deterministas en create/update.
3. Validar el payload runtime contra el schema persistido antes de cualquier ejecucion real de la tool.
4. Cubrir explicitamente todos los call sites actuales de ejecucion para impedir bypass de `is_active`, validacion de params y permission gate antes del executor.

## Subset minimo de schema aceptado
- El root del schema debe declarar `type: "object"`.
- `properties` es obligatorio y debe ser consistente con `required`.
- `additionalProperties` debe estar presente para que la politica de campos extra sea explicita.
- Se rechazan schemas con JSON invalido, root no-objeto, claves `required` no definidas en `properties`, y estructuras vacias o debiles que no permitan validacion determinista.

## Riesgos
- Bypass por ejecucion directa de executors fuera del gate esperado.
- Drift entre schema persistido y validacion runtime del payload.
- Inconsistencias de lifecycle o enforcement sobre tools built-in o tools inactivas.

## Criterios de cierre
- Existen y estan probados los endpoints/operaciones admin de `update`, `activate`, `deactivate` y `delete`, ademas de `list` y `create`.
- Una tool inactiva no puede ejecutarse en ninguno de los call sites actuales.
- Un payload invalido respecto del schema persistido falla de forma determinista antes del executor.
- Una ejecucion sin `requiredPermissions` suficientes falla de forma determinista antes del executor.
- Los call sites actuales de ejecucion comparten las mismas precondiciones efectivas: `is_active`, validacion de params y permission gate.

## Tests minimos
- Create/update con schema invalido o schema debil.
- Activate/deactivate/delete con verificacion del estado resultante.
- Runtime con payload invalido contra schema persistido.
- Permission denied por falta de `requiredPermissions`.
- Intento de ejecucion sobre tool inactiva.
- Aislamiento por workspace en lifecycle y ejecucion.

## Frontera con FR-211
FR-202 exige enforcement consistente en los puntos de ejecucion existentes y deja cerrados registry, lifecycle, validacion y gates base.
La unificacion arquitectonica en una sola pipeline universal para built-in tools queda como continuidad natural de FR-211 y no como requisito de cierre de FR-202.

## Implementacion realizada
- Se completo el lifecycle admin del registry con `list`, `create`, `update`, `activate`, `deactivate` y `delete`.
- Se endurecio la validacion de `inputSchema` en altas y ediciones con el subset minimo definido por la task.
- La ejecucion runtime valida payload, `is_active` y `requiredPermissions` antes de cualquier executor.
- Los call sites operativos actuales quedaron alineados sobre el path comun de ejecucion del registry.

## Evidencia de cierre
- Implementacion integrada en `main` hasta el commit `283091b`.
- Validacion final por GitHub Actions en workflow `CI`.
- Run verde: `22770201280`.
- Resultado: quality gates, lint y tests en verde sobre `main`.
