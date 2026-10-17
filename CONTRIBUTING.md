# Contribuir a MarinerDTL

## Preparación

```bash
git clone https://github.com/SolguardLabs/MarinerDTL.git
cd MarinerDTL
npm ci
npm run ci
```

## Criterios de cambio

- Mantenga importes en enteros y porcentajes en puntos básicos.
- Separe cálculos puros de la mutación del estado.
- Añada una prueba Go para cada rama matemática nueva.
- Añada un fixture o prueba TypeScript cuando cambie un contrato observable.
- Documente nuevas entradas, salidas, errores e invariantes.
- Evite dependencias de red, hora o aleatoriedad en pruebas.

## Commits y pull requests

Use mensajes imperativos y concretos. Un pull request debe indicar alcance, impacto económico, validaciones ejecutadas y compatibilidad. No mezcle cambios de formato con cambios funcionales extensos.

## Puerta local

`npm run ci` debe finalizar correctamente. El comando ejecuta el mismo conjunto de comprobaciones que GitHub Actions; `npm run release:check` verifica además los artefactos y las fuentes protegidas de la versión.
