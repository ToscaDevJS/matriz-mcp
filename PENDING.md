# Matriz — Elementos pendientes y seguimiento (v0.1.0)

Registro de decisiones técnicas, verificaciones manuales y alcance diferido conforme al handoff de implementación.

## 1. Verificación de marca de agua visible (A-03)
- **Estado**: Pendiente de ejecución manual con clave de Google AI Studio sin tarjeta.
- **Acción**: Antes de entregar imágenes finales a clientes, verificar esquinas de imágenes generadas para asegurar ausencia de marcas de agua visibles.

## 2. Renderizado gráfico nativo en terminal (TUI)
- **Estado**: Resuelto en v0.1.0 mediante apertura de visor HTML local en navegador al presionar `enter` (`internal/preview/html.go`).
- **Follow-up**: Soporte nativo de protocolos Kitty / Sixel / iTerm2 diferido a hitos posteriores.

## 3. Integración de proveedor fal.ai (PR-6)
- **Estado**: Diferido a cuando se cumpla al menos uno de los disparadores objetivos (§4 PR-6):
  1. Más de 50 fondos eliminados con Gemini.
  2. Requisito estricto de colores de marca exactos en hexadecimal.
  3. Necesidad de recorte de pelo fino superior con BiRefNet.

## 4. Codecs AVIF y WebP
- **Estado**: Compilados y enlazados exitosamente vía WASM / pure Go (`gen2brain/avif` y `gen2brain/webp`) sin requerir dependencias de sistema CGo.
- **Hallazgo (2026-09-02)**: compilan, pero el rendimiento de AVIF lo vuelve inviable para
  iteración. Medición sobre una imagen de 1408x768 con `Quality: 80`:

  | Ancho | AVIF | WebP |
  |---|---|---|
  | 420w | 6.598s | 4ms |
  | 768w | 21.791s | 10ms |
  | 1024w | 33.939s | 16ms |
  | **Total secuencial** | **1m02.368s** | **30ms** |

  La causa es que `gen2brain/avif` ejecuta libaom compilado a WASM sobre `wazero`
  (sin SIMD nativo). WebP, con el mismo enfoque, no muestra el problema.
- **Decisión tomada**: se eliminó la herramienta `img_export_web` de v0.1.0 (ver CHANGELOG).
- **Riesgo abierto**: AVIF sigue alcanzable a través de `img_transform`, que infiere el
  formato de salida desde la extensión del archivo. Un `output` terminado en `.avif`
  bloquea la llamada MCP durante más de 30 segundos. No hay aviso al modelo.
- **Acción**: decidir si `img_transform` debe rechazar AVIF, advertirlo en su descripción,
  o si conviene sustituir el codec por uno nativo.

## 5. `ImageConfig` y `CandidateCount` en generación de imágenes de Gemini
- **Estado**: Asunción sin verificar contra la API real.
- **Contexto**: `GenerateContentConfig.CandidateCount` (SDK `google.golang.org/genai` v1.70.0,
  `types.go:2954`) ya se envía con el número de borradores solicitado. No se pudo comprobar
  si los modelos de imagen lo respetan, porque hacerlo requiere clave y red.
- **Mitigación aplicada**: el costo liquidado se calcula sobre `len(result.Images)`, no sobre
  `req.Count`. Si Gemini ignora `CandidateCount` y devuelve una sola imagen, se cobra una sola
  imagen. El presupuesto no se ve afectado en ninguno de los dos escenarios.
- **Contexto adicional**: `GenerateContentConfig.ImageConfig` (`types.go:3041`) también se
  envía ahora, con `AspectRatio` y `ImageSize` derivados de las dimensiones solicitadas.
  Tampoco se pudo comprobar contra la API real si el modelo respeta ambos campos.
- **Mitigación aplicada**: el sidecar y el `Asset` devuelto al modelo registran las
  dimensiones **producidas**, decodificadas del archivo real. Si el modelo ignora
  `AspectRatio`, la discrepancia queda visible en el registro en vez de silenciada.
- **Acción**: ejecutar `img_generate_drafts` con `count: 4` y `aspect_ratio: "21:9"` con una
  clave real. Registrar cuántas imágenes devuelve y con qué dimensiones. Si devuelve una
  sola, evaluar N llamadas en paralelo; si ignora el ratio, revisar el mapeo de
  `nearestAspectRatio`.

## 6. `img_refine` no escribe sidecar
- **Estado**: Detectado el 2026-09-02, sin corregir.
- **Contexto**: `handleRefine` (`internal/mcpserver/tools_generative.go`) escribe el archivo
  de salida con `os.WriteFile`, pero nunca llama a `core.WriteSidecar`. Toda imagen refinada
  queda en disco sin su `.meta.json`.
- **Impacto**: viola §5.4 («todo archivo producido escribe su sidecar»). Se pierde la
  trazabilidad de proveedor, modelo, prompt, seed y costo de cada refinamiento, que es
  justamente la operación que cuesta dinero.
- **Acción**: decidir el alcance del arreglo. `BuildSidecar` asume `origin: generated`; un
  refinamiento parte de un asset existente, así que probablemente corresponda
  `origin: derived` con `derived_from` apuntando al origen.
