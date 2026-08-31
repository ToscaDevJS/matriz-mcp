# Matriz — Handoff de implementación (greenfield → v0.1.0)

> **Documento de handoff para Claude Code.** Contiene el contexto completo, las
> decisiones de diseño, los contratos exactos y el plan de implementación de un
> servidor MCP de generación/edición de imágenes para sitios web de clientes, más
> una TUI que comparte el mismo core.
>
> Es autosuficiente: no necesitás la conversación que lo originó.
>
> **Escrito contra**: repositorio inexistente (proyecto nuevo). Verificado el
> 2026-08-31 contra `modelcontextprotocol/go-sdk` rama `main`, el README de
> `googleapis/go-genai`, y la documentación pública de las librerías y precios
> citados en §2.2 y §5.2.
>
> **Proveedor de imagen elegido**: API nativa de Google Gemini (familia Nano
> Banana) vía SDK oficial de Go. fal.ai queda diferido a PR-6 con disparador
> explícito. Razonamiento completo en §3.1.

---

## 0. Instrucciones de uso (leé esto primero)

### 0.1 Precondiciones

| Requisito | Comprobación | Si falla |
|---|---|---|
| Go instalado, versión ≥ 1.25 | `go version` | Instalar antes de empezar; el SDK sólo soporta versiones de Go vigentes |
| Directorio de trabajo vacío o repo git recién inicializado | `git status` | No mezclar con otro proyecto |
| Acceso a red para `go get` | `go env GOPROXY` | Sin red no se puede resolver el módulo |
| Ninguna API key de proveedor de imagen en el repo | `git grep -i "api[_-]key"` → vacío | Ver regla dura §7.3 |

**Línea base**: no hay tests todavía. A partir de PR-0, la línea base es
`go test ./...` en verde. Lo que importa es **cero fallos**, no un número
concreto de tests.

### 0.2 División del trabajo

Siete unidades encadenadas. Cada una deja el sistema compilando, con la suite en
verde, y utilizable por sí sola.

```
PR-0 (esqueleto + tipos)
  └─ PR-1 (pipeline determinista)      ← el valor real, sin red y sin coste
       └─ PR-2 (proveedores + Gemini + guard de presupuesto)
            └─ PR-3 (servidor MCP)     ← primer entregable usable por un LLM
                 └─ PR-4 (manifiesto del sitio)
                      └─ PR-5 (TUI)
                           └─ PR-6 (fal) ← DIFERIDO, con disparador explícito
```

Dependencias explícitas:
- PR-2 depende de los tipos `Asset` y `AssetRef` que introduce PR-0.
- PR-3 depende del `Provider` de PR-2 **y** del `fakeProvider` de PR-2 (los
  tests de PR-3 no pueden llamar a un proveedor real ni gastar dinero).
- PR-4 depende del sidecar `.meta.json` que escribe PR-1.
- PR-5 no añade lógica: sólo consume `internal/core`.
- **PR-6 no se ejecuta con el resto.** Ver su disparador en §4.

**Advertencia de divergencia**: este documento se escribió contra un repositorio
que no existe. Si al empezar ya hay código, verificá cada unidad contra tu
working tree, saltá lo ya resuelto y anotalo en `PENDING.md`.

### 0.3 Convenciones del repo destino

| Aspecto | Convención |
|---|---|
| Idioma del código, comentarios, nombres de tools | Inglés |
| Idioma de la documentación (`docs/`, `README.md`) | Español |
| Mensajes de commit | Inglés, imperativo, `feat:` / `fix:` / `test:` / `docs:` |
| Ramas | `pr-0-scaffold`, `pr-1-deterministic`, … |
| Presupuesto por PR | Indicado en cada unidad de §4. Excederlo un 20 % es aceptable; el doble no — partir la unidad y avisar |
| Tests | Table-driven. Para transformaciones de imagen, **golden files**: se genera el fixture una vez con `-update`, se revisa a ojo, y a partir de ahí se compara byte a byte |
| Disciplina | RED → implementar → GREEN. El test de una unidad se escribe antes que su implementación |

### 0.4 Reglas de conflicto

1. Si una tarea de §4 contradice una regla dura de §7, **la regla gana y este
   documento está mal**: parar, anotar en `PENDING.md` y reportar al usuario.
2. La sección §5 (contratos) se copia **tal cual**. Los placeholders `<...>` se
   sustituyen; el resto no se "mejora" sobre la marcha. Cada línea de un
   contrato tiene una decisión detrás.
3. Si al implementar descubrís que una librería o API se comporta distinto a lo
   afirmado aquí, ver la nota final del documento.

---

## 1. Contexto mínimo suficiente

### 1.1 Qué es

**Matriz** es un servidor MCP local (transporte stdio) que da a un agente LLM
—Claude Code, principalmente— la capacidad de generar, editar y exportar las
imágenes de un sitio web de cliente, más una TUI en Go que sirve de visor y
curador humano sobre el mismo motor.

El caso de uso concreto que lo origina: desarrollo de landings para peluquerías
y barberías con un plazo de entrega objetivo de 15 días por sitio. El cuello de
botella real en ese trabajo **no es generar imágenes bonitas**, es normalizar el
material heterogéneo que manda el cliente (fotos de móvil, luces distintas,
fondos sucios, encuadres incoherentes) y exportarlo en variantes responsive
correctas.

### 1.2 Los seis principios que gobiernan el diseño

Estos existen para impedir "mejoras" destructivas. Si una decisión de
implementación los viola, la decisión está mal.

**P1 — Separación estricta entre generativo y determinista.**
Recortar, redimensionar, ajustar brillo/contraste/saturación, convertir formato
y generar variantes responsive son operaciones deterministas: se hacen con una
librería local, son gratis, instantáneas y reproducibles. Generar contenido
nuevo, inpainting, quitar fondo y expandir encuadre son operaciones generativas:
cuestan dinero, tardan segundos y no son reproducibles salvo por seed.
Si se exponen mezcladas en una sola tool, el LLM llamará a un modelo de pago
para bajarle el brillo a una foto. La separación se hace visible en los nombres
de las tools y en sus descripciones.

**P2 — El modelo tiene que poder ver el resultado.**
Una tool que devuelve `"guardado en /assets/hero-3.png"` obliga al LLM a iterar
a ciegas y a quemar presupuesto adivinando. Toda tool que produce o modifica una
imagen devuelve una **miniatura** como `ImageContent` además de los datos
estructurados. El archivo a resolución completa se queda en disco.

**P3 — Un core, dos frontends.**
La lógica vive en `internal/core`. El servidor MCP (`cmd/matriz-mcp`) y la TUI
(`cmd/matriz-tui`) son dos consumidores del mismo motor, no dos
implementaciones. Ninguna lógica de negocio vive en un `cmd/`.

**P4 — Proveedores intercambiables.**
El core no conoce ningún proveedor de generación concreto. Habla con una
interfaz `Provider`. Añadir un proveedor nuevo no debe obligar a tocar el core
ni las tools. (Este es el patrón `Adapter` de `gentle-ai`, que resolvió el mismo
problema para 16 agentes distintos.)

**P5 — Reproducibilidad por sidecar.**
Cada imagen generada escribe un archivo `.meta.json` junto a ella con proveedor,
modelo, prompt, seed y parámetros. Sin esto, "la de antes pero más azul" es
irrecuperable.

**P6 — El original del cliente es sagrado.**
Ninguna operación modifica un archivo de entrada in-place. Todo se escribe en el
directorio de derivados. Ver regla dura §7.5.

### 1.3 Qué NO es

- No es un editor de imágenes de propósito general.
- No es un generador de fotos de personas. Ver §3.4 (riesgo) y §7.6 (regla dura).
- No es un servicio remoto. Transporte **stdio**, proceso local, sin HTTP, sin
  autenticación, sin multi-tenancy. Todo eso es alcance futuro y no se
  implementa ahora.

---

## 2. Estado verificado

### 2.1 Código

**No existe.** Proyecto nuevo. Nada que re-hacer, nada que preservar.

### 2.2 Hechos externos verificados el 2026-08-31

Estos se comprobaron contra la fuente, no contra memoria. Se pueden dar por
ciertos al implementar.

| Hecho | Fuente | Estado |
|---|---|---|
| El SDK oficial de Go para MCP existe, es estable y se mantiene con Google | `README.md` del repo | ✅ verificado |
| `mcp.NewServer` + `mcp.AddTool[In, Out]` + `server.Run(ctx, &mcp.StdioTransport{})` es la API de arranque | `README.md`, ejemplo completo | ✅ verificado |
| `mcp.ImageContent` existe con campos `Data []byte`, `MIMEType string`, `Annotations`, `Meta` | `mcp/content.go` líneas 57-70 | ✅ verificado |
| `CallToolResult` tiene `Content []Content` y `StructuredContent any`; con `ToolHandlerFor` **no** se debe rellenar `StructuredContent` a mano (se puebla desde el valor `Out`) | `mcp/protocol.go` líneas 266-300 | ✅ verificado |
| Si `Content` queda sin poner, el SDK lo puebla con el JSON del output estructurado | `mcp/protocol.go`, doc de `Content` | ✅ verificado |
| `ToolAnnotations` tiene `ReadOnlyHint`, `DestructiveHint`, `IdempotentHint`, `OpenWorldHint`, `Title` | `mcp/protocol.go` línea 1967 | ✅ verificado |
| `server.AddResource(r *Resource, h ResourceHandler)` y `AddResourceTemplate` existen | `mcp/server.go` líneas 611, 630 | ✅ verificado |
| SDK v1.7.0+ soporta la spec MCP `2026-07-28` | tabla de compatibilidad del README | ✅ verificado |
| **roots, sampling y logging están deprecados** desde la spec `2026-07-28` (SEP-2577); el SDK los mantiene ≥12 meses por compatibilidad | README, sección Version Compatibility | ✅ verificado |
| `gen2brain/webp` y `gen2brain/avif` codifican WebP y AVIF sin CGo (libwebp / libavif+aom compilados a WASM, ejecutados con wazero; intentan primero librería dinámica vía purego) | READMEs de ambos repos | ✅ verificado por documentación, ⚠️ **sin compilar** |
| `deepteams/webp` es una alternativa pura Go, cero dependencias y cero CGo, con benchmarks publicados mejores que la vía WASM en lossy | README del repo | ✅ verificado por documentación, ⚠️ **sin compilar** |

#### Sobre el proveedor de imagen (Google Gemini / Nano Banana)

| Hecho | Fuente | Estado |
|---|---|---|
| Existe SDK oficial de Go: `go get google.golang.org/genai`, licencia Apache 2.0, soporta la Gemini Developer API con `genai.NewClient(ctx, &genai.ClientConfig{APIKey: ..., Backend: genai.BackendGeminiAPI})` | README de `googleapis/go-genai` | ✅ verificado |
| La generación de imagen se hace con `client.Models.GenerateContent(ctx, "<model-id>", contents, ...)`, con ejemplo oficial en Go | documentación de Google Cloud | ✅ verificado |
| Los modelos Nano Banana son la generación de imagen **nativa** de Gemini: un mismo modelo entiende texto, razona sobre él y genera imágenes | documentación pública de precios de Google | ✅ verificado |
| Una imagen de resolución estándar consume ~1.120 tokens de salida; una de 4K, ~2.000. El coste de entrada por imagen es ~$0.0011, despreciable | páginas de precios de Gemini API | ✅ verificado |
| Vía Pro (`gemini-3-pro-image-preview`): $0.134 por imagen de 1K **o** 2K, $0.24 por 4K. **1K y 2K cuestan lo mismo** porque consumen los mismos 1.120 tokens | páginas de precios de Gemini API | ✅ verificado |
| Nano Banana 2 Lite (Gemini 3.1 Flash-Lite Image, junio 2026) sólo saca 1K, tarda ~4 s, y cuesta $0.0336 estándar / $0.0168 en Batch — el precio por imagen más bajo de Google | análisis de precios de la familia Nano Banana | ✅ verificado |
| **Todas** las imágenes generadas por la API llevan marca SynthID (invisible) | misma fuente | ✅ verificado |
| Los outputs de tier gratuito y Google AI Pro conservan una **marca visible** de Gemini; los de Google AI Ultra y los de desarrollador de Google AI Studio la quitan | anuncio de Nano Banana Pro | ✅ verificado |
| A través de Google AI Studio, el tier gratuito da 50 peticiones al día sin tarjeta | análisis de precios | ✅ verificado, ⚠️ sujeto a cambio |
| BiRefNet v2 en fal cuesta $0.0008 por segundo de cómputo y es el mejor para detalle fino como pelo en retratos | comparativa de background removers de fal | ✅ verificado por documentación |

### 2.3 Lo que NO está verificado

Marcado explícitamente para que no se afirme de más:

- **No se compiló nada.** Ninguna de las librerías de §5.2 se probó en un
  entorno real, **incluido el SDK de Google**. La primera tarea de PR-0 es
  exactamente eso.
- La firma exacta con la que se combinan `Content` propio y `Out` estructurado
  en el mismo handler **hay que confirmarla con el MCP Inspector** en PR-3.
  El código del SDK sugiere que funciona; no se ejecutó.
- **Los precios de §2.2 salen de fuentes de febrero a agosto de 2026 y las
  páginas de precios cambian sin avisar.** Verificarlos en la página oficial de
  Gemini API antes de escribir la tabla de coste de §5.15, y anotar la fecha de
  consulta en `docs/proveedores.md`.
- El ID `gemini-3-pro-image-preview` lleva `-preview`: los modelos en preview se
  deprecan. Por eso va en configuración, no en código (regla dura §7.11).

---

## 3. Riesgos y decisiones abiertas — qué falta y por qué importa

### 3.1 (RESUELTO) Proveedor de generación

**Decisión**: se arranca con la **API nativa de Google Gemini (familia Nano
Banana)** a través del SDK oficial de Go. fal.ai queda diferido a PR-6.

**Por qué**, en orden de peso:

1. **Es el único con SDK oficial de Go.** fal publica clientes de TypeScript y
   Python, no de Go: empezar por fal significaría escribir el cliente HTTP a
   mano dentro de PR-2. Google lo da hecho y probado.
2. **Permite validar el bucle completo gratis.** El tier gratuito de Google AI
   Studio da 50 peticiones diarias sin tarjeta, así que PR-2 y PR-3 se prueban
   de punta a punta antes de gastar nada.
3. **Un solo modelo cubre generar y editar**, porque la generación es nativa del
   modelo multimodal. `img_generate_drafts` e `img_refine` pegan al mismo
   endpoint, lo que reduce la superficie de v0.1.0.
4. **El reparto barato/caro ya existe dentro de la familia**: Lite para
   borradores a 1K, Pro para el resultado final a 2K. Encaja exacto con el
   patrón de generación en dos pasos de §5.5.

**Objeción de facturación por tokens, resuelta.** Google factura por tokens,
igual que OpenAI. La diferencia es que aquí el consumo por imagen es **fijo y
documentado** (§2.2), así que `EstimateCostUSD` se implementa como tabla y sigue
sin tocar la red: el contrato §5.1 queda intacto. Con facturación variable por
longitud de prompt e imágenes de referencia, la estimación previa sería
imposible y el guard de presupuesto quedaría decorativo. Por eso Google y no
OpenAI.

**Lo que esta decisión NO cubre**, y es la razón de que PR-6 exista: quitar un
fondo con Gemini cuesta $0.134 por imagen; con BiRefNet en fal cuesta una
fracción de céntimo y con mejor resultado en pelo fino. Y no hay control exacto
de paleta de marca.

### 3.1-bis (MEDIO) Marca de agua visible según la ruta de salida

**Qué pasa**: los outputs del tier gratuito y de Google AI Pro conservan una
marca visible de Gemini; los de Google AI Ultra y los de desarrollador de Google
AI Studio la quitan. Además, **todas** las imágenes llevan marca SynthID
invisible, sin excepción.

**Consecuencia de no cerrarlo**: se entrega a un cliente una web con la estrella
de Gemini estampada en el hero. Es un fallo silencioso: el prototipo funciona,
la imagen se ve bien en la miniatura, y el problema aparece en producción.

**Acción**: la aceptación manual `A-03` (§6) comprueba la ruta de salida real
antes de que ninguna imagen llegue a un entregable.

### 3.2 (CRÍTICO) Coste sin límite

**Qué falta**: un guard de presupuesto.

**Consecuencia**: un agente LLM en bucle puede llamar a `img_generate_drafts`
cuarenta veces mientras "explora opciones". A un coste por imagen realista, eso
es una factura desagradable generada en dos minutos y sin intervención humana.

**Acción**: PR-2 implementa el guard antes de que exista ninguna tool
generativa expuesta. Contrato en §5.6. Esto es deuda de seguridad y por eso va
**antes** que la feature que la necesita.

### 3.3 (MEDIO) Renderizado de imágenes en la terminal

**Qué falta**: decidir cómo muestra imágenes la TUI.

**Consecuencia**: bubbletea no dibuja imágenes. Mostrarlas requiere el protocolo
gráfico de Kitty, el de iTerm2 o sixel, y ninguno funciona en todas las
terminales. Una TUI de curación visual que no puede mostrar la imagen es
inútil.

**Decisión de este cierre**: PR-5 implementa la TUI **sin** renderizado de
imagen en terminal. Muestra la rejilla de assets con metadatos, y la tecla
`enter` abre un visor HTML local en el navegador por defecto
(`internal/preview` genera un HTML estático y lo abre con `xdg-open`/`open`).
Menos elegante, funciona en todas partes, y desbloquea la unidad. El
renderizado nativo queda como follow-up opcional.

### 3.4 (MEDIO) Derechos de imagen y contenido generado

**Qué falta**: una política escrita.

**Consecuencia**: entregar a una peluquería una web con caras generadas
presentadas como clientes reales es un problema legal y reputacional, no un
detalle estético. Y las fotos que manda el cliente pueden incluir a personas que
no consintieron su publicación.

**Acción**: PR-4 hace que el manifiesto marque cada asset con
`"origin": "client" | "generated" | "derived"`, y el campo es obligatorio. Ver
regla dura §7.6.

### 3.5 (BAJO) El nombre

`matriz` es una **decisión de este cierre**, no una preferencia declarada. Si
se cambia, hacerlo **antes de PR-0** (`go mod init`), no después: renombrar el
módulo a mitad del plan toca todos los imports.

---

## 4. Plan de trabajo — 6 unidades encadenadas

> Orden: PR-0 → PR-1 → PR-2 → PR-3 → PR-4 → PR-5.
> Cada PR cierra con `go test ./...` en verde y `go vet ./...` limpio.

### PR-0 — Esqueleto, tipos core y verificación de dependencias (~250 líneas)

Primera tarea de la unidad: **compilar un programa mínimo que importe las
librerías de §5.2** y confirmar que resuelven y enlazan en la máquina destino.
Si `gen2brain/avif` no compila, anotarlo y evaluar la alternativa pura Go
antes de seguir. Esto es un spike, no una suposición.

| # | Archivo | Cambio |
|---|---|---|
| 1 | `go.mod` | ＋ `go mod init github.com/<usuario>/matriz`, Go 1.25 |
| 2 | `internal/core/types.go` | ＋ `Asset`, `AssetRef`, `Dimensions`, `Origin`, `Sidecar` (§5.3) |
| 3 | `internal/core/errors.go` | ＋ errores sentinela con mensajes accionables (§5.7) |
| 4 | `internal/core/paths.go` | ＋ resolución y validación de rutas de proyecto (§5.8) — **incluye el guard de path traversal**, no se deja para después |
| 5 | `internal/config/config.go` | ＋ carga desde variables de entorno **únicamente** (§5.9) |
| 6 | `docs/arquitectura.md` | ＋ los seis principios de §1.2, en español |
| 7 | `.gitignore` | ＋ `*.meta.json` NO se ignora; sí se ignoran `/out`, `.env`, binarios |

Tests: `T-01`, `T-02` (§6).

### PR-1 — Pipeline determinista (~450 líneas)

El corazón del valor. Sin red, sin coste, sin proveedor. Al terminar esta unidad
el sistema ya sirve para algo aunque nunca se conecte a un modelo generativo.

| # | Archivo | Cambio |
|---|---|---|
| 1 | `internal/core/transform.go` | ＋ `Crop`, `Resize`, `Adjust` (brillo/contraste/saturación), `Rotate`, `Sharpen` |
| 2 | `internal/core/encode.go` | ＋ codificación a JPEG / PNG / WebP / AVIF con calidad configurable |
| 3 | `internal/core/export.go` | ＋ `ExportWeb`: genera el set de variantes responsive y devuelve el `srcset` (§5.10) |
| 4 | `internal/core/sidecar.go` | ＋ lectura y escritura del `.meta.json` (§5.4) |
| 5 | `testdata/golden/` | ＋ fixtures de referencia + flag `-update` |

Tests: `T-03` a `T-07` (§6).

### PR-2 — Proveedores, Gemini y guard de presupuesto (~500 líneas)

Deuda de seguridad **antes** que la feature que la consume (§3.2). El guard se
escribe y se prueba antes de que exista ninguna tool que pueda gastar.

| # | Archivo | Cambio |
|---|---|---|
| 1 | `internal/providers/provider.go` | ＋ interfaz `Provider` (§5.1) |
| 2 | `internal/providers/registry.go` | ＋ registro por nombre, resolución desde config |
| 3 | `internal/providers/fake/fake.go` | ＋ `fakeProvider` determinista: devuelve un PNG sólido derivado del hash del prompt, y **cuenta invocaciones**. Sin red. Es lo que usan todos los tests |
| 4 | `internal/providers/gemini/gemini.go` | ＋ proveedor real sobre `google.golang.org/genai` (§5.15) |
| 5 | `internal/providers/gemini/pricing.go` | ＋ tabla modelo+resolución → tokens → USD (§5.15) |
| 6 | `internal/budget/budget.go` | ＋ guard de presupuesto (§5.6) |
| 7 | `docs/proveedores.md` | ＋ relleno con la decisión de §3.1, con **fecha de consulta de precios** |

Los tests de esta unidad usan `fakeProvider`. El proveedor real se valida a mano
con la aceptación `A-03`, contra el tier gratuito, **sin tarjeta asociada**.

Tests: `T-08` a `T-10` (§6).

### PR-3 — Servidor MCP (~500 líneas)

Primer entregable que un LLM puede usar de verdad.

Estructura obligatoria del resultado — el implementador debe producir, en este
orden, todas estas piezas:

1. `cmd/matriz-mcp/main.go` con el arranque literal de §5.11.
2. Las cinco tools de §5.5, cada una con su struct `In`/`Out`, sus tags
   `jsonschema`, su descripción y sus `ToolAnnotations` según la tabla de §5.5.
3. El helper de miniatura de §5.12, usado por **todas** las tools que producen
   imagen. Ninguna tool construye su `ImageContent` a mano.
4. El mapeo de errores del core a `CallToolResult{IsError: true}` con contenido
   de texto accionable (§5.7). **Los errores de la tool no se devuelven como
   error de protocolo**: el LLM no los vería y no podría corregirse.
5. `docs/tools.md` en español: qué hace cada tool, cuándo usarla, y la tabla de
   decisión generativo-vs-determinista de §5.13 copiada literal.

| # | Archivo | Cambio |
|---|---|---|
| 1 | `cmd/matriz-mcp/main.go` | ＋ servidor stdio, registro de tools |
| 2 | `internal/mcpserver/tools_deterministic.go` | ＋ `img_transform`, `img_export_web` |
| 3 | `internal/mcpserver/tools_generative.go` | ＋ `img_generate_drafts`, `img_refine` |
| 4 | `internal/mcpserver/tools_meta.go` | ＋ `img_list_models` |
| 5 | `internal/mcpserver/thumbnail.go` | ＋ helper de §5.12 |
| 6 | `internal/mcpserver/errors.go` | ＋ mapeo error → `CallToolResult` |

Tests: `T-11` a `T-15`, más la aceptación manual `A-01` (§6).

### PR-4 — Manifiesto del sitio como resource (~300 líneas)

Sin esto el modelo genera un hero cuadrado para un contenedor 21:9, una y otra
vez. Es la unidad que hace que "el LLM tenga contexto suficiente" sea cierto.

| # | Archivo | Cambio |
|---|---|---|
| 1 | `internal/manifest/manifest.go` | ＋ lectura/escritura de `matriz.json` (§5.14) |
| 2 | `internal/manifest/scan.go` | ＋ reconstrucción del inventario desde disco + sidecars |
| 3 | `internal/mcpserver/resources.go` | ＋ `server.AddResource` para `matriz://project/manifest` |
| 4 | `docs/manifiesto.md` | ＋ el schema explicado en español |

Tests: `T-16` a `T-18` (§6).

### PR-5 — TUI (~400 líneas)

Sin lógica nueva. Sólo consume `internal/core` y `internal/manifest`.

| # | Archivo | Cambio |
|---|---|---|
| 1 | `cmd/matriz-tui/main.go` | ＋ arranque bubbletea |
| 2 | `internal/tui/model.go` | ＋ modelo, lista de assets, navegación |
| 3 | `internal/tui/keys.go` | ＋ bindings: `↑↓` navegar, `enter` abrir visor, `e` exportar, `q` salir |
| 4 | `internal/preview/html.go` | ＋ visor HTML local (decisión §3.3) |

Tests: `T-19` (§6).

### PR-6 — Proveedor fal (~300 líneas) — **DIFERIDO, no se ejecuta con el resto**

Esta unidad **no forma parte de v0.1.0**. Se escribe aquí para que la decisión
esté tomada y no haya que rediscutirla, pero no se implementa hasta que se
cumpla al menos uno de estos disparadores, que son comprobables:

1. **Volumen de fondos**: se han quitado más de 50 fondos con `img_refine` vía
   Gemini. A $0.134 cada uno son ~$6.70 que en fal habrían costado céntimos.
   Consultar el acumulado en los sidecars: `jq -s 'map(.cost_usd) | add' assets/*.meta.json`.
2. **Control de paleta**: un cliente exige que las imágenes generadas respeten
   códigos hexadecimales exactos de su identidad, y el prompting con Gemini no
   lo consigue de forma repetible.
3. **Calidad de recorte**: el matting de pelo de Gemini falla de forma visible
   en fotos de peluquería, que es el caso central del negocio.

Cuando se dispare, el trabajo es **un archivo nuevo**, no una refactorización:

| # | Archivo | Cambio |
|---|---|---|
| 1 | `internal/providers/fal/fal.go` | ＋ cliente HTTP a mano (fal no publica cliente de Go) |
| 2 | `internal/providers/fal/pricing.go` | ＋ tabla de coste; ojo: fal factura por segundo de cómputo en algunos modelos, así que `EstimateCostUSD` debe usar el **peor caso**, nunca la media |
| 3 | `internal/providers/registry.go` | ✎ registrar `fal` |
| 4 | `docs/proveedores.md` | ✎ añadir la comparativa y el disparador que se cumplió |

**Regla**: si `EstimateCostUSD` no puede acotar el coste de un modelo de fal
antes de llamarlo, ese modelo **no se expone**. Un proveedor que no se puede
presupuestar rompe §5.6.

---

## 5. Contratos exactos — copiar tal cual

> Los placeholders `<...>` se sustituyen. El resto **no se mejora sobre la
> marcha**.

### 5.1 Interfaz `Provider`

```go
package providers

import "context"

// Capability describes what a generative provider can do. The core uses this
// to route requests and to refuse impossible ones before spending money.
type Capability string

const (
	CapabilityGenerate    Capability = "generate"
	CapabilityInpaint     Capability = "inpaint"
	CapabilityOutpaint    Capability = "outpaint"
	CapabilityRemoveBG    Capability = "remove_background"
	CapabilityUpscale     Capability = "upscale"
	CapabilityDeterminism Capability = "seeded" // same seed + params => same image
)

// GenerateRequest is provider-agnostic. Provider implementations translate it
// into their own wire format. Never add a provider-specific field here.
type GenerateRequest struct {
	Prompt         string
	NegativePrompt string
	Width, Height  int
	Count          int    // number of drafts
	Seed           *int64 // nil => provider picks; the result MUST report it back
	Model          string // empty => provider default
}

// EditRequest covers inpaint / outpaint / background removal.
type EditRequest struct {
	Source    []byte // raw image bytes
	Mask      []byte // nil for operations that do not take a mask
	Prompt    string
	Operation Capability
	Seed      *int64
	Model     string
}

// Result carries the produced images plus everything needed to reproduce them.
type Result struct {
	Images   [][]byte
	MIMEType string
	Seed     int64
	Model    string
	CostUSD  float64 // provider's own accounting; 0 if unknown
}

// Provider is the only thing the core knows about image generation.
// Adding a provider must not require touching the core or the tools.
type Provider interface {
	Name() string
	Capabilities() []Capability
	// EstimateCostUSD is called BEFORE the request and must not hit the network.
	EstimateCostUSD(req GenerateRequest) float64
	Generate(ctx context.Context, req GenerateRequest) (*Result, error)
	Edit(ctx context.Context, req EditRequest) (*Result, error)
}
```

Por qué `EstimateCostUSD` no toca la red: el guard de presupuesto (§5.6) tiene
que poder rechazar una petición **antes** de gastarla.

### 5.2 Dependencias exactas de `go.mod`

```
require (
	github.com/modelcontextprotocol/go-sdk v1.7.0   // mínimo: soporta spec 2026-07-28
	github.com/disintegration/imaging v1.6.2        // resize/crop/adjust, pure Go
	github.com/gen2brain/webp <última>              // encode WebP sin CGo (WASM/wazero)
	github.com/gen2brain/avif <última>              // encode AVIF sin CGo (WASM/wazero)
	google.golang.org/genai <última>                // SDK oficial de Google, Apache 2.0; sólo PR-2
	github.com/charmbracelet/bubbletea <última>     // sólo PR-5
	github.com/charmbracelet/lipgloss <última>      // sólo PR-5
)
```

**Advertencia sin verificar**: ninguna de estas se compiló al redactar el
documento. La primera tarea de PR-0 es confirmarlo. Si `gen2brain/avif` falla o
pesa demasiado en el binario, la alternativa evaluada es `deepteams/webp` (pura
Go, cero CGo) para WebP, y **dejar AVIF fuera de v0.1.0** anotándolo en
`PENDING.md`. No inventar una tercera vía sin consultarlo.

### 5.3 Tipos core

```go
package core

type Origin string

const (
	OriginClient    Origin = "client"    // uploaded by the client; never modified in place
	OriginGenerated Origin = "generated" // produced by a generative model
	OriginDerived   Origin = "derived"   // deterministic transform of another asset
)

type Dimensions struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// AssetRef is the stable handle the LLM passes around. It is a project-relative
// path, never an absolute one — see hard rule 7.4.
type AssetRef string

type Asset struct {
	Ref      AssetRef   `json:"ref"`
	Origin   Origin     `json:"origin"`
	MIMEType string     `json:"mime_type"`
	Dims     Dimensions `json:"dims"`
	Bytes    int64      `json:"bytes"`
}
```

### 5.4 Schema del sidecar `.meta.json`

Se escribe junto a cada archivo producido: `hero-01.png` → `hero-01.png.meta.json`.

```json
{
  "schema": "matriz.sidecar/v1",
  "ref": "assets/hero-01.png",
  "origin": "generated",
  "created_at": "<RFC3339>",
  "provider": "<nombre del proveedor>",
  "model": "<id del modelo>",
  "prompt": "<prompt literal enviado>",
  "negative_prompt": "<o cadena vacía>",
  "seed": 0,
  "params": { "width": 1920, "height": 1080 },
  "cost_usd": 0.0,
  "derived_from": null
}
```

Para `origin: "derived"`, `provider`/`model`/`prompt`/`seed` van vacíos o a cero,
`derived_from` lleva el `AssetRef` del origen, y `params` lleva la operación
determinista aplicada. El campo `schema` es obligatorio y permite migrar el
formato más adelante sin adivinar.

### 5.5 Las cinco tools

Nombres con prefijo `img_` para que el agente las agrupe. Ninguna tool devuelve
la imagen a resolución completa en el `content` (regla dura §7.2).

| Tool | ReadOnly | Destructive | Idempotent | OpenWorld | Coste |
|---|---|---|---|---|---|
| `img_list_models` | `true` | — | — | `false` | 0 |
| `img_transform` | `false` | `false` | `true` | `false` | 0 |
| `img_export_web` | `false` | `false` | `true` | `false` | 0 |
| `img_generate_drafts` | `false` | `false` | `false` | `true` | **> 0** |
| `img_refine` | `false` | `false` | `false` | `true` | **> 0** |

`DestructiveHint` y `OpenWorldHint` son `*bool` en el SDK: hay que pasar la
dirección de una variable, no un literal.

Las descripciones de las tools **deben** llevar el marcador de coste en la
primera línea, porque es lo que el modelo lee para decidir:

```go
// Deterministic tools:
Description: "FREE and instant, no model call. Crop, resize, rotate, sharpen, "+
	"or adjust brightness/contrast/saturation of an existing asset. "+
	"Always prefer this over img_refine for anything a filter can do."

// Generative tools:
Description: "COSTS MONEY and takes seconds. Generates N new draft images from "+
	"a text prompt. Returns low-resolution previews you can look at. "+
	"Do NOT use for cropping, resizing, format conversion or colour "+
	"adjustment — use img_transform for those."
```

Structs de entrada y salida, literales:

```go
type TransformIn struct {
	Ref        string   `json:"ref" jsonschema:"project-relative path of the source asset"`
	Crop       *CropBox `json:"crop,omitempty" jsonschema:"optional crop box in pixels"`
	Width      int      `json:"width,omitempty" jsonschema:"target width in px; 0 keeps source width"`
	Height     int      `json:"height,omitempty" jsonschema:"target height in px; 0 preserves aspect ratio"`
	Brightness float64  `json:"brightness,omitempty" jsonschema:"-100..100, 0 is no change"`
	Contrast   float64  `json:"contrast,omitempty" jsonschema:"-100..100, 0 is no change"`
	Saturation float64  `json:"saturation,omitempty" jsonschema:"-100..100, 0 is no change"`
	Output     string   `json:"output" jsonschema:"project-relative path to write the result"`
}

type TransformOut struct {
	Asset      Asset  `json:"asset"`
	ThumbnailW int    `json:"thumbnail_width"`
	Note       string `json:"note" jsonschema:"human-readable summary of what was applied"`
}

type GenerateDraftsIn struct {
	Prompt         string `json:"prompt" jsonschema:"what to generate"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	Count          int    `json:"count" jsonschema:"number of drafts, 1..4"`
	AspectRatio    string `json:"aspect_ratio" jsonschema:"e.g. 16:9, 1:1, 21:9"`
	Slot           string `json:"slot,omitempty" jsonschema:"manifest slot id this image is for; fills dimensions automatically"`
	Seed           *int64 `json:"seed,omitempty" jsonschema:"omit for random; the response always reports the seed used"`
}

type GenerateDraftsOut struct {
	Drafts    []Asset `json:"drafts"`
	Seeds     []int64 `json:"seeds"`
	CostUSD   float64 `json:"cost_usd"`
	BudgetLeft float64 `json:"budget_left_usd"`
}
```

**Decisión de este cierre**: `img_generate_drafts` produce siempre a resolución
reducida (lado mayor 768 px, configurable). El escalado a resolución final es una
llamada aparte. Motivo: generar cuatro borradores a 1920 px cuesta cuatro veces
más que generarlos a 768 y escalar sólo el elegido.

### 5.6 Guard de presupuesto

```go
package budget

// Guard enforces a per-session spending ceiling. It is consulted BEFORE any
// paid call and it fails closed: unknown cost counts as the configured
// worst-case, never as zero.
type Guard struct {
	limitUSD float64
	spentUSD float64
	calls    int
	maxCalls int
}

// Reserve returns an error if the estimated cost would exceed the ceiling.
// The error message MUST state the limit, the amount already spent and the
// estimate, so the agent can decide instead of retrying blindly.
func (g *Guard) Reserve(estimateUSD float64) error

// Commit records actual spend after a successful call.
func (g *Guard) Commit(actualUSD float64)
```

Defaults (**decisión de este cierre**, cambiables por variable de entorno):
`MATRIZ_BUDGET_USD=2.00` por sesión, `MATRIZ_MAX_GENERATIVE_CALLS=20`.

Calibración con los precios reales de §2.2, para que el número signifique algo:

| Escenario | Coste unitario | Cuántos entran en $2.00 |
|---|---|---|
| Borradores en Lite (1K) | $0.0336 | ~59 |
| Imagen final en Pro (2K) | $0.134 | ~14 |
| Mezcla realista: 8 borradores Lite + 3 finales Pro | — | ~$0.67, sobra presupuesto |

Con el flujo de dos pasos (borradores baratos → escalar sólo el elegido), $2.00
cubre de sobra una landing completa. Si se generase todo directamente en Pro a
4K ($0.24), el tope saltaría a las 8 imágenes — que es exactamente el
comportamiento que el guard debe cortar.

Al agotarse, la tool devuelve `IsError: true` con el texto:

```
Budget exhausted: spent $<gastado> of $<limite> across <n> calls.
Raise MATRIZ_BUDGET_USD and restart the server, or use img_transform
(free) if the change you want is deterministic.
```

### 5.7 Errores accionables

Cada error del core lleva sugerencia y siguiente paso. Plantilla obligatoria:

```
<qué falló>: <por qué>.
<qué hacer en su lugar>.
```

Ejemplos literales a usar:

```
Asset not found: "assets/hero.png" does not exist in this project.
Call the matriz://project/manifest resource to see available assets.

Aspect ratio mismatch: slot "hero" expects 21:9 but the asset is 1:1.
Use img_transform with a crop box, or regenerate with aspect_ratio "21:9".

Provider does not support inpainting: "<nombre>" reports capabilities [generate, upscale].
Remove the mask argument, or switch provider with MATRIZ_PROVIDER.
```

### 5.8 Validación de rutas (guard de path traversal)

```go
// ResolveRef converts a project-relative AssetRef into an absolute filesystem
// path, and refuses anything that escapes the project root.
//
// This runs on EVERY ref that arrives from the MCP boundary, including refs
// that came back from a previous tool call. An LLM-supplied path is untrusted
// input.
func ResolveRef(projectRoot string, ref AssetRef) (string, error)
```

Rechaza: rutas absolutas, `..` en cualquier segmento, symlinks que salgan de la
raíz (comprobar con `filepath.EvalSymlinks` después de resolver), y cadenas
vacías. Mensaje de error: `invalid asset ref "<ref>": paths must stay inside the project root`.

### 5.9 Configuración — sólo entorno

```
MATRIZ_PROVIDER=gemini            # único proveedor en v0.1.0
GOOGLE_API_KEY=<clave>            # nombre que espera el SDK oficial de Google
MATRIZ_PROJECT_ROOT=<ruta>        # raíz del proyecto web
MATRIZ_MODEL_DRAFT=<id-lite>      # p. ej. el ID de Gemini Flash-Lite Image
MATRIZ_MODEL_FINAL=<id-pro>       # p. ej. gemini-3-pro-image-preview
MATRIZ_BUDGET_USD=2.00
MATRIZ_MAX_GENERATIVE_CALLS=20
MATRIZ_DRAFT_MAX_EDGE=768
```

**Los IDs de modelo son configuración, nunca constantes en el código.** Motivo:
`gemini-3-pro-image-preview` lleva `-preview`, y los modelos en preview se
deprecan sin ceremonia. Si el ID vive en una constante de Go, la deprecación es
una recompilación; si vive en el entorno, es una línea. Ver regla dura §7.11.

La clave se llama `GOOGLE_API_KEY` y no `MATRIZ_API_KEY` porque es el nombre que
el SDK de Google lee por convención. No renombrarla: reinventar el nombre obliga
a pasarla a mano y multiplica los sitios por donde puede escaparse.

No hay archivo de configuración en v0.1.0. Motivo: un archivo de config es el
sitio donde termina apareciendo una API key commiteada.

### 5.10 Contrato de `ExportWeb`

Dado un asset y un slot, produce:

```
<basename>-<ancho>w.<ext>   para cada ancho del set
```

Set de anchos por defecto (**decisión de este cierre**): `[420, 768, 1024, 1440, 1920]`,
recortado a los que no superen el ancho del original — nunca se escala hacia
arriba, porque agrandar no añade información y engorda la página.

Formatos por defecto: `avif` y `webp`, más el original como fallback.

Devuelve el `srcset` ya montado:

```
assets/hero-420w.webp 420w, assets/hero-768w.webp 768w, assets/hero-1024w.webp 1024w
```

y también un campo `sizes_hint` con el valor sugerido para el atributo `sizes`,
tomado del slot del manifiesto si existe.

### 5.11 Arranque del servidor MCP — literal

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "matriz",
		Version: "v0.1.0",
	}, nil)

	registerDeterministicTools(server)
	registerGenerativeTools(server)
	registerMetaTools(server)
	registerResources(server)

	// stdio only. No HTTP transport in v0.1.0.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
	_ = os.Stdout // stdout belongs to the protocol: never print to it.
}
```

**Crítico**: en transporte stdio, `stdout` es el canal del protocolo. Cualquier
`fmt.Println` de depuración corrompe la sesión. Todo log va a `stderr`.

### 5.12 Helper de miniatura — literal

```go
// thumbnailContent builds the ImageContent every image-producing tool returns.
// Full-resolution bytes NEVER travel through the protocol: they blow up the
// context window and cost tokens for no benefit.
//
// maxEdge is 512 px by default (decision of this handoff): large enough for a
// model to judge composition, colour and obvious artefacts; small enough that
// four of them in one response stay manageable.
func thumbnailContent(img image.Image, maxEdge int) (*mcp.ImageContent, error) {
	small := imaging.Fit(img, maxEdge, maxEdge, imaging.Lanczos)
	var buf bytes.Buffer
	if err := png.Encode(&buf, small); err != nil {
		return nil, err
	}
	return &mcp.ImageContent{
		Data:     buf.Bytes(), // encoding/json base64-encodes []byte — VERIFY with the Inspector
		MIMEType: "image/png",
	}, nil
}
```

⚠️ La línea marcada es la única suposición del contrato: el campo está
documentado como `// base64-encoded` en `mcp/content.go`, y `ImageContent` tiene
`MarshalJSON` propio. Confirmar en `A-01` que el cliente muestra la imagen. Si
no, revisar `mcp/content.go` en la versión que hayas fijado.

### 5.13 Tabla de decisión generativo vs determinista

Copiar literal en `docs/tools.md` y resumida en las descripciones de las tools.

| Lo que se quiere | Tool | Coste |
|---|---|---|
| Recortar, encuadrar de nuevo | `img_transform` | 0 |
| Cambiar tamaño | `img_transform` | 0 |
| Corregir luz, contraste, saturación | `img_transform` | 0 |
| Convertir a WebP/AVIF | `img_export_web` | 0 |
| Generar variantes responsive y `srcset` | `img_export_web` | 0 |
| Rotar, enderezar | `img_transform` | 0 |
| Crear una imagen que no existe | `img_generate_drafts` | **paga** |
| Quitar o sustituir el fondo | `img_refine` | **paga** |
| Borrar un objeto de la foto | `img_refine` (inpaint) | **paga** |
| Ampliar el encuadre más allá del borde | `img_refine` (outpaint) | **paga** |

### 5.14 Schema del manifiesto `matriz.json`

```json
{
  "schema": "matriz.manifest/v1",
  "project": "<nombre del sitio>",
  "palette": ["#0b0b0b", "#c8a45c", "#f4f1ea"],
  "slots": [
    {
      "id": "hero",
      "usage": "portada, primera pantalla",
      "aspect_ratio": "21:9",
      "min_width": 1920,
      "sizes_hint": "100vw",
      "asset": "assets/hero-01.avif",
      "alt": "<texto alternativo>"
    }
  ],
  "assets": [
    {
      "ref": "assets/hero-01.avif",
      "origin": "generated",
      "mime_type": "image/avif",
      "dims": { "width": 1920, "height": 823 },
      "bytes": 84210
    }
  ]
}
```

El resource se expone como:

```go
server.AddResource(&mcp.Resource{
	URI:         "matriz://project/manifest",
	Name:        "project manifest",
	Description: "Slots, palette and full asset inventory of the current site. Read this BEFORE generating anything.",
	MIMEType:    "application/json",
}, manifestHandler)
```

---

### 5.15 Proveedor Gemini — contrato

**Cliente.** El SDK oficial se instala con `go get google.golang.org/genai` y se
inicializa así (literal, verificado contra el README del SDK):

```go
client, err := genai.NewClient(ctx, &genai.ClientConfig{
	APIKey:  apiKey,
	Backend: genai.BackendGeminiAPI, // Developer API, no Vertex/Enterprise
})
```

**Generación.** La imagen sale de una llamada multimodal normal, no de un
endpoint aparte: `client.Models.GenerateContent(ctx, modelID, contents, cfg)`.
El `modelID` viene de `MATRIZ_MODEL_DRAFT` o `MATRIZ_MODEL_FINAL` según el paso.
La imagen de referencia, cuando la hay (edición), viaja como parte del contenido:

```go
parts := []*genai.Part{
	{Text: instruction},
	{InlineData: &genai.Blob{Data: sourceBytes, MIMEType: "image/jpeg"}},
}
```

**Tabla de coste — `pricing.go`.** Esto es lo que hace que `EstimateCostUSD`
funcione sin tocar la red. Rellenar con los precios **verificados en la página
oficial el día de la implementación**, no con los de este documento:

```go
// tokensPerImage maps a model+resolution pair to its FIXED output-token cost.
// Google bills images as output tokens, but the count per image is fixed, so a
// pre-flight estimate is exact rather than a guess. Verified figures as of
// <fecha de consulta>:
//   standard (1K/2K) => ~1120 output tokens
//   4K               => ~2000 output tokens
// Input cost per image is ~$0.0011 and is folded into the constant below.
var tokensPerImage = map[modelResolution]int{ /* ... */ }

// usdPerMillionOutputTokens must be read from the official pricing page.
// If a model is missing from this table, EstimateCostUSD returns the
// worst-case price in the table — never zero. Failing open on cost is how
// budgets get blown.
```

**Regla de fallo cerrado**: un modelo que no esté en la tabla se estima al
**peor caso conocido**, nunca a cero. Un coste desconocido tratado como gratis
es exactamente el fallo que el guard existe para impedir.

**Semillas.** Si el modelo no acepta `seed`, `Result.Seed` se rellena con `0` y
el sidecar anota `"seed": 0` con `"params": {"seeded": false}`. **No se inventa
una seed** para que el JSON quede bonito: un sidecar que promete
reproducibilidad que no existe es peor que uno honesto.

**Marca de agua.** Todas las salidas llevan SynthID invisible. Además, según la
ruta de cuenta, pueden llevar marca **visible**. El proveedor no puede
detectarlo por sí solo, así que la verificación es manual y es la aceptación
`A-03`.

### 5.16 `docs/proveedores.md` — contenido mínimo

Se crea en PR-2 con esta estructura exacta:

```markdown
# Proveedores de imagen

## Activo: Google Gemini (familia Nano Banana)
- SDK: google.golang.org/genai (oficial, Apache 2.0)
- Modelo de borrador: <id> — <coste> — <resolución máxima>
- Modelo final: <id> — <coste> — <resolución máxima>
- Precios consultados el: <fecha> en <URL oficial>
- Uso comercial: <cláusula citada literalmente de los términos>
- Marca de agua: SynthID invisible siempre. Visible según ruta de cuenta —
  verificado el <fecha> con resultado: <con marca | sin marca>
- Acepta seed: <sí | no>
- Soporta máscara de inpainting: <sí | no>

## Diferido: fal.ai
- Disparador para integrarlo: ver PR-6 en el handoff
- Motivo de la espera: no hay cliente oficial de Go
```

El campo de fecha de consulta no es burocracia: es lo que permite saber, dentro
de seis meses, si la tabla de coste sigue siendo válida.

---

## 6. Tests y criterios de aceptación

| ID | PR | Tipo | Aserción |
|---|---|---|---|
| T-01 | PR-0 | compilación | Un `main` que importa las 5 librerías de §5.2 compila y ejecuta en la máquina destino |
| T-02 | PR-0 | unitario | `ResolveRef` rechaza `../etc/passwd`, `/etc/passwd`, `""` y un symlink que apunta fuera de la raíz; acepta `assets/a.png` |
| T-03 | PR-1 | golden | `Resize` de un fixture 1000×500 a ancho 400 produce bytes idénticos al golden |
| T-04 | PR-1 | golden | `Adjust` con brillo +20 produce bytes idénticos al golden |
| T-05 | PR-1 | unitario | `ExportWeb` sobre un original de 900 px de ancho genera exactamente los anchos `[420, 768]` y **ninguno mayor que el original** |
| T-06 | PR-1 | unitario | El `srcset` devuelto tiene formato `<ruta> <n>w` separado por `, ` y los anchos en orden ascendente |
| T-07 | PR-1 | unitario | Escribir y releer un sidecar preserva todos los campos; falta de `schema` es error |
| T-08 | PR-2 | unitario | `fakeProvider.Generate` con la misma seed devuelve bytes idénticos en dos llamadas |
| T-09 | PR-2 | unitario | `Guard.Reserve` rechaza cuando `spent + estimate > limit`, y el mensaje de error contiene los tres números |
| T-10 | PR-2 | unitario | `Guard` rechaza la llamada número `maxCalls+1` aunque quede presupuesto |
| T-10b | PR-2 | unitario | `EstimateCostUSD` de un modelo **ausente** de la tabla de precios devuelve el peor caso conocido, nunca `0` |
| T-10c | PR-2 | unitario | `EstimateCostUSD` no abre ninguna conexión de red (test con transporte HTTP que falla si se usa) |
| T-10d | PR-2 | unitario | Un resultado sin seed produce un sidecar con `"seed": 0` y `"seeded": false`, no una seed inventada |
| T-11 | PR-3 | unitario | `img_transform` con un `ref` inexistente devuelve `CallToolResult.IsError == true`, **no** un error de Go propagado al protocolo |
| T-12 | PR-3 | unitario | Toda tool que produce imagen devuelve al menos un `*mcp.ImageContent` en `Content` |
| T-13 | PR-3 | unitario | La miniatura devuelta tiene lado mayor ≤ 512 px |
| T-14 | PR-3 | unitario | `img_generate_drafts` con el guard agotado no llama al provider (el fake cuenta invocaciones y debe quedar en 0) |
| T-15 | PR-3 | estructural | `grep` confirma que la descripción de cada tool generativa contiene `COSTS MONEY` y la de cada determinista contiene `FREE` |
| T-16 | PR-4 | unitario | `scan` reconstruye el inventario desde disco y coincide con el manifiesto escrito |
| T-17 | PR-4 | unitario | Un asset sin campo `origin` hace fallar la validación del manifiesto |
| T-18 | PR-4 | unitario | El handler del resource devuelve JSON que valida contra el schema `matriz.manifest/v1` |
| T-19 | PR-5 | unitario | El modelo de la TUI se construye desde un manifiesto de fixture sin tocar disco fuera de `testdata/` |

### Aceptación manual final

**A-01** (al cerrar PR-3): arrancar `npx @modelcontextprotocol/inspector` contra
el binario, y verificar en la interfaz:

1. Las cinco tools aparecen con sus descripciones y sus annotations.
2. `img_transform` sobre un fixture devuelve una miniatura **visible** en el
   inspector. Si sólo se ve un blob de base64, el contrato §5.12 está mal y hay
   que corregirlo antes de seguir.
3. El output estructurado aparece además de la imagen, no en su lugar.

**A-02** (al cerrar PR-4): abrir Claude Code con el servidor conectado y pedir
*"exportá el hero para el sitio"* sin dar más contexto. El agente debe leer el
resource del manifiesto antes de actuar. Si no lo lee, la descripción del
resource es insuficiente y hay que reforzarla.

**A-03** (al cerrar PR-2, contra el proveedor real): con la clave del tier
gratuito de Google AI Studio y **sin tarjeta asociada a la cuenta**:

1. Generar una imagen con `MATRIZ_MODEL_DRAFT` y abrirla a tamaño completo.
2. **Mirar las esquinas.** Si aparece la marca visible de Gemini, la ruta de
   cuenta es la equivocada: anotarlo en `docs/proveedores.md` y en `PENDING.md`,
   y **no entregar ninguna imagen a un cliente por esa ruta** hasta resolverlo.
   Este es el fallo silencioso de §3.1-bis: el prototipo parece correcto y el
   problema sólo aparece en producción.
3. Comprobar que el coste real facturado por Google, cuando aparezca en la
   consola, coincide con lo que estimó `EstimateCostUSD`. Si difiere más de un
   10 %, la tabla de `pricing.go` está mal y hay que corregirla antes de PR-3.

---

## 7. Reglas duras — NO tocar, NO "mejorar"

1. **Una operación determinista jamás llama a un proveedor generativo.** Si
   parece más cómodo unificar `img_transform` e `img_refine` en una sola tool,
   no se hace. Es el principio P1 y es la razón de ser del diseño.
2. **Nunca se devuelve la imagen a resolución completa en el `content` de una
   tool.** Sólo miniaturas. El archivo grande vive en disco y se referencia por
   `AssetRef`.
3. **Ninguna API key se escribe a disco, se loguea, se incluye en un mensaje de
   error, ni aparece en el repositorio.** Sólo variable de entorno, sólo en
   memoria. Si un error necesita mencionar el proveedor, menciona su nombre, no
   su credencial.
4. **Todo `ref` que cruza la frontera MCP pasa por `ResolveRef`**, incluidos los
   que devolvió una tool anterior. Un path que viene de un LLM es entrada no
   confiable, siempre.
5. **Ningún archivo de entrada del cliente se modifica in-place ni se borra.**
   Todo resultado se escribe como archivo nuevo. No hay tool de borrado en
   v0.1.0 y no se añade por iniciativa propia.
6. **Todo asset lleva `origin` obligatorio**, y una imagen `generated` nunca se
   escribe en un slot cuyo `usage` la presente como cliente, trabajo real o
   testimonio del negocio. Si un slot es ambiguo, se pregunta, no se asume.
7. **`stdout` pertenece al protocolo.** Ningún `fmt.Print*` fuera de stderr en
   `cmd/matriz-mcp`.
8. **No se construye nada sobre roots, sampling ni logging de MCP.** Están
   deprecados desde la spec `2026-07-28` (SEP-2577) y el soporte del SDK es una
   ventana de compatibilidad, no una base.
9. **El core no importa nada de `cmd/` ni de `internal/mcpserver`.** La
   dependencia va en una sola dirección. Si hace falta al revés, el diseño está
   mal y hay que parar.
10. **Documentación en español, código en inglés.** No se traduce el código ni
    se anglifica la documentación.
11. **Los IDs de modelo viven en configuración, nunca en constantes de Go.**
    `gemini-3-pro-image-preview` es un ID en preview y se va a deprecar. Si está
    hardcodeado, la deprecación obliga a recompilar y redistribuir.
12. **Un coste desconocido nunca se estima como cero.** Modelo ausente de la
    tabla de precios ⇒ peor caso conocido. Fallar abierto en coste es cómo se
    revienta un presupuesto.
13. **No se inventan seeds.** Si el proveedor no devuelve una, el sidecar dice
    `"seeded": false`. Un sidecar que promete reproducibilidad inexistente es
    peor que uno honesto.
14. **Ninguna imagen llega a un entregable de cliente sin que `A-03` haya
    confirmado la ausencia de marca visible** por esa ruta de cuenta.
15. **No se añade un segundo proveedor "de paso" mientras se implementa otra
    cosa.** fal entra por PR-6, con su disparador cumplido y anotado. Añadir
    proveedores por impulso es cómo la capa de abstracción se llena de casos
    especiales y deja de abstraer.

---

## 8. Mapa de archivos afectados

`✎` edita · `＋` crea · `⌕` sólo verifica

| Archivo | PR-1 | PR-2 | PR-3 | PR-4 | PR-5 | PR-6 |
|---|---|---|---|---|---|---|
| `internal/core/types.go` | ✎ | ⌕ | ⌕ | ✎ | ⌕ | |
| `internal/core/transform.go` | ＋ | | ⌕ | | ⌕ | |
| `internal/core/encode.go` | ＋ | | ⌕ | | | |
| `internal/core/export.go` | ＋ | | ⌕ | ✎ | ⌕ | |
| `internal/core/sidecar.go` | ＋ | | | ⌕ | | |
| `internal/core/paths.go` | ⌕ | | ⌕ | | | |
| `internal/providers/provider.go` | | ＋ | ⌕ | | | ⌕ |
| `internal/providers/registry.go` | | ＋ | ✎ | | | ✎ |
| `internal/providers/fake/fake.go` | | ＋ | ⌕ | | | |
| `internal/providers/gemini/gemini.go` | | ＋ | ⌕ | | | ⌕ |
| `internal/providers/gemini/pricing.go` | | ＋ | | | | ⌕ |
| `internal/providers/fal/` | | | | | | ＋ |
| `internal/budget/budget.go` | | ＋ | ✎ | | | ⌕ |
| `internal/mcpserver/*.go` | | | ＋ | ✎ | | |
| `internal/manifest/*.go` | | | | ＋ | ⌕ | |
| `internal/tui/*.go` | | | | | ＋ | |
| `internal/preview/html.go` | | | | | ＋ | |
| `cmd/matriz-mcp/main.go` | | | ＋ | ✎ | | |
| `cmd/matriz-tui/main.go` | | | | | ＋ | |
| `docs/tools.md` | | | ＋ | ✎ | | |
| `docs/manifiesto.md` | | | | ＋ | | |
| `docs/proveedores.md` | | ＋ | | | | ✎ |

PR-0 queda fuera de la matriz. Sus archivos: `go.mod`, `.gitignore`,
`internal/core/types.go`, `internal/core/errors.go`, `internal/core/paths.go`,
`internal/config/config.go`, `docs/arquitectura.md`.

**PR-6 está en la matriz sólo como referencia**: no se ejecuta en v0.1.0. Su
columna muestra qué se tocaría el día que se cumpla su disparador, y sirve para
comprobar que la abstracción aguanta — si añadir fal exigiera editar el core o
las tools, el diseño de `Provider` habría fallado.

---

## 9. Definition of Done

- [ ] `go build ./...` y `go vet ./...` limpios en cada PR
- [ ] `go test ./...` en verde al cerrar cada una de las seis unidades de v0.1.0 (PR-0 a PR-5), cero fallos
- [ ] Las 23 aserciones de §6 implementadas y pasando
- [ ] `A-01` superada: el MCP Inspector **muestra la miniatura como imagen**, no como base64 crudo
- [ ] `A-02` superada: un agente sin contexto previo lee el resource del manifiesto antes de generar
- [ ] `A-03` superada: imagen real generada contra Gemini, **esquinas revisadas** y ausencia de marca visible confirmada y anotada con fecha
- [ ] El coste real facturado por Google coincide con `EstimateCostUSD` dentro de un 10 %
- [ ] Test de autosuficiencia: un agente sin acceso a esta conversación puede ejecutar `img_export_web` sobre un asset y obtener un `srcset` correcto **sin improvisar** ningún parámetro
- [ ] `git grep -iE "sk-|api[_-]key *=" ` no devuelve nada en el repositorio
- [ ] Ningún ID de modelo aparece como constante en el código (`git grep "gemini-"` sólo devuelve documentación y valores por defecto de config)
- [ ] Cada dependencia de §5.2 está fijada a una versión concreta en `go.sum`, no a `latest`
- [ ] `docs/proveedores.md` relleno según §5.16, **con fecha de consulta de precios**
- [ ] `PENDING.md` existe y recoge: AVIF si se descartó, renderizado nativo en terminal, resultado de la revisión de marca de agua, y cualquier divergencia encontrada
- [ ] Los seis principios de §1.2 están en `docs/arquitectura.md` en español
- [ ] PR-6 **no** está implementado (es correcto que falte: v0.1.0 cierra sin fal)

---

> **Nota final para el implementador**: si durante la implementación descubrís
> que el SDK, una librería o un proveedor se comporta distinto a lo afirmado
> aquí —especialmente el contrato de `ImageContent` de §5.12 y los precios de
> §2.2, marcados como no verificados o sujetos a cambio— **no adaptes en
> silencio**. Capturá la evidencia (el
> output real, el error literal, la versión exacta), anotala en `PENDING.md` con
> su condición, y decidilo con el usuario. Este documento se escribió contra un
> estado; la realidad manda sobre el documento.
