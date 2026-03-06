# Task - FR-211 Gap Closure (Built-in Tools)

**Status**: Closed
**Depends on**: FR-202, FR-070

## Objetivo
Unificar la ejecucion de built-in tools bajo una pipeline unica y universal que aplique validacion, autorizacion, auditoria y ejecucion con contratos operacionales consistentes.

## Entregables
- Dispatcher unico para built-in tools con secuencia efectiva `validate -> authorize -> audit -> execute`.
- Contrato uniforme de errores para built-ins, sin divergencias por executor o caller.
- Auditoria consistente por invocacion de built-in tool.
- Tests de seguridad y operacion para `permission denied` y `malformed input`.

## Frontera con FR-202
FR-202 deja cerrados el registry, el lifecycle admin, la validacion de schema/payload y el enforcement base de `requiredPermissions` e `is_active`.
FR-211 no reabre ese alcance: toma esa base y la eleva a una pipeline universal para todos los built-in tools, eliminando rutas alternativas y endureciendo auditoria y contrato de errores.

## Built-ins y puntos de ejecucion en alcance
- Built-ins en alcance: `create_task`, `update_case`, `send_reply`, `get_lead`, `get_account`, `create_knowledge_item`, `update_knowledge_item`, `query_metrics`.
- Call sites actuales que deben quedar cubiertos por el dispatcher unico:
  - agentes que consumen built-ins desde `prospecting`, `kb` e `insights`;
  - cualquier otro caller futuro debe entrar por el mismo dispatcher, sin acceso operativo alternativo al executor.

## Pasos
1. Consolidar un dispatcher unico para built-ins que sea el unico entrypoint operativo permitido para validacion, autorizacion, auditoria y ejecucion.
2. Reencaminar todos los built-ins y call sites actuales al dispatcher unico, eliminando bypass funcionales o rutas paralelas.
3. Normalizar el contrato de error operativo para built-ins, de modo que input invalido, permiso denegado, tool inactiva y error interno expongan categorias consistentes al caller.
4. Asegurar auditoria uniforme por invocacion con evidencia suficiente de `tool_name`, actor, workspace, resultado y tipo de fallo cuando aplique.
5. Agregar tests de seguridad y robustez para denegacion de permisos y entrada malformada sobre built-ins representativos.

## Riesgos
- Rutas alternativas que sigan invocando executors sin pasar por la pipeline universal.
- Contratos de error inconsistentes entre built-ins, agents y futuros callers.
- Auditoria parcial que impida demostrar enforcement o diagnosticar fallos por herramienta.

## Criterios de cierre
- Todos los built-ins en alcance usan la misma pipeline operativa y no existen call sites activos que ejecuten built-ins fuera de ese dispatcher.
- La validacion, autorizacion y auditoria son consistentes para todos los built-ins en alcance.
- El caller recibe categorias de error uniformes para `permission denied`, `malformed input`, `tool inactive` y fallo interno.
- Cada invocacion de built-in genera auditoria consistente con identificacion de herramienta, actor/workspace y resultado.
- Los tests de seguridad para `permission denied` y `malformed input` estan en verde sobre la pipeline universal.

## Tests minimos
- Built-in con permisos insuficientes devuelve denegacion consistente y registra auditoria.
- Built-in con payload malformado o invalido falla antes del executor con contrato uniforme.
- Built-in inactiva no se ejecuta y devuelve error consistente.
- Los built-ins representativos de lectura y escritura pasan por el mismo dispatcher.
- No hay bypass en los call sites actuales de `prospecting`, `kb` e `insights`.

## Implementacion realizada
- Se introdujo una pipeline universal de built-ins con secuencia efectiva de validacion, autorizacion, auditoria y ejecucion.
- Los built-ins en alcance quedaron reencaminados al dispatcher comun, sin bypass operativos en `prospecting`, `kb` e `insights`.
- Se normalizo el contrato de errores operativos para input invalido, permiso denegado, tool inactiva y fallo interno.
- La auditoria se emite de forma consistente por invocacion con contexto de herramienta, actor/workspace y resultado.

## Evidencia de cierre
- Implementacion integrada en `main` hasta el commit `283091b`.
- Validacion final por GitHub Actions en workflow `CI`.
- Run verde: `22770201280`.
- Resultado: quality gates, lint y tests en verde sobre `main`.
