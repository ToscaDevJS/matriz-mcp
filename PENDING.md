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
- **Estado**: Resuelto el 2026-09-03.
- **Hallazgo**: `gen2brain/avif` (libaom compilado a WASM sobre `wazero` sin SIMD nativo) generaba un cuello de botella de ~62s secuenciales contra 30ms de WebP (~2000x más lento).
- **Resolución**: Se removió completamente la dependencia `gen2brain/avif` y `tetratelabs/wazero`. Matriz ahora rechaza explícitamente cualquier solicitud de codificación AVIF en `core.ParseFormat` e `img_transform` con un mensaje accionable que instruye el uso de `.webp`, `.png` o `.jpg`/`.jpeg`. Los diagnósticos de `matriz doctor` verifican la terna operativa: PNG, JPEG y WebP.

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

## 6. `img_refine` no escribía sidecar
- **Estado**: Resuelto el 2026-09-02.
- **Contexto**: `handleRefine` escribía el archivo de salida con `os.WriteFile`, pero nunca
  llamaba a `core.WriteSidecar`. Toda imagen refinada quedaba en disco sin su `.meta.json`,
  violando §5.4 justo en la operación que cuesta dinero.
- **Decisión de modelado**: se agregó `providers.BuildEditSidecar`. Un refinamiento lleva
  `origin: generated`, no `derived`, porque `derived` está definido en `core/types.go` como
  «transformación determinista de otro asset» y un refinamiento lo produce un modelo
  generativo. La cadena hacia el original se registra en `derived_from`, que es un campo
  independiente de `origin`.
- **Corregido en el mismo paso**: `handleRefine` indexaba `result.Images[0]` sin verificar
  el largo y descartaba el error de `image.Decode` antes de llamar `decoded.Bounds()`. Ambos
  eran panics; un panic en un handler stdio se lleva puesto el servidor en vez de llegar al
  modelo (regla 7.9). Ahora devuelven `CallToolResult{IsError: true}`.
