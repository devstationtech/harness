# Discovery: harness de LLMs para arquitetura homogenea multi-stack

Data da pesquisa: 2026-06-11

## Sumario executivo

O mercado ainda nao tem uma solucao unica que resolva "arquitetura homogenea + LLM agents + multi-stack + specs + enforcement" de ponta a ponta. O que existe sao pecas que se complementam:

- **Agent Skills / skills de agentes** para empacotar conhecimento operacional reutilizavel.
- **Repository instructions** (`AGENTS.md`, `CLAUDE.md`, `.github/copilot-instructions.md`, `.cursor/rules`, etc.) para contexto local minimo.
- **Spec-driven development** para transformar requisitos em `spec -> plan -> tasks -> implementation`.
- **Golden paths / IDP / Backstage** para padronizar criacao e descoberta de servicos.
- **Templates versionados + drift management** para manter projetos alinhados depois do bootstrap.
- **Policy-as-code, linters, codemods e validadores arquiteturais** para transformar padrao em gate verificavel.

A recomendacao e construir o harness como uma **plataforma de engenharia interna**, nao como uma unica skill grande. A skill abstrata de arquitetura homogenea deve existir, mas como **contrato declarativo e pequeno**, composto por capabilities especificas por stack/framework e reforcado por validadores deterministas.

Em relacao as specs: **specs de features devem morar junto do codigo do projeto que sera alterado**. Um repositorio central deve guardar **templates, contratos, ADRs globais, presets, capabilities, validadores e exemplos**, nao todas as specs de todos os servicos. Para iniciativas cross-service, use uma spec de portfolio/integracao em repo central ou monorepo de arquitetura, com links para as specs locais dos servicos afetados.

## Premissas do problema

Premissas obrigatorias citadas:

- Todos os projetos seguem arquitetura hexagonal.
- CQS: commands/handlers e queries isoladas.
- DDD tatico.
- Logging e obfuscacao padronizados.
- Frameworks e stacks definidos por radar tecnologico e ADRs.
- Estrutura base por stack deve ser igual.
- Times podem divergir em detalhes de implementacao, nomenclatura e preferencias locais.
- Objetivo: convergir para padronizacao unica e facilitar compartilhamento de skills.

O ponto critico: skills ajudam a orientar agentes, mas nao devem ser a unica camada de governanca. Padroes arquiteturais precisam de:

1. **Contratos legiveis por humanos e agentes**.
2. **Geradores/templates** para criar o caminho correto.
3. **Validadores deterministas** para impedir regressao.
4. **Codemods/migrations** para corrigir drift em escala.
5. **Skills concisas** para coordenar o trabalho do agente usando esses artefatos.

## O que existe no mercado

### 1. Agent Skills como formato portavel

O padrao aberto Agent Skills define uma skill como uma pasta contendo `SKILL.md`, podendo incluir `scripts/`, `references/` e `assets/`. O modelo recomendado e progressive disclosure: o agente carrega no inicio apenas nome/descricao; carrega o corpo da skill quando ela e ativada; e so le arquivos auxiliares quando precisar.

Isso se encaixa bem no seu desenho de "skill abstrata + capabilities", mas com um ajuste importante: o padrao de skills nao define heranca, interface ou traits. Portanto, a composicao deve ser feita pelo harness, usando metadados e manifestos proprios.

Uso recomendado:

- Skill pequena para o contrato de arquitetura homogenea.
- Capabilities por stack/framework com `SKILL.md` proprio.
- `references/` para exemplos, matrizes de decisao, estrutura de pastas e convencoes.
- `scripts/` para validadores ou coletores de contexto.
- Manifesto externo do harness para `requires`, `provides`, versoes e compatibilidade.

Fontes:

- https://agentskills.io/home
- https://agentskills.io/specification
- https://agentskills.io/client-implementation/adding-skills-support
- https://agentskills.io/skill-creation/best-practices

### 2. Claude Code Skills e escopos de compartilhamento

Claude Code documenta skills em escopos enterprise, pessoal, projeto e plugin. Tambem suporta descoberta por diretorios pai/nested, o que e relevante para monorepos. Isso valida uma ideia importante para o harness: **skills compartilhadas devem existir em camadas**, e nao copiadas integralmente em cada repo.

Boa leitura para seu caso:

- Enterprise/org: padroes obrigatorios, guardrails, skills base.
- Project: comandos especificos de build/test/run e contexto local.
- Nested/monorepo: capabilities por pacote quando o monorepo mistura stacks.
- Plugin/bundle: distribuicao versionada de skills, hooks e ferramentas.

Fonte:

- https://code.claude.com/docs/en/skills

### 3. GitHub Copilot custom instructions

Copilot usa instrucoes de repositorio em `.github/copilot-instructions.md` e instrucoes por caminho em `.github/instructions/*.instructions.md`. A documentacao alerta que instrucoes repository-wide e path-specific podem ser combinadas.

Para o harness, isso sugere um principio pratico: mantenha instrucoes de repo **curtas e locais**. Nao copie todo o manual de arquitetura para cada projeto; referencie o perfil/preset versionado e inclua apenas comandos, estrutura real e excecoes locais.

Fonte:

- https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/add-custom-instructions/add-repository-instructions

### 4. Spec Kit e spec-driven development

O GitHub Spec Kit e uma referencia direta para o seu problema. Ele propoe um fluxo `constitution -> specify -> plan -> tasks -> implement`, suporta varias integracoes de agentes e tem duas extensoes conceituais importantes:

- **Presets**: customizam formatos, templates e principios.
- **Extensions**: adicionam novos comandos/capabilities.

Isso e muito proximo do que voce descreveu como skill abstrata + capacidades. A diferenca e que, no Spec Kit, standards organizacionais tendem a entrar como **constitution/presets**, enquanto novas fases ou integracoes entram como **extensions**.

Recomendacao: se voces ja tem spec kit proprio, mantenham a ideia, mas acrescentem:

- profile de arquitetura por tipo de projeto;
- presets por stack;
- validadores obrigatorios por fase;
- rastreabilidade entre spec, ADR, capability e CI check.

Fonte:

- https://github.com/github/spec-kit

### 5. Backstage, golden paths e platform engineering

Backstage Software Templates resolvem a parte de "criar projetos no caminho certo". TechDocs reforca docs-as-code perto do servico. Esse padrao e amplamente usado em platform engineering: oferecer golden paths com defaults aprovados, sem impedir excecoes controladas.

Para seu harness, Backstage/IDP pode ser a camada de entrada:

- Criar microservico a partir de template aprovado.
- Selecionar stack/framework permitido pelo radar.
- Registrar ownership, lifecycle, system, domain e standards profile.
- Expor TechDocs, ADRs, specs e score de conformidade.

Fontes:

- https://backstage.io/docs/features/software-templates/
- https://backstage.io/docs/features/techdocs/

### 6. Templates versionados e controle de drift

Cookiecutter cria projetos a partir de templates; Cruft adiciona capacidade de manter projetos sincronizados com o template original, incluindo `cruft check`, `cruft update` e automacao via CI/PR.

Isso e essencial porque padronizacao nao termina no bootstrap. Sem drift management, cada service repo vira uma copia congelada do template em algum ponto do passado.

Recomendacao:

- Cada projeto deve registrar `template_id`, `template_version` e `standards_profile`.
- O harness deve detectar drift estrutural.
- Atualizacoes de golden path devem abrir PRs automaticos quando possivel.

Fontes:

- https://cookiecutter.readthedocs.io/en/stable/
- https://cruft.github.io/cruft/

### 7. Refactoring/codemods em escala

OpenRewrite e um exemplo forte de refactoring deterministico por receitas. Ele usa receitas para migracoes, correcoes de seguranca e consistencia estilistica, preservando formatacao quando possivel. Embora tenha origem forte em Java, o conceito e o que importa: **nao pedir para o LLM fazer migracoes repetitivas quando uma receita deterministica pode fazer melhor**.

Equivalentes por stack:

- Kotlin/Java: OpenRewrite, ArchUnit, Detekt, ktlint.
- PHP: Rector, PHPStan/Psalm, PHP-CS-Fixer.
- Node/TypeScript: ESLint custom rules, jscodeshift, ts-morph, dependency-cruiser.
- Go: gofmt, go vet, staticcheck, custom analyzers.

Fonte:

- https://docs.openrewrite.org/

### 8. Policy-as-code

OPA/Rego e Conftest mostram a abordagem de separar policy decision de enforcement. Para o harness, isso e util para invariantes sobre manifests, catalogos, IaC, pipelines e ate convencoes estruturais exportadas como JSON.

Use policy-as-code para regras como:

- projeto deve declarar `standards_profile`;
- stack deve estar aprovada no radar;
- repo deve ter owner e lifecycle;
- capabilities obrigatorias devem estar presentes;
- excecoes precisam de ADR e data de expiracao.

Fonte:

- https://www.openpolicyagent.org/docs

### 9. Evidencias recentes sobre arquivos de contexto

Estudos recentes sobre `AGENTS.md` e manifests de agentes apontam um trade-off: arquivos de contexto podem ajudar runtime/custo quando sao objetivos, mas podem prejudicar sucesso quando carregam requisitos desnecessarios, conflitantes ou verbosos.

Implicacao para sua estrategia:

- Nao transforme `AGENTS.md` em wiki de arquitetura.
- Nao coloque todas as regras de todas as stacks em todos os projetos.
- Prefira contexto minimo + skills/capabilities ativadas sob demanda + validadores.

Fontes:

- https://arxiv.org/abs/2602.11988
- https://arxiv.org/abs/2601.20404
- https://arxiv.org/abs/2509.14744

## Recomendacao de arquitetura do harness

### Modelo mental

Use a analogia de interfaces/traits apenas como design conceitual. A implementacao mais robusta e:

- **Contrato**: define o que precisa ser verdadeiro.
- **Capability**: implementa como aplicar o contrato em uma stack/framework.
- **Validator**: prova que a capability foi aplicada.
- **Skill**: orienta o agente a usar contrato, capability e validator.
- **Preset/template**: gera o caminho feliz.
- **ADR/radar**: justifica escolhas e limita variabilidade.

Skills sao texto procedural. Contratos e validadores devem ser dados/codigo versionados.

### Camadas propostas

1. **Organizational Standards**
   - ADRs globais.
   - Radar tecnologico.
   - Principios obrigatorios.
   - Politicas de excecao.

2. **Architecture Contracts**
   - `homogeneous-architecture`.
   - `hexagonal-architecture`.
   - `cqs`.
   - `tactical-ddd`.
   - `logging-obfuscation`.
   - `observability`.
   - `security-baseline`.

3. **Stack Capabilities**
   - `php-symfony-hexagonal`.
   - `php-laravel-hexagonal`.
   - `kotlin-springboot-hexagonal`.
   - `node-nestjs-hexagonal`.
   - `go-chi-hexagonal`.
   - Cada capability declara quais contratos implementa.

4. **Project Overlay**
   - bounded contexts reais;
   - comandos de build/test/run;
   - decisoes locais aprovadas;
   - excecoes temporarias.

5. **Enforcement**
   - linters;
   - testes;
   - validadores arquiteturais;
   - policy-as-code;
   - codemods;
   - CI gates;
   - PR review assistido por agente.

### Estrutura sugerida para o repositorio central

```text
harness-platform/
  standards/
    adr/
    radar/
    principles/
  contracts/
    homogeneous-architecture/
      contract.yaml
      invariants.md
      examples/
    hexagonal-architecture/
    cqs/
    tactical-ddd/
    logging-obfuscation/
  capabilities/
    php/
      symfony/
        capability.yaml
        skills/
        validators/
        templates/
      laravel/
    kotlin/
      springboot/
    node/
      nestjs/
    go/
      chi/
  skills/
    architecture-homogeneous/
      SKILL.md
      references/
    spec-review/
    adr-review/
    architecture-review/
  spec-kit/
    presets/
    templates/
    extensions/
  templates/
    services/
    libraries/
    workers/
  validators/
    structure/
    policies/
    import-rules/
  codemods/
    php/
    kotlin/
    node/
    go/
  evals/
    fixtures/
    golden-tasks/
  registry/
    bundles.yaml
```

### Estrutura sugerida em cada projeto

```text
service-a/
  .harness.yaml
  AGENTS.md
  .github/
    copilot-instructions.md
    instructions/
  .agents/
    skills/
      run-verify/
        SKILL.md
  .specify/
    memory/
      constitution.md
  specs/
    001-create-order/
      spec.md
      plan.md
      tasks.md
      checks.md
  docs/
    adr/
  src/
  tests/
```

`AGENTS.md` e `.github/copilot-instructions.md` devem ser finos:

- como instalar;
- como testar;
- onde estao specs/ADRs;
- qual `standards_profile` usar;
- quais comandos nunca rodar;
- excecoes locais.

Nao devem duplicar todos os padroes corporativos.

### Exemplo de `.harness.yaml`

```yaml
schema: harness.devstation.tech/v1
project:
  name: orders-api
  type: microservice
  owner: payments
  lifecycle: production

standards_profile: backend-service-v1

stack:
  language: typescript
  runtime: node
  framework: nestjs
  package_manager: pnpm

architecture:
  archetype: node-nestjs-hexagonal
  contracts:
    - homogeneous-architecture@1
    - hexagonal-architecture@1
    - cqs@1
    - tactical-ddd@1
    - logging-obfuscation@1

capabilities:
  required:
    - node-nestjs-hexagonal@1
    - node-cqs-handlers@1
    - node-ddd-tactical@1
    - node-structured-logging-obfuscation@1

template:
  id: service-node-nestjs
  version: 1.8.0

exceptions:
  - adr: ADR-014
    contract: cqs@1
    reason: legacy read model migration
    expires_on: 2026-12-31
```

### Exemplo de contrato

```yaml
id: homogeneous-architecture
version: 1
status: active
requires:
  - hexagonal-architecture
  - cqs
  - tactical-ddd
  - logging-obfuscation

invariants:
  - id: domain-has-no-framework-imports
    severity: error
    description: Domain layer must not import framework, transport, persistence or logging implementation packages.
  - id: commands-and-queries-are-isolated
    severity: error
    description: Command handlers must not return read models; query handlers must not mutate state.
  - id: logs-use-approved-obfuscation
    severity: error
    description: Logs containing sensitive fields must use the approved obfuscation library.

evidence:
  required:
    - structure-validator
    - import-rule-validator
    - test-command
```

### Como compor skills/capabilities

Fluxo recomendado do harness:

1. Detectar stack por arquivos (`composer.json`, `build.gradle.kts`, `package.json`, `go.mod`) e por `.harness.yaml`.
2. Resolver `standards_profile`.
3. Consultar radar/ADRs para validar se stack/framework sao permitidos.
4. Montar catalogo de skills disponiveis.
5. Ativar apenas:
   - skill base de arquitetura se a tarefa toca design/codigo;
   - capability da stack afetada;
   - skill de spec/ADR quando a tarefa altera artefatos de planejamento;
   - skill de run/verify local quando precisa validar.
6. Executar validadores antes/depois da alteracao.
7. Produzir relatorio de conformidade no PR.

## Specs: por projeto ou repo central?

### Recomendacao direta

Use os dois, mas com responsabilidades diferentes.

**Specs de feature devem ficar no projeto que implementa a mudanca.**

Motivos:

- versionam junto com o codigo;
- entram no mesmo PR;
- refletem realidade local de testes e design;
- reduzem divergencia entre spec e implementacao;
- permitem que agentes encontrem contexto sem depender de outro checkout.

**Repo central deve guardar o sistema de specs, nao todas as specs.**

Ele deve conter:

- templates de `spec.md`, `plan.md`, `tasks.md`;
- schema/validador de spec;
- exemplos canonicos;
- presets por tipo de projeto;
- instrucoes para agentes;
- contratos e policies;
- ADRs globais;
- radar tecnologico;
- catalogo de capabilities.

### Quando usar repo central de specs

Use um repo central ou area central para:

- iniciativas cross-service;
- contratos de integracao entre dominios;
- RFCs/plataforma;
- specs de migracao em massa;
- padroes de arquitetura;
- visao de produto que se desdobra em varios repos.

Mesmo nesses casos, a spec central deve apontar para specs locais nos servicos afetados.

Exemplo:

```text
platform-specs/
  initiatives/
    2026-07-payment-obfuscation-rollout/
      overview.md
      affected-services.yaml
      integration-contract.md
      rollout-plan.md

orders-api/
  specs/
    023-payment-obfuscation/
      spec.md
      plan.md
      tasks.md

billing-worker/
  specs/
    017-payment-obfuscation/
      spec.md
      plan.md
      tasks.md
```

### Monolitos e monorepos

Para monolitos:

- `specs/<feature>/` na raiz.
- Se o monolito tem modulos bem delimitados, inclua `affected_modules` no frontmatter da spec.

Para monorepos multi-stack:

- specs transversais na raiz: `specs/<feature>/`.
- specs locais quando a mudanca e isolada: `services/<service>/specs/<feature>/`.
- `.harness.yaml` por pacote/servico quando stacks variam.

## Padronizacao vs preferencias dos times

Classifique variacoes em tres categorias:

1. **Invariantes**
   - Nao negociaveis.
   - Exemplo: domain nao importa infra; commands e queries isolados; logs usam obfuscacao aprovada.
   - Enforcement em CI.

2. **Defaults canonicos**
   - Padrao unico recomendado.
   - Exemplo: nomes de pastas, nomes de handlers, convencao de testes.
   - Divergencia exige ADR local ou aprovacao de plataforma.

3. **Extensoes permitidas**
   - Pontos onde times podem variar sem quebrar homogeneidade.
   - Exemplo: estrategia interna de mapper, nomes de metodos privados, granularidade de arquivos.
   - Documentar no capability contract.

Se tudo vira preferencia local, o harness perde valor. Se tudo vira regra rigida, a plataforma vira gargalo. O equilibrio e tornar rigido o que afeta interoperabilidade, seguranca, qualidade e compartilhamento de skills.

## Governanca recomendada

### Pipeline minimo por PR

1. Validar `.harness.yaml`.
2. Validar se stack/framework esta no radar.
3. Validar estrutura hexagonal.
4. Validar regras de import/dependency direction.
5. Validar CQS.
6. Validar uso de logging/obfuscacao.
7. Rodar linters/testes da stack.
8. Rodar skill de review arquitetural como comentario auxiliar, nao como unica fonte de verdade.

### Score de conformidade

Cada repo deveria ter um score visivel:

- template drift;
- standards profile;
- capabilities faltantes;
- excecoes ativas;
- excecoes expiradas;
- ultimo run dos validadores;
- cobertura de specs;
- ADRs locais pendentes.

Esse score pode aparecer no Backstage/catalogo interno.

### Excecoes

Toda excecao deve ter:

- ADR ou issue;
- owner;
- motivo;
- impacto;
- prazo de expiracao;
- plano de remocao;
- regra exata que foi excepcionada.

## Roadmap de implementacao

### Fase 1: inventario e contratos

- Inventariar stacks reais: PHP, Kotlin, Node, Go.
- Mapear frameworks aprovados por radar/ADR.
- Definir `standards_profile` iniciais.
- Separar invariantes de defaults.
- Criar contratos YAML para arquitetura homogenea, hexagonal, CQS, DDD tatico e logging/obfuscacao.

### Fase 2: MVP em uma ou duas stacks

- Escolher uma stack com alto volume e uma com alta criticidade.
- Criar template base.
- Criar capability da stack.
- Criar skill base e skill da stack.
- Criar validadores estruturais e de import.
- Integrar com CI.

### Fase 3: spec kit interno

- Padronizar templates `spec.md`, `plan.md`, `tasks.md`.
- Adicionar `constitution` organizacional.
- Criar presets por tipo de projeto.
- Criar validador de spec.
- Exigir link entre spec, ADRs e contracts afetados.

### Fase 4: registry e distribuicao

- Publicar bundles versionados de skills/capabilities.
- Assinar ou pelo menos checksum dos bundles.
- Definir resolucao de versao.
- Gerar instrucoes de agente por repo a partir do profile.
- Suportar instalacao local, projeto e CI.

### Fase 5: drift e modernizacao

- Detectar drift de template.
- Criar PRs automaticos para atualizacoes simples.
- Usar codemods para migracoes repetitivas.
- Medir taxa de conformidade por time/produto.

### Fase 6: avaliacoes de qualidade de skills

- Criar fixtures por stack.
- Criar tarefas golden.
- Rodar skills contra cenarios reais.
- Comparar resultado com/sem skill.
- Medir falsos positivos, tempo, tokens, alteracoes incorretas e violacoes de arquitetura.

## Riscos e mitigacoes

| Risco | Mitigacao |
| --- | --- |
| Skill gigante demais | Progressive disclosure, `SKILL.md` curto, referencias sob demanda |
| Instrucoes conflitantes entre repo/org | Precedencia explicita e geracao automatica de instrucoes locais |
| LLM "acha" que cumpriu arquitetura | Validadores deterministas e CI gates |
| Times copiam e editam skills | Registry versionado, bundles e profiles |
| Divergencias viram padrao informal | Excecoes com ADR, owner e expiracao |
| Template drift | Cruft-like check/update e PRs automaticos |
| Skill supply-chain | Trust de repos, allowlist, assinatura/checksum, revisao de scripts |
| Multi-stack vira duplicacao | Contratos abstratos + capabilities por stack |

## Decisoes recomendadas

1. Criar um **repo central de harness/platform**, nao um repo central para todas as specs de feature.
2. Manter **specs locais junto do codigo**, com templates e validadores vindos do repo central.
3. Modelar arquitetura homogenea como **contrato + capabilities + validators**, nao como uma unica skill monolitica.
4. Usar skills como camada de orquestracao do agente, nao como mecanismo primario de enforcement.
5. Definir uma matriz oficial de archetypes por stack/framework.
6. Criar `.harness.yaml` em cada projeto.
7. Gerar `AGENTS.md`, Copilot instructions e skills locais a partir do profile.
8. Comecar com poucas regras fortes e validaveis; expandir conforme evidencias.
9. Usar Backstage/IDP ou catalogo equivalente para visibilidade, onboarding e score de conformidade.
10. Tratar excecoes como divida arquitetural rastreada.

## Proxima proposta de desenho

Um MVP pragmmatico teria:

- `harness-platform` com contratos YAML.
- Uma skill base `architecture-homogeneous`.
- Duas capabilities iniciais, por exemplo `node-nestjs-hexagonal` e `kotlin-springboot-hexagonal`.
- Um template de service por stack.
- `.harness.yaml` por repo.
- Um validador CLI `harness validate`.
- Integracao CI.
- Preset do spec kit interno.
- Um dashboard simples de conformidade por repo.

Esse MVP ja permitiria testar a hipotese central: agentes conseguem produzir codigo mais alinhado quando recebem um contrato pequeno, uma capability especifica e validadores executaveis, em vez de receber um manual longo de arquitetura.

