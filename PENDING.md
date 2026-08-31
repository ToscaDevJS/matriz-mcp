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
