# Historial de cambios

Todos los cambios relevantes se documentan en este archivo. El proyecto sigue versionado semántico.

## [1.0.0] - 2026-10-17

### Añadido

- Proyección de exposición por ruta, corredor, operador y custodio.
- Perfiles de stress operativo y severo.
- Cálculo desglosado de capital requerido, objetivo, cobertura, superávit y déficit.
- Puerta de límites con decisiones `approve`, `review` y `hold`.
- Comando `risk` y endpoint `POST /v1/risk/analyze`.
- Digest SHA-256 determinista para informes de riesgo.
- Documentación operativa, económica, de arquitectura e integración.
- Matriz CI para Linux y Windows e integridad de release.

### Consolidado

- Flujo de escrow, certificados, liberaciones, disputas, cancelaciones y rebates.
- Fixtures reproducibles y pruebas de integración a través del binario real.
