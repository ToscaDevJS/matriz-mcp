# Proveedores de imagen

## Activo: Google Gemini (familia Nano Banana)
- **SDK**: `google.golang.org/genai` (oficial de Google, licencia Apache 2.0).
- **Modelo de borrador**: `gemini-3.1-flash-lite-image` — $0.0336/imagen — resolución 1K (máx. 768 px en borrador).
- **Modelo final**: `gemini-3-pro-image-preview` — $0.134/imagen (1K/2K), $0.240/imagen (4K).
- **Precios consultados el**: 2026-08-31 en `https://ai.google.dev/pricing`.
- **Uso comercial**: Permitido bajo los términos de la Gemini Developer API / Google AI Studio.
- **Marca de agua**: SynthID invisible en todas las salidas generadas. Salidas gratuitas de AI Studio sujetas a verificación de marca visible según ruta de cuenta (aceptación manual A-03).
- **Acepta seed**: Sí, soporta semilla determinista vía configuración de generación.
- **Soporta máscara de inpainting**: Sí, a través del endpoint multimodal con imagen de entrada y máscara binaria.

## Diferido: fal.ai
- **Disparador para integrarlo**: Ver PR-6 en el handoff (volumen de fondos > 50 con BiRefNet, control exacto de paleta hex o calidad de matting de pelo fino).
- **Motivo de la espera**: No existe SDK oficial de Go publicado por fal.ai en esta versión.
