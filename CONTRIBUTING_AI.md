# CONTRIBUTING_AI.md

Instrucoes para futuras sessoes de IA e colaboradores automatizados.

## Regra Principal

O repositorio atual e a fonte primaria da verdade. Antes de alterar codigo, leia o estado real dos arquivos relevantes.

## Leitura Obrigatoria Antes de Codar

Toda IA deve ler, nesta ordem:

1. `BRAIN.md`
2. `CONTEXT.md`
3. `ARCHITECTURE.md`
4. `DECISIONS.md`
5. `ROADMAP.md`
6. `README.md`

Tambem deve ler os arquivos de codigo diretamente relacionados a tarefa.

## Regras de Atualizacao da Documentacao

- Atualize `DECISIONS.md` ao tomar decisoes arquiteturais relevantes.
- Atualize `ROADMAP.md` ao concluir, iniciar ou replanejar funcionalidades.
- Atualize `BRAIN.md` quando uma nova regra permanente surgir.
- Atualize `CONTEXT.md` quando novos conceitos de produto forem implementados.
- Atualize `ARCHITECTURE.md` quando a estrutura, fluxos ou dependencias mudarem.

## Regras de Desenvolvimento

- Nao reestruture o projeto sem necessidade clara.
- Nao remova arquivos existentes sem pedido explicito.
- Nao implemente IA, voz ou agentes antes da fase correspondente.
- Respeite a arquitetura existente em `backend/internal` e `frontend`.
- Prefira evolucao incremental.
- Mantenha configuracao por environment variables.
- Preserve a separacao entre backend, frontend e banco.
- Evite acoplar handlers diretamente a provedores externos.
- Introduza services e repositories quando regras de negocio ou persistencia crescerem.
- Registre hipoteses como pendentes, nao como decisoes definitivas.

## Checklist Antes de Finalizar Uma Tarefa

- O codigo alterado foi formatado.
- Testes ou builds relevantes foram executados quando possivel.
- A documentacao viva foi atualizada se a tarefa mudou arquitetura, roadmap ou decisoes.
- O resumo final informa o que foi alterado e o que foi validado.
- Riscos ou limitacoes encontrados foram registrados.

## Modulos Reservados

Os seguintes modulos existem como fronteiras planejadas e nao devem ser preenchidos com implementacoes improvisadas:

- `backend/internal/ai`
- `backend/internal/voice`
- `backend/internal/memory`
- `backend/internal/tools`
- `backend/internal/agents`
- `backend/internal/services`
- `backend/internal/repositories`
- `frontend/hooks`

## Padrao Para Novas Decisoes

Ao adicionar uma decisao em `DECISIONS.md`, use:

```text
### YYYY-MM-DD - Titulo da Decisao

Status: aceita | planejada | pendente de validacao | substituida

Contexto:

- ...

Decisao:

- ...

Impacto:

- ...
```
