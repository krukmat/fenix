---
doc_type: task
title: Limpiar CRM tab + agregar Contacts en Sales
status: completed
created: 2026-04-19
completed: 2026-04-19
---

## Context

Se implementó previamente un CRM grid dentro de Governance que no convence. Contacts es la única entidad sin punto de entrada visible. El objetivo es:

1. **Quitar los entry points visibles del hub `crm`** y el CRM grid en Governance.
2. **Agregar** Contacts como 4to tab segmentado en Sales (junto a Accounts, Deals, Leads).
3. **Mantener `/contacts` como ruta canónica hidden** para lista y detalle de contactos.

## Decisiones cerradas

- `mobile/app/(tabs)/contacts/` sigue siendo la ruta canónica de Contacts. Ya está registrada como hidden tab en `mobile/app/(tabs)/_layout.tsx` y no debe aparecer en la barra inferior.
- Sales renderiza Contacts como tab embebido, pero la navegación al detalle usa ruta absoluta:
  ```tsx
  wedgeHref(`/contacts/${item.id}`)
  ```
- No se agregan `Stack.Screen name="contacts/*"` en `mobile/app/(tabs)/sales/_layout.tsx` porque no existen archivos físicos bajo `mobile/app/(tabs)/sales/contacts/`.
- El árbol `mobile/app/(tabs)/crm/` no se elimina en esta tarea. Queda como shim legacy sin entry point visible. Si se quiere borrarlo, debe abrirse una tarea separada con revisión de deep links, Maestro/Detox y compatibilidad.
- Para evitar duplicación, se extrae la lista de contactos a un componente compartido y lo usan tanto `/contacts` como el tab Contacts dentro de Sales.

## Archivos afectados

| Archivo | Cambio |
|---|---|
| `mobile/app/(tabs)/_layout.tsx` | Quitar `<Tabs.Screen name="crm" .../>` |
| `mobile/app/(tabs)/governance/_layout.tsx` | Quitar todas las rutas `crm/*` del Stack |
| `mobile/app/(tabs)/governance/index.tsx` | Quitar `CRMGrid`, `CRM_ENTITIES`, import `Card`, estilos crm* |
| `mobile/app/(tabs)/sales/_layout.tsx` | Sin cambios funcionales para contacts; no registrar rutas que no viven bajo `sales/` |
| `mobile/app/(tabs)/sales/index.tsx` | Agregar tab "Contacts" al selector segmentado |
| `mobile/app/(tabs)/contacts/index.tsx` | Reemplazar lógica de lista inline por componente compartido |
| `mobile/src/components/contacts/ContactsListContent.tsx` | Nuevo componente compartido para lista de contactos |

## Orden de ejecución y dependencias

| Orden | Paso | Depende de | Complejidad | Motivo |
|---:|---|---|---|---|
| 1 | Quitar `crm` de `mobile/app/(tabs)/_layout.tsx` | Ninguno | Baja | Retira el entry point global del hub CRM antes de limpiar consumidores internos. Cambio mecánico de una screen hidden. |
| 2 | Limpiar rutas `crm/*` de `governance/_layout.tsx` | Paso 1 | Baja | Governance deja de anunciar rutas CRM que ya no deben ser parte de su superficie. Es eliminación de bloque acotada. |
| 3 | Limpiar `CRMGrid` de `governance/index.tsx` | Paso 2 | Media | La UI deja de enlazar a rutas CRM desde Governance después de retirar esas rutas del stack. Requiere limpiar imports, JSX, constantes y estilos sin dejar referencias colgantes. |
| 4 | Validar que `sales/_layout.tsx` no registre `contacts/*` | Ninguno, pero debe hacerse antes del Paso 7 | Baja | Fija la estrategia de routing: Contacts vive en `/contacts`, no bajo `/sales/contacts`. No debería producir cambios. |
| 5 | Crear `ContactsListContent` compartido | Ninguno | Media | Genera la pieza reutilizable que necesitan la ruta canónica `/contacts` y el tab Sales. Debe conservar búsqueda, paginación, refresh, error state y test IDs. |
| 6 | Simplificar `contacts/index.tsx` para usar `ContactsListContent` | Paso 5 | Media | Mantiene la ruta canónica funcionando con el nuevo componente antes de exponerlo en Sales. Riesgo principal: romper imports, wrapper o `testID="contacts-screen"`. |
| 7 | Agregar tab Contacts en `sales/index.tsx` | Pasos 4 y 5 | Media | Sales puede renderizar la lista compartida y navegar al detalle por la ruta absoluta `/contacts/[id]`. Requiere revisar selector de 4 tabs en pantallas angostas. |
| 8 | Ejecutar verificación manual y QA local obligatorio | Pasos 1-7 | Alta | Valida ausencia del hub CRM en Governance, presencia de Contacts en Sales y gates mobile completos. La complejidad viene de tiempo de ejecución y posibles fallos de entorno o tests existentes. |

Notas de ejecución:

- Los pasos 1-3 son la limpieza de Governance y pueden hacerse en una misma tanda de edición.
- Los pasos 5-7 son la incorporación de Contacts en Sales y deben respetar el orden porque `sales/index.tsx` depende del componente compartido.
- El paso 4 es una validación deliberada: no debe producir cambios salvo que el archivo tenga rutas `contacts/*` agregadas por error.

Complejidad global estimada: **Media**. La implementación es acotada, pero toca navegación Expo Router, refactor compartido y QA mobile completo.

## Implementación

### Paso 1 — Quitar `crm` de `_layout.tsx`
**Archivo:** `mobile/app/(tabs)/_layout.tsx`

Eliminar la línea:
```tsx
<Tabs.Screen name="crm" options={{ href: null }} />
```
y su comentario asociado.

### Paso 2 — Limpiar `governance/_layout.tsx`
**Archivo:** `mobile/app/(tabs)/governance/_layout.tsx`

Eliminar el bloque completo de Stack.Screen de rutas `crm/*` (13 entradas con su comentario).

Resultado esperado: solo quedan `index`, `audit`, `usage`.

### Paso 3 — Limpiar `governance/index.tsx`
**Archivo:** `mobile/app/(tabs)/governance/index.tsx`

Eliminar:
- Import `Card` de `react-native-paper` (volver a solo `useTheme`)
- Constante `CRM_ENTITIES`
- Componente `CRMGrid`
- El `<View style={styles.section}><CRMGrid .../></View>` dentro de `GovernanceContent`
- Estilos: `crmGrid`, `crmCard`, `crmCardContent`, `crmIcon`

### Paso 4 — Validar `sales/_layout.tsx`
**Archivo:** `mobile/app/(tabs)/sales/_layout.tsx`

No agregar rutas `contacts/*` en este archivo.

Motivo: las pantallas reales viven en `mobile/app/(tabs)/contacts/`, que es sibling de `sales/`, no child de `sales/`. Registrar `Stack.Screen name="contacts/index"` en el stack de Sales crea una expectativa de ruta física `mobile/app/(tabs)/sales/contacts/index.tsx` que no existe.

Resultado esperado: `sales/_layout.tsx` mantiene sus rutas actuales:
- `index`
- `[id]`
- `deal-[id]`
- `leads/[id]`

La navegación de Contacts desde Sales debe usar rutas absolutas hacia `/contacts`.

### Paso 5 — Extraer lista compartida de Contacts
**Archivo nuevo:** `mobile/src/components/contacts/ContactsListContent.tsx`

Crear un componente que contenga la lógica actual de `mobile/app/(tabs)/contacts/index.tsx`:

- `useContacts()`
- estado `searchValue`
- flatten de `data.pages`
- filtro por `name`, `email`, `accountName`
- `CRMListScreen`
- render de cada contacto
- navegación a detalle con:
  ```tsx
  router.push(wedgeHref(`/contacts/${item.id}`))
  ```

Contrato sugerido:

```tsx
export function ContactsListContent() {
  // misma lógica actual de contacts/index.tsx
}
```

Test IDs requeridos:

- pantalla/lista: mantener `contacts-screen` en el wrapper de ruta existente
- items: mantener `contacts-list-item-${index}`
- prefijo de `CRMListScreen`: mantener `contacts`

### Paso 6 — Simplificar pantalla canónica `/contacts`
**Archivo:** `mobile/app/(tabs)/contacts/index.tsx`

Reemplazar la lógica inline por:

```tsx
import { ContactsListContent } from '../../../src/components/contacts/ContactsListContent';

export default function ContactsScreen() {
  return (
    <View style={styles.container} testID="contacts-screen">
      <ContactsListContent />
    </View>
  );
}
```

Mantener `testID="contacts-screen"` para no romper pruebas existentes.

### Paso 7 — Agregar tab Contacts en `sales/index.tsx`
**Archivo:** `mobile/app/(tabs)/sales/index.tsx`

#### 7a. Agregar 'Contacts' al tipo Tab y al TabBar:
El archivo tiene `type Tab = 'accounts' | 'deals' | 'leads'` y un componente `TabBar` hardcoded con 3 botones.
- Extender tipo: `type Tab = 'accounts' | 'deals' | 'leads' | 'contacts'`
- Agregar 4to botón en `TabBar` con `testID="sales-tab-contacts"`

#### 7b. Renderizar Contacts con el componente compartido:

```tsx
import { ContactsListContent } from '../../../src/components/contacts/ContactsListContent';

// En SalesScreen:
{activeTab === 'contacts' ? <ContactsListContent /> : null}
```

No crear un `ContactsTab` duplicando la lógica de `contacts/index.tsx` dentro de `sales/index.tsx`.

#### 7c. Layout del selector

El selector pasa de 3 a 4 tabs. Verificar que no haya overflow en pantallas angostas. Si el texto no entra, ajustar padding/font size local del `TabBar` sin cambiar la barra inferior global.

**Hooks y componentes a reutilizar (ya existentes):**
- Hook: `useContacts` de `mobile/src/hooks/useCRM.ts` (confirmado en `contacts/index.tsx:7`)
- Componente: `CRMListScreen` de `mobile/src/components/crm` (confirmado en `contacts/index.tsx:6`)
- Nuevo componente compartido: `ContactsListContent`
- Tipo Tab: `type Tab = 'accounts' | 'deals' | 'leads'` → extender a incluir `'contacts'`
- `TabBar` component: hardcoded con 3 botones → agregar 4to botón Contacts

## Resultado visual esperado

**Sales tab — selector:**
```
[ Accounts ] [ Deals ] [ Leads ] [ Contacts ]
```

**Governance hub — limpio** (sin CRM grid):
```
Recent Usage       [View All]
Audit Trail              →
Quota States
```

**Barra inferior:** Inbox | Support | Sales | Activity | Governance (sin tab CRM)

## Verificación

1. Abrir tab Governance → verificar que NO aparecen los CRM cards
2. Abrir tab Sales → verificar que aparece el tab "Contacts" en el selector
3. Tocar tab Contacts en Sales → verificar que lista contactos
4. Tocar un contacto desde Sales → verificar navegación a `/contacts/[id]`
5. Abrir `/contacts` directamente o desde cualquier vínculo legacy → verificar que sigue mostrando la lista
6. Abrir un contacto desde `/contacts` → verificar navegación a detalle

## QA local obligatorio

Este cambio toca `mobile/`, por lo que aplica la Mobile Rule de `AGENTS.md`.

Atajo preferido:

```bash
bash scripts/qa-mobile-prepush.sh
```

Gates mínimos equivalentes si no se usa el atajo:

```bash
bash scripts/check-no-inline-eslint-disable.sh
cd mobile && npm run typecheck
cd mobile && npm run lint
cd mobile && npm run quality:arch
cd mobile && npm run test:coverage
```

## Ejecución

Estado: completado el 2026-04-19.

Validación ejecutada:

```bash
bash scripts/qa-mobile-prepush.sh
```

Resultado:

- no-inline-eslint: passed
- typecheck: passed
- lint: passed
- quality:arch: passed
- test:coverage: passed
- Jest: 57 suites passed, 445 tests passed
