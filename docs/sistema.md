# Caracterização do sistema

## Finalidade

A **API de Tarefas** permite cadastrar tarefas com título, descrição, prioridade e prazo, e acompanhar
cada tarefa por um ciclo de vida de status (`todo` → `in_progress` → `done`). É um sistema pequeno,
escolhido para que o esforço do trabalho fique na automação da qualidade e não nas regras de negócio.

## Tecnologias

| Item                  | Escolha                         | Justificativa                                                                    |
|-----------------------|---------------------------------|----------------------------------------------------------------------------------|
| Linguagem             | Go 1.27                         | Compila rápido, gera um binário único e já traz testes, cobertura e `vet` nativos |
| HTTP                  | `net/http` da biblioteca padrão | Roteamento por método e caminho sem dependência externa                          |
| Persistência          | Repositório em memória          | Execução reproduzível sem banco; a interface permite trocar depois               |
| Dependências          | Go Modules (`go.mod`)           | Arquivo declarativo e verificável (`go mod verify`)                              |
| Testes                | `testing` + `net/http/httptest` | Testes unitários e de API sem subir servidor real                               |

## Arquitetura resumida

```text
cliente HTTP
    │
    ▼
cmd/server         → monta as dependências e sobe o servidor
internal/httpapi   → traduz HTTP/JSON ⇄ casos de uso, mapeia erros para status HTTP
internal/task      → domínio: entidade Tarefa, validações, transições de status,
                     serviço (casos de uso) e repositório em memória
```

As camadas dependem apenas da camada de baixo. O domínio não conhece HTTP, então as regras
podem ser testadas isoladamente (testes unitários) e o contrato HTTP pode ser testado de ponta
a ponta com `httptest` (testes de API e integração).

## Funcionalidade estudada

**Ciclo de vida da tarefa**: criação com validação dos dados e mudança de status com regras de
transição.

| Regra | Descrição                                                                                  |
|-------|--------------------------------------------------------------------------------------------|
| RN01  | Título obrigatório, com 3 a 120 caracteres (espaços nas pontas são ignorados)              |
| RN02  | Descrição com no máximo 1000 caracteres                                                    |
| RN03  | Prioridade `low`, `medium` ou `high`; se omitida, assume `medium`                          |
| RN04  | Prazo opcional; se informado, não pode ser anterior à data atual                           |
| RN05  | Toda tarefa nasce com status `todo`                                                        |
| RN06  | Transições permitidas: `todo → in_progress`, `in_progress → todo`, `in_progress → done`    |
| RN07  | Tarefa `done` é final: não muda de status e não pode ser editada                           |

## Riscos de qualidade

| ID  | Risco                                                                    | Impacto                                   | Verificação que mitiga               |
|-----|--------------------------------------------------------------------------|-------------------------------------------|--------------------------------------|
| R1  | Código que não compila ou dependências inconsistentes chegam à `main`    | Ninguém consegue executar o sistema       | Gate de build                        |
| R2  | Validação quebrada aceita dados inválidos (título vazio, prazo no passado) | Dados corrompidos                       | Testes unitários (Gate de testes)    |
| R3  | Transição de status indevida (ex.: reabrir tarefa concluída)             | Regra de negócio violada                  | Testes unitários e de API            |
| R4  | API responde com status HTTP ou JSON diferente do contrato               | Clientes quebram                          | Testes de API/integração             |
| R5  | Condição de corrida no repositório em memória com acessos simultâneos    | IDs duplicados, dados perdidos            | Teste concorrente com `-race`        |
| R6  | Erros ignorados, código morto ou padrões inseguros                       | Defeitos silenciosos e vulnerabilidades   | Gate de análise estática             |
| R7  | Regras críticas sem nenhum teste                                         | Regressões passam sem ser detectadas      | Gate de cobertura                    |
| R8  | Uso de componente (biblioteca padrão ou módulo) com vulnerabilidade conhecida | Exposição a ataques                 | Gate de dependências                 |
| R9  | Segredos versionados no repositório                                      | Vazamento de credenciais                  | `.gitignore` + revisão no PR (manual) |

## Execução local

Veja o [README](../README.md#instalação-e-execução).
