# CI Local

> Fecha: 2026-03-09
> Estado: baseline operativo para ejecutar `make ci`

---

## Conclusion

`make ci` esta diseñado para un entorno POSIX. En este Windows nativo no puede ejecutarse
de forma confiable con la configuracion actual del repo.

La via soportada y recomendada es:

- **WSL2 + Ubuntu**

No se recomienda intentar sostener `make ci` completo desde PowerShell puro.

---

## Bloqueantes detectados en este entorno

### Herramientas base ausentes

- `make` no esta instalado
- `bash` no esta instalado
- `python` no esta utilizable en esta sesion
- no existe `.venv`
- no estan disponibles `doorstop` ni `schemathesis`

### Suposiciones POSIX en el repo

El `Makefile` y los scripts asumen:

- `bash`
- `find`
- `awk`
- `grep`
- `sed`
- `tee`
- `mktemp`
- `seq`
- `sleep`
- `curl`
- paths tipo `.venv/bin/...`
- escritura en `/tmp/...`

Tambien dependen de scripts shell:

- `scripts/pattern-refactor-gate.sh`
- `tests/contract/run.sh`

### CI remota

La pipeline en GitHub Actions corre en **`ubuntu-latest`** y confirma que el camino esperado
de ejecucion es Linux/POSIX, no Windows nativo.

---

## Requisito practico

Si `make ci` es fundamental, el entorno local debe alinearse con el entorno de CI.

La opcion correcta es:

1. Instalar WSL2
2. Instalar Ubuntu en WSL
3. Instalar toolchain dentro de WSL
4. Ejecutar el repo desde el filesystem accesible por WSL
5. Correr `make ci` dentro de WSL

---

## Setup minimo en WSL

Paquetes base:

```bash
sudo apt update
sudo apt install -y build-essential bash curl git python3 python3-venv python3-pip make
```

Toolchain Go:

- instalar la version de Go alineada con CI

Entorno Python:

```bash
python3 -m venv .venv
./.venv/bin/pip install --upgrade pip
./.venv/bin/pip install doorstop schemathesis
```

Tooling Go:

```bash
go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

Node para `jscpd`:

- instalar Node 20 o equivalente en WSL

---

## Orden de validacion recomendado

Antes de intentar `make ci`, validar:

```bash
make fmt-check
make complexity
make doorstop-check
make trace-check
make lint
make test
make race-stability
make coverage-gate
make coverage-tdd
make build
make contract-test-strict
```

Si todo eso pasa, entonces `make ci` deberia pasar.

---

## Decision operativa

Para este repo:

- `make ci` local soportado = **WSL2/Ubuntu**
- `make ci` en Windows nativo = **no soportado con la configuracion actual**

Si mas adelante queremos soportar Windows nativo, habra que redisenar:

- Makefile
- scripts shell
- paths de `.venv`
- manejo de `/tmp`
- runner de contract tests

Eso seria una tarea aparte. No conviene mezclarlo con la transicion AGENT_SPEC.
