---
doc_type: task
id: mobile_screenshots_runner_fix
title: Corregir runner Maestro screenshots y artefactos estables
status: completed
created: 2026-04-19
completed: 2026-04-19
---

## Context

`cd mobile && npm run screenshots` debe producir capturas finales en:

```text
mobile/artifacts/screenshots/
```

En la ejecucion del 2026-04-19, el comando llego a correr Maestro pero fallo en la fase autenticada al navegar a Support:

```text
Assertion is false: id: support-cases-list-item-0 is visible
```

El directorio estable `mobile/artifacts/screenshots/` quedo vacio porque el runner limpia el destino antes de ejecutar los flows y solo copia PNGs desde `REPORTS_DIR` al final exitoso. Cuando Maestro falla a mitad de camino, las capturas parciales y el screenshot de fallo quedan en el directorio temporal de reportes, no en el destino estable del repo.

Artefacto de fallo observado:

```text
/var/folders/1d/dtw017jx7l53n_b4hy8p75yc0000gn/T/fenixcrm-maestro-reports/2026-04-19_184135
```

Captura de fallo observada: la UI seguia en Inbox con el filtro Signals activo despues de `tapOn id: tab-support`. Por lo tanto, el fallo no prueba que Support no tenga casos; prueba que la navegacion por tab en ese punto del flow es fragil o no produjo el cambio de pantalla esperado.

## Problemas encontrados

1. **Contrato de ejecucion ambiguo**
   - `npm run screenshots` falla desde la raiz porque no hay `package.json` root.
   - El comando valido hoy es `cd mobile && npm run screenshots`.
   - Si se quiere soportar `npm run screenshots` desde la raiz, hay que agregar un wrapper root o documentar formalmente que el comando es mobile-only.

2. **Backend local no validado antes del seed**
   - El runner asume backend Go en `localhost:8080`.
   - Si no esta levantado, falla con `connect: connection refused`.
   - Si falta `JWT_SECRET`, el backend responde 500 en `/auth/login`.
   - El runner no hace preflight claro de `/health` ni de las variables requeridas.

3. **Artefactos no se copian en fallo**
   - `seed-and-run.sh` hace `rm -rf "${OUTPUT_DIR}" "${REPORTS_DIR}"` antes de los flows.
   - `copy_reports_screenshots` se ejecuta solo despues de que ambos flows terminan bien.
   - Si el segundo flow falla, `mobile/artifacts/screenshots/` queda vacio aunque Maestro haya tomado PNGs parciales.

4. **Destino de reportes temporales poco operativo**
   - Por defecto `REPORTS_DIR` apunta a `${TMPDIR}/fenixcrm-maestro-reports`.
   - Eso es correcto para no commitear reportes pesados, pero dificulta recuperacion y debugging.
   - El runner deberia imprimir y preservar el path incluso en fallo, y opcionalmente copiar una captura de fallo saneada al output estable.

5. **Navegacion Maestro a Support es fragil**
   - El flow usa `tapOn id: tab-support` despues de interacciones dentro de Inbox.
   - En el fallo observado, Maestro marco el tap como completed, pero la pantalla visible siguio siendo Inbox.
   - El flow deberia usar deep links para cambiar de area cuando la captura no depende del gesto de tab bar.

6. **El flow no toma screenshots parciales en el destino final**
   - `takeScreenshot` genera PNGs en `REPORTS_DIR`.
   - El destino final recibe copias solo al final exitoso.
   - Para auditoria visual, conviene copiar cada captura completada o copiar en `trap` aunque el flow falle.

7. **La barra inferior muestra tabs fantasma sin icono**
   - En el screenshot de fallo se ve la barra inferior con los 5 tabs esperados y, ademas, rutas ocultas truncadas como `crm`, `contac...` y otras sin icono.
   - `mobile/app/(tabs)/_layout.tsx` declara iconos para `Inbox`, `Support`, `Sales`, `Activity` y `Governance`, pero algunas rutas sibling bajo `(tabs)` siguen apareciendo como tabs automaticos porque no estan declaradas/ocultas explicitamente.
   - El tab `crm` debe desaparecer por completo de la barra inferior.
   - Los dos ultimos tabs fantasma visibles sin imagen deben identificarse por ruta real y ocultarse con `href: null` o moverse fuera del Tabs navigator si corresponde.
   - La validacion visual debe comprobar que solo existen 5 tabs visibles y que los ultimos dos tabs reales (`Activity`, `Governance`) tienen icono renderizado.

## Decisiones propuestas

- Mantener `mobile/artifacts/screenshots/` como directorio estable de capturas finales y parciales utiles.
- Mantener reportes Maestro completos fuera del repo por defecto, pero copiar PNGs saneados al output estable tambien en fallo.
- Usar deep links para saltos entre areas principales del authenticated audit: `/support`, `/sales`, `/governance`, `/activity`, `/contacts`, `/workflows`.
- Conservar taps de tab bar solo cuando el objetivo explicito sea validar la barra inferior.
- Agregar preflight de dependencias runtime antes del seed: backend `/health`, BFF reachable, device conectado, APK instalada o disponible, `JWT_SECRET` documentado para backend local.
- La barra inferior debe tener exactamente 5 tabs visibles: `Inbox`, `Support`, `Sales`, `Activity`, `Governance`. `crm`, `contacts`, `workflows` y cualquier otra ruta legacy o auxiliar deben quedar ocultas.

## Restriccion de ejecucion: modelo medium

Este plan debe ejecutarse asumiendo capacidad de razonamiento **medium**. No depender de una pasada grande con muchos cambios acoplados.

Reglas operativas:

- Ejecutar una tarea por vez y reportar al terminar cada tarea.
- No combinar cambios de `seed-and-run.sh` y `authenticated-audit.yaml` en la misma tanda salvo que la tarea previa ya este validada.
- Preferir cambios pequeños con verificacion inmediata:
  - despues de preflight: probar fallo controlado sin backend y exito con `/health`;
  - despues de trap/copia: inducir fallo Maestro temprano y verificar que `artifacts/screenshots/` no queda vacio;
  - despues de deep links: correr solo el flow autenticado o el menor subconjunto Maestro posible si la herramienta lo permite.
- Evitar refactors generales del runner. Cambiar solo lo necesario para preflight, preservacion de artefactos y navegacion robusta.
- Si una validacion depende de infraestructura no disponible, parar y reportar el bloqueo con evidencia concreta en vez de seguir encadenando cambios.
- Mantener una lista explicita de archivos tocados por tarea para no mezclar cambios previos del worktree.

Formato de reporte por tarea:

```text
Tarea N - <nombre>
Estado: Completado | Bloqueado | Requiere ajuste
Complejidad: Baja | Media | Alta
Archivos tocados:
- <archivo>
Validacion:
- <comando/resultado>
Riesgos o siguiente ajuste:
- <si aplica>
Tokens: ~N
```

## Orden de ejecucion y dependencias

| Orden | Tarea | Depende de | Complejidad | Motivo |
|---:|---|---|---|---|
| 1 | Definir contrato de invocacion | Ninguno | Baja | Aclarar si el comando canonico es `cd mobile && npm run screenshots` o si se agrega wrapper root. |
| 2 | Agregar preflight runtime en `seed-and-run.sh` | Tarea 1 | Media | Debe fallar temprano con mensajes claros para backend, BFF, device, Maestro y APK. |
| 3 | Copiar screenshots en fallo | Ninguno | Media | Requiere `trap`/handler de error que no oculte el exit code original y que copie PNGs parciales. |
| 4 | Separar reportes temporales de output estable | Tarea 3 | Baja | Mantener `REPORTS_DIR` temporal pero garantizar que `OUTPUT_DIR` contenga lo recuperable. |
| 5 | Auditar y corregir tabs fantasma del footer | Ninguno | Media | Identificar rutas bajo `(tabs)` que Expo Router esta mostrando sin icono; ocultar `crm` y los tabs auxiliares sin imagen. |
| 6 | Endurecer navegacion de authenticated audit | Tarea 5 | Media | Reemplazar saltos por tab con deep links cuando la captura no prueba el tab bar; evita depender de una barra contaminada por tabs fantasma. |
| 7 | Corregir bloque Support | Tarea 6 | Media | Usar `openLink: fenixcrm:///support` o `fenixcrm:///support/${SEED_CASE_ID}` y esperar el testID adecuado. |
| 8 | Validar que Contacts/Sales nuevos se capturan | Tareas 6-7 | Media | El flujo debe llegar a `15_sales_contacts_tab`, `16_contacts_list` y `17_contact_detail`. |
| 9 | Ejecutar `npm run screenshots` completo | Tareas 1-8 | Alta | Requiere backend, BFF, emulador, APK, Maestro y seed deterministico funcionando juntos. En modo medium, ejecutar solo despues de validaciones parciales verdes. |

## Implementacion propuesta

### Tarea 1 - Contrato de invocacion

Opcion recomendada: mantener el contrato mobile-only y documentarlo en `mobile/README.md` y en el output del runner:

```bash
cd mobile && npm run screenshots
```

Opcion alternativa: crear wrapper root si el equipo quiere soportar literalmente:

```bash
npm run screenshots
```

Esta alternativa implica agregar `package.json` root o un target `make screenshots`; no hacerla si el repo evita Node root.

### Tarea 2 - Preflight runtime

Agregar funciones en `mobile/maestro/seed-and-run.sh`:

- `check_backend_health`: `curl -fsS http://localhost:8080/health`
- `check_bff_reachable`: endpoint BFF real; si no hay `/health`, usar una ruta interna estable o al menos validar que `localhost:3000` acepta conexion.
- Mensaje especifico para backend local:
  ```bash
  JWT_SECRET='test-secret-32-chars-minimum!!!' go run ./cmd/fenix serve --port 8080
  ```
- No correr seed si el backend no esta sano.

### Tarea 3 - Copiar screenshots en fallo

Agregar handler de salida que:

1. capture `$?`
2. si existe `REPORTS_DIR`, ejecute sanitizacion best-effort
3. copie `*.png` a `OUTPUT_DIR`
4. preserve el exit code original
5. imprima ambos paths

Pseudoflujo:

```bash
finish() {
  local code=$?
  sanitize_reports || true
  copy_reports_screenshots || true
  log "Screenshots available in ${OUTPUT_DIR}"
  log "Temporary Maestro reports available in ${REPORTS_DIR}"
  exit "${code}"
}

trap finish EXIT
```

Debe cuidarse de no borrar `SEED_FILE` antes de terminar ni ejecutar `sanitize_reports` si `SEED_BOOTSTRAP_URL` no existe.

### Tarea 4 - Output estable

Mantener:

```bash
OUTPUT_DIR="${MOBILE_DIR}/artifacts/screenshots"
```

Definir comportamiento:

- run exitoso: contiene todas las capturas esperadas.
- run fallido: contiene capturas completadas y al menos el screenshot de fallo de Maestro.
- reportes completos: siguen en `REPORTS_DIR`.

### Tarea 5 - Footer: eliminar CRM y tabs fantasma sin icono

Auditar todas las rutas directas bajo:

```text
mobile/app/(tabs)/
```

Verificar que cada ruta que no sea una de las 5 visibles este declarada como hidden screen en `mobile/app/(tabs)/_layout.tsx`:

```tsx
<Tabs.Screen name="..." options={{ href: null }} />
```

Rutas a revisar de forma explicita:

- `crm` - debe estar oculto o eliminado de la superficie de tabs; no debe aparecer como tab visible.
- `contacts` - debe seguir existiendo como ruta canónica hidden, sin tab visible.
- `workflows` - debe seguir existiendo como ruta hidden, sin tab visible.
- `accounts`, `deals`, `cases`, `copilot/index`, `home` - deben permanecer hidden.
- cualquier otra carpeta sibling bajo `(tabs)` que aparezca en la barra inferior sin icono.

Validacion esperada:

- La barra inferior muestra exactamente: `Inbox`, `Support`, `Sales`, `Activity`, `Governance`.
- `Activity` y `Governance`, los dos ultimos tabs reales, renderizan sus iconos.
- No aparece `crm`.
- No aparecen tabs truncados como `contac...` ni rutas auxiliares sin imagen.
- `fenixcrm:///contacts` y `fenixcrm:///workflows` siguen funcionando por deep link aunque no sean tabs visibles.

Agregar o ajustar tests si existe cobertura de navegacion para tabs visibles. El test debe afirmar que solo los 5 wedge tabs tienen `tabBarButtonTestID` visible y que las rutas auxiliares tienen `href: null`.

### Tarea 6 - Navegacion robusta por deep links

Modificar `mobile/maestro/authenticated-audit.yaml`:

- Para Support:
  ```yaml
  - openLink: "fenixcrm:///support"
  ```
  o directo:
  ```yaml
  - openLink: "fenixcrm:///support/${SEED_CASE_ID}"
  ```
- Para Sales/Governance/Activity/Contacts/Workflows, preferir `openLink` cuando el objetivo sea capturar una pantalla, no validar la interaccion con tabs.

### Tarea 7 - Bloque Support

Camino mas robusto:

```yaml
- openLink: "fenixcrm:///support/${SEED_CASE_ID}"
- extendedWaitUntil:
    visible:
      id: "support-case-detail-screen"
    timeout: 20000
- takeScreenshot:
    path: "03_support_case_detail"
```

Si se quiere conservar cobertura de lista:

```yaml
- openLink: "fenixcrm:///support"
- extendedWaitUntil:
    visible:
      id: "support-cases-list"
    timeout: 20000
```

Pero hoy el contenedor real es `support-cases-list` via `CRMListScreen`; el item `support-cases-list-item-0` solo existe si hay casos visibles y renderizados.

### Tarea 8 - Capturas nuevas de Contacts

Confirmar que el flow llega a:

- `15_sales_contacts_tab`
- `16_contacts_list`
- `17_contact_detail`

La captura de Contacts dentro de Sales debe validar el nuevo tab segmentado:

```yaml
- openLink: "fenixcrm:///sales"
- extendedWaitUntil:
    visible:
      id: "sales-screen"
    timeout: 20000
- tapOn:
    id: "sales-tab-contacts"
- extendedWaitUntil:
    visible:
      id: "contacts-list"
    timeout: 20000
- takeScreenshot:
    path: "15_sales_contacts_tab"
```

La lista esperada de screenshots debe incluir tambien `18_workflows_list`.

### Tarea 9 - Validacion final

Precondiciones:

```bash
JWT_SECRET='test-secret-32-chars-minimum!!!' go run ./cmd/fenix serve --port 8080
```

BFF en `localhost:3000`, emulador conectado, APK debug instalada o disponible.

Comando:

```bash
cd mobile && npm run screenshots
```

Criterios de aceptacion:

- exit code 0 en run completo.
- `mobile/artifacts/screenshots/` contiene los PNG esperados.
- En run fallido inducido, `mobile/artifacts/screenshots/` no queda vacio si Maestro alcanzo a tomar capturas.
- El log imprime `Screenshots available in ...` y `Temporary Maestro reports available in ...` tanto en exito como en fallo.

## QA local

Por tocar `mobile/`, despues de implementar correcciones deben pasar:

```bash
bash scripts/qa-mobile-prepush.sh
cd mobile && npm run screenshots
```

Si `npm run screenshots` falla por infraestructura local no disponible, reportar exactamente:

- backend health
- BFF reachability
- `adb devices`
- APK presente/instalada
- path de `REPORTS_DIR`
- contenido de `mobile/artifacts/screenshots/`

## Resultado 2026-04-19

Validacion completada en modo medium, una tarea por vez.

Comandos ejecutados:

```bash
bash scripts/qa-mobile-prepush.sh
cd mobile && npm run e2e:build
JWT_SECRET='test-secret-32-chars-minimum!!!' go run ./cmd/fenix serve --port 8080
cd mobile && npm run screenshots
```

Resultado:

- QA mobile pre-push: pass.
- Build Detox Android debug: pass.
- Screenshots Maestro: pass.
- Output estable: `mobile/artifacts/screenshots/`.
- Reportes temporales: `${TMPDIR}/fenixcrm-maestro-reports`.

PNGs generados:

- `01_auth_login.png`
- `02_inbox.png`
- `03_support_case_detail.png`
- `04_sales_brief.png`
- `05_governance.png`
- `06_inbox_signal_detail.png`
- `08_activity_run_detail_denied.png`
- `09_governance_audit.png`
- `10_governance_usage.png`
- `11_support_kb_trigger.png`
- `12_sales_lead_prospecting.png`
- `13_sales_deal_risk_active.png`
- `14_activity_insights.png`
- `15_sales_contacts_tab.png`
- `16_contacts_list.png`
- `17_contact_detail.png`
- `18_workflows_list.png`
