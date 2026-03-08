# FR Gaps - Master Plan de Cierre

**Status**: En ejecucion  
**Fuente base**: `docs/fr-gaps-implementation-criteria.md`  
**Objetivo**: descomponer y ejecutar el cierre de gaps FR pendientes con trazabilidad, orden de dependencias y criterios de cierre verificables.

---

## 1) FRs en alcance

- FR-001
- FR-060
- FR-061
- FR-070
- FR-071
- FR-090
- FR-091
- FR-200
- FR-201
- FR-202
- FR-211
- FR-230
- FR-231
- FR-232
- FR-240

> Exclusion explicita: FR-052.

---

## 2) Orden de ejecucion acordado

1. **Governance/Auth**: FR-060, FR-061, FR-071, FR-202, FR-211
2. **AI runtime quality/safety**: FR-200, FR-201, FR-230, FR-231, FR-232
3. **Knowledge reliability**: FR-090, FR-091
4. **Prompt lifecycle**: FR-240
5. **CRM consistency hardening**: FR-001 (y endurecimiento transversal con FR-070)

---

## 3) Entregables de planificacion en `docs/tasks`

- `task_fr_060_gap_closure.md`
- `task_fr_061_gap_closure.md`
- `task_fr_071_gap_closure.md`
- `task_fr_202_gap_closure.md`
- `task_fr_211_gap_closure.md`
- `task_fr_200_gap_closure.md`
- `task_fr_201_gap_closure.md`
- `task_fr_230_gap_closure.md`
- `task_fr_231_gap_closure.md`
- `task_fr_232_gap_closure.md`
- `task_fr_090_gap_closure.md`
- `task_fr_091_gap_closure.md`
- `task_fr_240_gap_closure.md`
- `task_fr_001_gap_closure.md`
- `task_fr_070_gap_closure.md`

---

## 4) Estado actual consolidado

### Cerradas
- FR-060
- FR-061
- FR-071
- FR-090
- FR-091
- FR-200
- FR-201
- FR-202
- FR-211
- FR-230
- FR-231
- FR-232
- FR-240

### Pendientes
- FR-001
- FR-070

### Lectura por bloques
1. **Governance/Auth**
   - Cerradas: FR-060, FR-061, FR-071, FR-202, FR-211
   - Pendientes: ninguna
   - Siguiente tarea desbloqueada: ninguna
2. **AI runtime quality/safety**
   - Cerradas: FR-200, FR-201, FR-230, FR-231, FR-232
   - Pendientes: ninguna
3. **Knowledge reliability**
   - Cerradas: FR-090, FR-091
   - Pendientes: ninguna
4. **Prompt lifecycle**
   - Cerrada: FR-240
5. **CRM consistency hardening**
   - Pendientes: FR-001, FR-070

### Dependencias que marcan el siguiente paso
- FR-202 y FR-211 quedaron cerradas en `main` con validacion final de CI verde.
- FR-200, FR-201, FR-230, FR-231 y FR-232 quedaron cerradas en `main` con validacion final de CI verde.
- FR-090 y FR-091 quedaron cerradas en `main` con validacion final de CI verde.
- FR-240 quedo cerrada en `main` con validacion final de CI verde.
- FR-001 depende de FR-070.
- `FR-070` pasa a ser la siguiente tarea desbloqueada del frente CRM consistency hardening.

---

## 5) Definicion de DONE (global)

Un FR se considera cerrado solo si cumple todo:

1. Criterios de `fr-gaps-implementation-criteria.md` satisfechos.
2. Contrato API/documentacion actualizado (OpenAPI/docs tecnicas si aplica).
3. Tests requeridos (unit/integration/e2e) en verde.
4. Trazabilidad FR->tests->artefactos verificable.
5. Sin regresiones en quality gates relevantes.

---

## 6) Riesgos transversales

- Divergencia entre contrato documentado y runtime real.
- Decisiones ABAC/policy no deterministas entre paths distintos.
- Falta de observabilidad para demostrar SLA (<60s en FR-091).
- Inconsistencias entre rutas de tool execution (FR-202/211).

Mitigacion general: definicion de contrato primero, TDD por flujo critico, y cierre con pruebas de regresion por dominio.
