# Arquitectura de Matriz

Matriz es un servidor MCP local (transporte stdio) y visor TUI en Go para gestionar y normalizar los assets de imagen de sitios web de clientes.

## Los seis principios de diseño

### P1 — Separación estricta entre generativo y determinista
Recortar, redimensionar, ajustar brillo/contraste/saturación, convertir formato y generar variantes responsive son operaciones deterministas: se hacen con librerías locales en Go/WASM, son gratuitas, instantáneas y reproducibles. Generar contenido nuevo, inpainting, quitar fondo y expandir encuadre son operaciones generativas: cuestan dinero, tardan segundos y no son reproducibles salvo por seed. Las herramientas y descripciones separan taxativamente ambas clases para impedir gastos accidentales.

### P2 — El modelo tiene que poder ver el resultado
Toda herramienta que produce o modifica una imagen devuelve una miniatura visual (`ImageContent` con lado mayor ≤ 512 px) además de los metadatos estructurados. El archivo a resolución completa se conserva en disco referenciado mediante un `AssetRef`.

### P3 — Un core, dos frontends
Toda la lógica de negocio reside en `internal/core`. El servidor MCP (`cmd/matriz-mcp`) y la TUI (`cmd/matriz-tui`) son frontends desacoplados que consumen el mismo motor. Ninguna lógica de negocio vive en paquetes `cmd/`.

### P4 — Proveedores intercambiables
El core interactúa únicamente con la interfaz genérica `Provider`. Integrar un nuevo proveedor generativo no requiere modificar el core ni las herramientas expuestas.

### P5 — Reproducibilidad por sidecar
Cada imagen generada o transformada escribe un archivo `.meta.json` anexo que registra proveedor, modelo, prompt, seed, parámetros, coste estimado y procedencia (`derived_from`).

### P6 — El original del cliente es sagrado
Ninguna operación modifica ni elimina archivos de entrada in-place. Todo resultado se genera como un derivado nuevo dentro de la estructura de assets del proyecto.
