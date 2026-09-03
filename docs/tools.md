# Herramientas MCP de Matriz

Matriz expone cuatro herramientas estructuradas agrupadas bajo el prefijo `img_`.

## Tabla de decisión: generativo vs determinista

| Lo que se quiere | Tool | Coste |
|---|---|---|
| Recortar, encuadrar de nuevo | `img_transform` | 0 |
| Cambiar tamaño | `img_transform` | 0 |
| Corregir luz, contraste, saturación | `img_transform` | 0 |
| Convertir a WebP / JPEG / PNG | `img_transform` | 0 |
| Rotar, enderezar | `img_transform` | 0 |
| Crear una imagen que no existe | `img_generate_drafts` | **paga** |
| Quitar o sustituir el fondo | `img_refine` | **paga** |
| Borrar un objeto de la foto | `img_refine` (inpaint) | **paga** |
| Ampliar el encuadre más allá del borde | `img_refine` (outpaint) | **paga** |

> Para una guía completa del ciclo de vida y mejores prácticas para agentes, ver [docs/workflow.md](workflow.md).

## Detalle de herramientas

### `img_list_models`
- **Coste**: Gratuito (0).
- **Uso**: Consulta el proveedor activo, modelos configurados, llamadas realizadas y estado de presupuesto restante.

### `img_transform`
- **Coste**: Gratuito (0) e instantáneo.
- **Uso**: Operaciones locales de transformación (recorte, redimensionado, ajustes de brillo/contraste/saturación, rotación y enfoque) sin conexión a la red. El formato de salida se deduce de la extensión de `output`, soportando conversión de alta velocidad a JPEG, PNG y WebP.

### `img_generate_drafts`
- **Coste**: De pago (facturado por tokens del proveedor).
- **Uso**: Genera de 1 a 4 borradores a resolución reducida (máx. 768 px) a partir de un prompt textual. Retorna miniaturas para inspección visual directa.

### `img_refine`
- **Coste**: De pago (facturado por tokens del proveedor).
- **Uso**: Operaciones multimodales generativas (inpainting con máscara, outpainting o eliminación de fondo).
