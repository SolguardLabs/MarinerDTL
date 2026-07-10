# Security Model

MarinerDTL asume un entorno de settlement coordinado por operadores autorizados,
custodios logisticos y cuentas de tesoreria. Los escenarios locales modelan
validaciones economicas esperadas sin depender de infraestructura externa.

## Invariantes Esperadas

- Los escrows de ruta deben estar financiados antes de bloquear hitos.
- Un hito solo puede liberarse con certificado emitido por su custodian.
- Las disputas abiertas deben impedir la liberacion directa del hito afectado.
- Las cancelaciones deben reconciliar refund, penalty y saldo reservado.
- Los rebates deben pagarse desde una cuenta de tesoreria con saldo suficiente.
- El journal debe conservar trazabilidad para cada movimiento economico.

## Validaciones Automatizadas

La suite de tests TypeScript ejecuta fixtures de registro, funding, liberacion,
disputa, penalizacion y cancelacion. El CI local tambien ejecuta `gofmt`,
`go vet`, build del binario y comprobacion de LOC en `src/`.

## Dependencias

El core Go no usa dependencias externas. Las dependencias Node solo se emplean
para tests, formato y ejecucion TypeScript.

## Alcance De Revision

El alcance principal es la logica economica del servicio: estado de rutas,
escrow, certificados, disputas, rebates y reportes. Los fixtures son
deterministas para facilitar reproduccion en auditoria.
