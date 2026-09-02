# Manifiesto del sitio (`matriz.json`)

El manifiesto de proyecto proporciona contexto unificado sobre los slots visuales de la landing, la paleta de colores de marca y el inventario completo de assets.

## Schema `matriz.manifest/v1`

```json
{
  "schema": "matriz.manifest/v1",
  "project": "peluqueria-estilo",
  "palette": ["#0b0b0b", "#c8a45c", "#f4f1ea"],
  "slots": [
    {
      "id": "hero",
      "usage": "portada, primera pantalla",
      "aspect_ratio": "21:9",
      "min_width": 1920,
      "sizes_hint": "100vw",
      "asset": "assets/hero-01.avif",
      "alt": "Interior del salón con iluminación cálida"
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

## Reglas de procedencia (`origin`)
- **`client`**: Material original enviado por el cliente. Jamás se modifica ni elimina in-place.
- **`generated`**: Generado mediante modelos generativos de IA (Gemini).
- **`derived`**: Producido mediante transformaciones deterministas (filtros, recortes, conversiones de formato).

## Acceso MCP Resource
El manifiesto se expone en la URI:
```
matriz://project/manifest
```
Los agentes LLM leen este recurso antes de cualquier generación para respetar encuadres (`aspect_ratio`), anchos mínimos (`min_width`) y paleta de marca.
