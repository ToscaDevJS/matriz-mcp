# Herramientas MCP de Matriz

Matriz expone ocho herramientas estructuradas agrupadas bajo los dominios de imagen (`img_`) y video asíncrono (`video_`), además de dos recursos MCP.

## Tabla de decisión: generativo vs determinista

| Lo que se quiere | Tool | Tipo | Coste |
|---|---|:---:|:---:|
| Recortar, encuadrar de nuevo | `img_transform` | Local | **FREE ($0)** |
| Cambiar tamaño de imagen | `img_transform` | Local | **FREE ($0)** |
| Corregir luz, contraste, saturación | `img_transform` | Local | **FREE ($0)** |
| Convertir a WebP / JPEG / PNG | `img_transform` | Local | **FREE ($0)** |
| Rotar, enderezar | `img_transform` | Local | **FREE ($0)** |
| Crear borradores rápidos de imagen (1..4) | `img_generate_drafts` | Generativo | **Paga (Tokens)** |
| Quitar fondo, inpaint o outpaint compositivo | `img_refine` | Generativo | **Paga (Tokens)** |
| Elevar borrador a calidad Pro de producción | `img_upscale` | Generativo | **Paga (Tokens)** |
| Generar video desde prompt de texto (Async) | `video_generate` | Generativo | **Paga ($/seg)** |
| Animar imagen existente a video (Async) | `video_generate` | Generativo | **Paga ($/seg)** |
| Consultar renderizado de video en curso | `video_status` | Motor local | **FREE ($0)** |
| Cancelar video y liberar retención de saldo | `video_cancel` | Motor local | **FREE ($0)** |
| Consultar estado de presupuesto y modelos | `img_list_models` | Motor local | **FREE ($0)** |

> Para una guía completa del ciclo de vida y mejores prácticas para agentes, ver [docs/workflow.md](workflow.md).

---

## Detalle de herramientas de imagen (`img_`)

### `img_list_models`
- **Coste**: Gratuito (0).
- **Uso**: Consulta el proveedor activo, modelos configurados de imagen y video, llamadas realizadas y estado de presupuesto restante.

### `img_transform`
- **Coste**: Gratuito (0) e instantáneo (< 15 ms).
- **Uso**: Operaciones locales puramente matemáticas (recorte, redimensionado, ajustes de brillo/contraste/saturación, rotación y enfoque) sin conexión a la red. Deduce el formato de la extensión de `output` (JPEG, PNG, WebP).

### `img_generate_drafts`
- **Coste**: De pago (facturado por tokens del proveedor).
- **Uso**: Genera de 1 a 4 borradores a resolución reducida (máx. 768 px) con el modelo draft (`gemini-3.1-flash-lite-image`). Retorna miniaturas para inspección visual directa.

### `img_refine`
- **Coste**: De pago (facturado por tokens del proveedor).
- **Uso**: Operaciones multimodales generativas compositivas (inpainting con máscara, outpainting o eliminación de fondo).

### `img_upscale`
- **Coste**: De pago (facturado por tokens del proveedor).
- **Uso**: Eleva un borrador preexistente a calidad Pro de alta resolución (1K/2K/4K) usando el modelo final (`gemini-3-pro-image-preview`). Genera un thumbnail de previsualización y escribe el sidecar con trazabilidad `derived_from`.

---

## Detalle de herramientas de video (`video_`)

### `video_generate`
- **Coste**: De pago (facturado por segundos de video en Veo 3.1 / Gemini Omni Flash).
- **Comportamiento**: **Asíncrono y no bloqueante**.
  - Valida el presupuesto y emite un ticket de retención (`ReserveTicket`).
  - Despacha la tarea a los servidores de Google y retorna de inmediato con `job_id`, tiempo estimado y directivas operativas.
  - **No congela el cliente MCP** ni genera timeouts (las generaciones duran 1 a 5 minutos).
- **Parámetros principales**:
  - `prompt`: Descripción visual del movimiento y escena.
  - `ref`: (Opcional) Ruta al asset de imagen ancla para animación **Image-to-Video**.
  - `duration_seconds`: Duración del clip (4.0 a 10.0 s, default 5.0).
  - `model_tier`: `'draft'` (Gemini Omni Flash) o `'final'` (Veo 3.1).
  - `aspect_ratio`: `'16:9'`, `'9:16'`, `'1:1'`.

### `video_status`
- **Coste**: Gratuito (0).
- **Comportamiento**:
  - Consulta el estado de la tarea (`processing`, `completed`, `failed`, `cancelled`).
  - Aplica **smart-wait acotado** (por defecto 5 segundos, máx 10 s): si el render termina mientras espera, responde inmediatamente sin forzar al LLM a bucles de polling apretados.
  - Al completarse: persiste el video `.mp4` en disco, genera el sidecar `.meta.json` y retorna un póster miniatura PNG en base64 para inspección visual directa.

### `video_cancel`
- **Coste**: Gratuito (0).
- **Uso**: Cancela un trabajo en progreso y libera inmediatamente los fondos retenidos en el guard presupuestario.

---

## Recursos MCP Declarativos

### `matriz://project/manifest`
- Inventario completo de slots del sitio web, paleta de colores, formatos y dimensiones mínimas requeridas. El agente debe leerlo antes de generar contenido.

### `matriz://jobs`
- Lista de trabajos de video activos y recientes en la sesión, con porcentajes de avance, tiempos estimados y referencias a los assets producidos.
