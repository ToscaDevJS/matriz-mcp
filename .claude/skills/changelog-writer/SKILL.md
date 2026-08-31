---
name: changelog-writer
description: Escribe y mantiene archivos CHANGELOG.md siguiendo el estilo de changelog de Claude Code (lista plana de bullets que empiezan por verbo-categoría, SemVer, enfoque en el usuario). Úsala SIEMPRE que el usuario quiera crear un changelog, añadir una entrada de versión, documentar una release, convertir commits de git en notas de cambios, o cuando mencione "changelog", "notas de versión", "release notes", "qué cambió en esta versión" o "documentar esta versión" — aunque no pida explícitamente "un changelog". También aplícala al revisar o reescribir entradas de un changelog existente para que cumplan estas convenciones.
---

# Changelog Writer

Escribe entradas de changelog en el estilo concreto del changelog de Claude Code. Este estilo NO es el de "Keep a Changelog" con sub-encabezados agrupados (`### Added`, `### Fixed`). Es una **lista plana de bullets donde cada bullet empieza por su verbo-categoría**.

## Reglas de formato (innegociables)

1. El archivo empieza con `# Changelog`.
2. Cada release es un encabezado `## X.Y.Z` (SemVer: MAJOR.MINOR.PATCH), **más reciente arriba**.
3. **No hace falta listar todas las versiones.** Los huecos son normales (p. ej. 2.1.196 → 2.1.195 → 2.1.193). Solo se documentan las versiones que tienen cambios que comunicar.
4. Bajo cada versión va una **lista plana de bullets (`-`)**. Sin sub-encabezados, sin agrupar por secciones.
5. **Cada bullet empieza por un verbo-categoría** (ver vocabulario abajo). Un cambio = un bullet.
6. Una release sin nada destacable se documenta con un único bullet: `- Bug fixes and reliability improvements`.

## Vocabulario de categorías

Ordenadas por frecuencia de uso real. La inmensa mayoría de bullets caen en las tres primeras:

- **Added** — una característica, comando, flag, setting o variable de entorno nuevos.
- **Fixed** — corrección de un bug. Es la categoría más usada con diferencia.
- **Improved** — mejora de algo que ya existía (fiabilidad, mensajes, rendimiento percibido, UX).
- **Changed** — un cambio de comportamiento por defecto que el usuario notará.
- **Reduced** — reducción mensurable de coste (CPU, memoria, tokens, latencia). A menudo con cifra: "by ~37%".
- **Removed** / **Deprecated** — algo que se quita o se marca como obsoleto.
- **Updated** / **Renamed** — actualización de dependencia/modelo o renombrado de algo.
- **Security:** — ítems de seguridad. Se prefijan con la palabra y dos puntos.

Estas palabras-categoría van **siempre en inglés**, sea cual sea el idioma del resto del changelog (ver «Idioma de salida»).

También es válido **empezar el bullet directamente por el sujeto** cuando se describe un estado nuevo en vez de una acción:
- `The streaming idle watchdog is now on by default for all providers — it aborts and retries...`
- `Remote Control is now disabled when ANTHROPIC_BASE_URL points at a non-Anthropic host...`

Usa esta forma con moderación, solo cuando leer "Changed: ..." sonaría más torpe que enunciar el hecho directamente.

## Reglas de estilo (lo que da el "tono")

**Escribe para quien USA el producto, no para quien lo programó.** Un commit dice "refactor auth handler"; un bullet dice qué puede hacer ahora el usuario o qué síntoma desaparece.

1. **Los fixes describen el síntoma real, no el arreglo abstracto.** Patrón: `Fixed [lo que pasaba mal] when [bajo qué condición]`.
   - ✅ `Fixed the rate-limit warning flickering off when multiple parallel requests were in flight at the moment a usage limit was hit`
   - ❌ `Fixed rate limit bug`

2. **Comandos, flags, settings y env vars van en `backticks`.** Siempre con su nombre exacto.
   - ✅ `Added CLAUDE_CODE_DISABLE_MOUSE_CLICKS to disable mouse click/drag/hover in fullscreen mode`

3. **Cuando un cambio se puede desactivar o invertir, dilo en la misma línea.**
   - ✅ `... Set CLAUDE_ENABLE_STREAM_WATCHDOG=0 to disable.`
   - ✅ `... (disable with CLAUDE_CODE_DISABLE_BG_SHELL_PRESSURE_REAP=1)`

4. **Las mejoras de rendimiento llevan cifra cuando se conoce.**
   - ✅ `Reduced CPU usage during streaming responses by ~37% by coalescing text updates to 100ms`

5. **Un bullet = un cambio.** Puede ser una frase larga con guion largo (`—`) para añadir matiz o consecuencia, pero no mezcles dos cambios sin relación.

6. **Si un cambio tiene una consecuencia que el usuario debe saber al actualizar, hazla explícita** (sobre todo en seguridad y telemetría). Estos bullets pueden ser más largos.

7. **Sé concreto con los nombres propios del producto**: comandos (`/model`, `/rewind`), vistas (`claude agents`), proveedores (Bedrock, Vertex), plataformas (Windows, macOS, Linux). El lector reconoce esas piezas.

## Orden dentro de una versión

No hay un orden estricto, pero la tendencia es: primero los **Added**, luego los **Fixed**, después **Improved**, y al final **Reduced/Changed/Removed** y misceláneos. Mantén juntos los de la misma categoría.

## Highlights y tipos de versión

En versiones **menores o mayores estables** (`x.Y.0`, `X.0.0`), abre el bloque de la versión con una sección de destacados antes de la lista plana:

```markdown
## 1.4.0

**Highlights**

- **Inicio de sesión con Google:** entra con tu cuenta, sin crear otra contraseña.
- **Modo oscuro:** actívalo en Ajustes; sigue el tema del sistema.

- Added inicio de sesión con Google
- Fixed el cierre inesperado al subir imágenes de más de 5MB
```

Reglas de los Highlights:

1. De **3 a 5 puntos** (2 si la versión es pequeña), cada uno con **título en negrita + dos puntos + una sola frase** centrada en qué gana el usuario.
2. **Prioriza características nuevas** sobre fixes, rendimiento y tareas internas.
3. **Sin números de PR, sin enlaces y sin nombres de autores.**
4. **Nada experimental o en preview en los Highlights de una versión estable**: una característica solo se destaca ahí cuando se gradúa a estable.
5. Los Highlights **resumen, no sustituyen**: la lista plana completa va siempre debajo.

Comportamiento según el tipo de versión:

- **Patch (`x.y.Z`)** — sin Highlights; directo a la lista plana (normalmente solo `Fixed`/`Improved`).
- **Preview/beta (`x.Y.0-preview.N`)** — el sufijo va en el encabezado (`## 1.5.0-preview.1`) y lo inmaduro se marca con `(experimental)` dentro del propio bullet. Puede llevar Highlights propios si el salto es grande.
- **Nightly o builds internos** — no se documentan en el changelog.

## Flujo de trabajo

### Caso A — Crear un CHANGELOG.md desde cero
1. Pregunta (o deduce) la versión inicial. Si es una primera entrega, `## 0.1.0` o `## 1.0.0` según madurez.
2. Genera el esqueleto con `# Changelog` y la primera versión.
3. Rellena los cambios siguiendo las reglas de arriba.

### Caso B — Añadir una nueva versión a un changelog existente
1. **Lee el changelog actual** para imitar su estilo exacto (a veces difiere en matices del de Claude Code; respeta el que ya tiene el proyecto si está consolidado).
2. Determina el número de versión nuevo según SemVer:
   - PATCH: solo Fixed/Improved internos.
   - MINOR: hay algún Added compatible hacia atrás.
   - MAJOR: hay un Breaking change / Removed que rompe compatibilidad.
3. Inserta el nuevo bloque `## X.Y.Z` **encima** del más reciente.

### Caso C — Generar entradas a partir de git (si hay acceso al repo)
1. Obtén los commits desde la última versión etiquetada:
   ```bash
   git log $(git describe --tags --abbrev=0)..HEAD --pretty=format:'%s%n%b' --no-merges
   ```
   Si no hay tags: `git log --pretty=format:'%s%n%b' --no-merges -n 50`.
2. **Traduce cada commit al estilo del changelog**, no lo copies. Descarta los commits internos sin impacto para el usuario (bumps de lint, formateo, merges). Agrupa los relacionados en un solo bullet si describen un mismo cambio observable.
3. Asigna la categoría (Added/Fixed/Improved/...) según el tipo de commit (`feat:` → Added, `fix:` → Fixed, `perf:` → Reduced/Improved, etc.) pero **fíate del contenido real**, no solo del prefijo.

En cualquier caso, **muestra el bloque generado al usuario y pídele que confirme la versión y el recorte de cambios** antes de escribir el archivo. El usuario casi siempre sabe mejor que git qué merece aparecer.

## Idioma de salida

**Antes de generar nada, ofrece al usuario elegir el idioma del changelog.** No asumas inglés por defecto.

- Si ya hay un `CHANGELOG.md` en el proyecto, detecta su idioma y **propón continuar en ese mismo idioma** ("Veo que tu changelog está en inglés, ¿lo continúo en inglés?"). El usuario puede cambiarlo.
- Si es un changelog nuevo, **pregunta explícitamente**: "¿En qué idioma quieres el changelog: inglés o español?" (u otro que indique). Sugiere inglés si el código y la documentación del proyecto están en inglés, ya que es lo más común para librerías y APIs.
- Si el usuario ya ha dejado claro el idioma antes en la conversación, respeta esa elección sin volver a preguntar.

Una vez elegido el idioma, **mantenlo consistente en todo el archivo**. La elección aplica solo a las **descripciones**; las palabras-categoría van **siempre en inglés** — `Added`, `Fixed`, `Improved`, `Changed`, `Reduced`, `Removed`, `Deprecated`, `Updated`, `Renamed`, `Security:` — igual que el encabezado `**Highlights**`. Un changelog en español mezcla así: `- Fixed el cierre inesperado al subir imágenes de más de 5MB`. Mantén también en su forma original los nombres propios de comandos, flags y variables de entorno (`/model`, `CLAUDE_ENABLE_STREAM_WATCHDOG`).

## Plantilla mínima

```markdown
# Changelog

## 0.2.0

**Highlights**

- **<Título del cambio estrella>:** <qué gana el usuario, en una frase>.

- Added <característica nueva>, configurable con `<flag-o-setting>`
- Fixed <síntoma> when <condición>
- Improved <qué> — <consecuencia para el usuario>
- Reduced <recurso> by ~<n>% by <cómo>

## 0.1.0

- Initial release
```
