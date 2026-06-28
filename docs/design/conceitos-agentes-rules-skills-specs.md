# Conceitos: rules, skills, specs, agents e padroes de harness para LLMs

Data da pesquisa: 2026-06-11

## Objetivo

Este documento resume os principais artefatos usados no mercado para orientar, estender, controlar e validar agentes de LLM em engenharia de software. O foco e diferenciar conceitos que frequentemente se misturam: rules, instructions, skills, specs, agents, subagents, tools, MCP, hooks, plugins, templates, policies e evals.

A leitura pragmatica:

- **Specs** dizem o que sera construido e por que.
- **Rules/instructions** dizem como o agente deve se comportar naquele contexto.
- **Skills** ensinam um procedimento reutilizavel ou uma capability.
- **Agents/subagents** sao executores com contexto, ferramentas e permissoes proprias.
- **Tools/MCP** dao acesso estruturado a sistemas externos.
- **Hooks** rodam automacao em pontos do ciclo de vida do agente.
- **Policies/validators** provam conformidade de forma deterministica.
- **Templates/golden paths** criam o caminho correto desde o inicio.
- **Plugins/extensions/presets** empacotam e distribuem capacidades.
- **Evals** medem se tudo isso realmente melhora o resultado.

## Mapa rapido

| Conceito | Resposta curta | Carrega quando | Melhor uso | Erro comum |
| --- | --- | --- | --- | --- |
| Prompt | Instrucao pontual de uma conversa | Turno atual | Pedido especifico | Colocar politica permanente no prompt |
| System/developer instructions | Regras de alto nivel da plataforma ou organizacao | Sempre | Guardrails globais | Misturar com detalhes locais de projeto |
| Rules/instructions | Orientacao persistente por usuario, repo ou caminho | Inicio da sessao ou conforme escopo | Convencoes locais curtas | Virar manual gigante de arquitetura |
| Permission rules | Politica de execucao de comandos/ferramentas | Antes de tool use | Sandbox, aprovacao, bloqueio | Confundir com convencao de codigo |
| Skills | Pacotes reutilizaveis de workflow/conhecimento | Sob demanda | Procedimentos recorrentes e capabilities | Usar como unica forma de enforcement |
| Specs | Artefatos estruturados de requisito/design/tarefas | Durante planejamento e implementacao | Spec-driven development | Guardar todas em repo central sem codigo |
| Agents | Runtime executor com modelo, prompt, ferramentas e permissoes | Durante a tarefa | Trabalho autonomo ou semi-autonomo | Criar muitos agentes sem contrato claro |
| Subagents | Agentes especializados em contexto separado | Quando delegados | Pesquisa, review, paralelizacao | Aumentar custo/contexto sem necessidade |
| Tools/function calling | Funcoes executaveis com schema | Quando o modelo invoca | Acoes deterministicas | Tool sem descricao clara ou sem limites |
| MCP | Protocolo para conectar agentes a ferramentas/dados | Via cliente/servidor MCP | Integracoes portaveis | Dar acesso amplo sem seguranca |
| Hooks | Automacao no ciclo de vida do agente | Eventos como start, pre-tool, stop | Validacao, auditoria, contexto dinamico | Hook opaco que altera comportamento sem visibilidade |
| Slash commands/workflows | Atalhos invocaveis para prompts ou fluxos | Sob comando do usuario/agente | Rotinas de baixa friccao | Virar substituto de pipeline |
| Plugins/extensions | Unidade de distribuicao | Instalacao/habilitacao | Compartilhar skills, hooks, agents | Instalar sem trust/review |
| Memory | Contexto persistente aprendido | Entre sessoes ou subagentes | Preferencias e fatos duraveis | Usar como fonte canonica de arquitetura |
| ADRs/radar | Decisoes humanas versionadas | Consulta/referencia | Governanca tecnica | Esperar que o agente aplique sem validadores |
| Templates/golden paths | Scaffold aprovado | Criacao/migracao | Padronizacao inicial | Nao controlar drift depois |
| Policies/validators | Regras executaveis | CI, hooks, pre/post task | Enforcement | Deixar regra so em texto |
| Evals | Testes do comportamento do agente/skill | Desenvolvimento e regressao | Medir qualidade | Avaliar so pelo resultado final subjetivo |

## Conceitos em detalhe

### Prompt

Prompt e a instrucao ou pedido enviado ao modelo em uma interacao. Ele e adequado para intencao imediata: "adicione endpoint X", "revise este PR", "gere plano para Y".

Use para:

- descrever a tarefa atual;
- informar restricoes especificas daquele pedido;
- pedir um formato de resposta.

Nao use para:

- padroes permanentes de arquitetura;
- politicas de seguranca;
- regras de stack que devem valer em todos os repos.

Para harness, prompt deve ser tratado como entrada volatil, nao como fonte de verdade.

### System/developer instructions

Sao instrucoes de maior precedencia, definidas pela ferramenta, plataforma, organizacao ou harness. Elas moldam o comportamento geral do agente e normalmente nao sao editadas pelo usuario final durante a tarefa.

Use para:

- limites de seguranca;
- comportamento operacional;
- formato de comunicacao;
- regras globais do ambiente.

Nao use para:

- detalhes de uma feature;
- particularidades de uma stack especifica;
- exemplos longos.

No seu caso, as obrigatoriedades da arquitetura homogenea podem ser referenciadas nessa camada, mas o conteudo completo deve viver em contratos, skills e validadores versionados.

### Rules e instructions

No mercado, "rules" pode significar duas coisas diferentes.

**1. Rules/instructions em linguagem natural**

Sao arquivos de contexto persistente. Exemplos:

- `AGENTS.md` em Codex.
- `CLAUDE.md` em Claude Code.
- `.github/copilot-instructions.md` e `.github/instructions/*.instructions.md` em GitHub Copilot.
- `.cursor/rules` ou equivalentes em IDEs agenticas.

Elas servem para dar contexto local ao agente: comandos de build/test, convencoes do repo, estrutura importante, restricoes locais, padroes de PR.

Boas rules sao:

- curtas;
- locais;
- nao contraditorias;
- especificas o suficiente para evitar erro recorrente;
- orientadas a comportamento observavel.

Mas rules nao sao enforcement. O agente pode obedecer mal, esquecer contexto ou interpretar errado. Regras criticas precisam de validator, linter, teste, policy ou hook.

**2. Permission/sandbox rules**

Algumas ferramentas tambem usam "rules" para controlar execucao. No Codex, por exemplo, rules podem controlar quais comandos podem rodar fora do sandbox, com decisoes como `allow`, `prompt` e `forbidden`.

Isso nao e convencao de arquitetura. E politica de permissao operacional.

Regra pratica:

- "Use handlers em `application/commands`" e instruction/rule de arquitetura.
- "Bloqueie `rm -rf`" e permission rule.

### Skills

Skills sao pacotes reutilizaveis de conhecimento e workflow. O padrao Agent Skills define uma skill como uma pasta com `SKILL.md` e, opcionalmente, `scripts/`, `references/` e `assets/`. Ferramentas como Claude Code e Codex usam progressive disclosure: no inicio o agente conhece nome/descricao; quando a skill e relevante, carrega o `SKILL.md`; se necessario, le recursos auxiliares.

Use skills para:

- tarefas recorrentes;
- procedimentos multi-step;
- capabilities por stack;
- reviews especializados;
- runbooks;
- geracao baseada em templates;
- coleta e validacao de contexto.

Exemplos para o harness:

- `architecture-homogeneous`
- `node-nestjs-hexagonal`
- `kotlin-springboot-cqs`
- `php-symfony-ddd`
- `go-service-logging-obfuscation`
- `spec-review`
- `adr-review`
- `architecture-review`

Skills nao devem substituir validadores. Elas orientam o agente a fazer o certo; validadores provam que ficou certo.

### Specs

Specs sao artefatos estruturados que formalizam uma mudanca. Em ferramentas de spec-driven development, como Spec Kit e Kiro, o fluxo costuma decompor a ideia em:

- requisitos ou user stories;
- criterios de aceite;
- design tecnico;
- plano de implementacao;
- tarefas executaveis.

Kiro, por exemplo, descreve specs como artefatos estruturados para transformar ideias de alto nivel em planos detalhados, normalmente com `requirements.md`, `design.md` e `tasks.md`. O GitHub Spec Kit usa um fluxo semelhante com `constitution`, `specify`, `plan`, `tasks` e `implement`.

Use specs para:

- registrar intencao;
- reduzir ambiguidade;
- alinhar produto e engenharia;
- quebrar trabalho em tarefas;
- criar rastreabilidade entre requisito, design, implementacao e testes.

Specs nao sao o mesmo que rules:

- spec e sobre uma mudanca concreta;
- rule e sobre comportamento recorrente do agente ou projeto.

Specs tambem nao sao ADR:

- spec fala do que sera entregue;
- ADR fala de uma decisao tecnica e suas consequencias.

Para seu contexto, specs de feature devem morar no repo do servico que implementa a mudanca. O repo central deve guardar templates, schemas, presets e validadores de spec.

### Agents

Um agent e um executor que combina:

- modelo;
- instructions/system prompt;
- estado de conversa;
- ferramentas;
- permissoes;
- memoria;
- politica de sandbox;
- estrategia de planejamento/execucao.

Em engenharia de software, o agent e quem le o repo, planeja, edita arquivos, roda testes e resume resultado.

Use agents quando ha necessidade de:

- executar tarefas multi-step;
- interagir com arquivos/ferramentas;
- manter estado;
- tomar decisoes instrumentadas por feedback;
- operar sob permissoes controladas.

Nao confunda agent com skill:

- agent e o ator;
- skill e uma capacidade/instrucao carregada pelo ator.

Nao confunda agent com tool:

- agent decide;
- tool executa uma acao especifica.

### Subagents e custom agents

Subagents sao agentes especializados, geralmente com contexto separado, ferramentas proprias e, as vezes, modelo diferente. Claude Code e Codex documentam subagents para isolar pesquisa, paralelizar tarefas e reduzir poluicao do contexto principal.

Use subagents para:

- exploracao read-only de codebase;
- review especializado;
- analise de seguranca;
- comparacao de alternativas;
- execucao paralela de tarefas independentes;
- investigacoes que geram muito output.

Evite subagents quando:

- a tarefa e simples;
- o custo de coordenacao supera o ganho;
- as tarefas dependem fortemente uma da outra;
- o resultado exige uma unica linha de raciocinio consistente.

Para o harness, subagents sao uteis para:

- `architecture-reviewer`;
- `spec-consistency-reviewer`;
- `stack-capability-checker`;
- `migration-planner`;
- `test-gap-reviewer`.

### Tools e function calling

Tools sao funcoes expostas ao modelo com uma interface definida. Podem ler arquivos, buscar dados, chamar APIs, rodar comandos, consultar banco, abrir PR, criar issue, aplicar patch.

Boas tools tem:

- nome claro;
- descricao precisa;
- schema de entrada;
- saida estruturada;
- limites de permissao;
- erros bem definidos;
- logging/auditoria.

Tools sao melhores que prompts quando a tarefa exige acao deterministica ou acesso a dado externo. Se o agente precisa "saber se o repo esta conforme", prefira uma tool `harness_validate` a pedir para ele inferir tudo por leitura manual.

### MCP

MCP, Model Context Protocol, e um padrao aberto para conectar aplicacoes de IA a sistemas externos. Ele padroniza como um cliente de IA acessa data sources, tools e workflows via servidores MCP.

Use MCP para:

- expor ferramentas internas ao agente;
- conectar catalogos, ADRs, radar, Backstage, Jira, GitHub, observabilidade;
- fornecer recursos estruturados;
- evitar integracoes proprietarias por ferramenta;
- tornar capacidades portaveis entre agentes.

No harness, MCP pode expor:

- `get_standards_profile(project)`;
- `resolve_capabilities(stack, framework)`;
- `validate_architecture(repo)`;
- `get_radar_decision(technology)`;
- `list_adrs(project)`;
- `create_spec_from_template(type)`;
- `open_compliance_report(repo)`.

MCP nao substitui skills. MCP fornece acesso a ferramentas/dados; skills dizem quando e como usar essas ferramentas.

### Hooks

Hooks sao scripts, endpoints ou prompts executados automaticamente em eventos do ciclo de vida do agente. Exemplos de eventos: inicio de sessao, envio de prompt, antes de usar uma tool, depois de usar uma tool, stop da resposta, compactacao de contexto.

Use hooks para:

- injetar contexto dinamico;
- bloquear secrets no prompt;
- validar comandos antes da execucao;
- registrar auditoria;
- rodar checks apos mudanca;
- exigir conformidade ao fim de uma tarefa;
- atualizar memoria ou traces.

Hooks sao mais fortes que rules porque executam codigo, mas tambem sao mais perigosos. Precisam de trust, revisao, versao e logs.

Para o harness:

- `PreToolUse`: impedir comandos proibidos ou escrita fora do workspace.
- `PostToolUse`: coletar evidencia de testes/validadores.
- `Stop`: exigir relatorio de conformidade.
- `SessionStart`: carregar profile do projeto.
- `UserPromptSubmit`: detectar se uma tarefa exige spec/ADR antes de implementar.

### Slash commands, commands e workflows

Slash commands sao atalhos invocaveis pelo usuario ou agente para rodar um prompt/fluxo. Exemplo: `/review`, `/specify`, `/plan`, `/tasks`, `/implement`.

Use para:

- tornar workflows descobraveis;
- reduzir repeticao de prompts;
- padronizar entrada/saida;
- criar rituais de desenvolvimento.

Eles sao parecidos com skills quando carregam instrucoes, mas normalmente sao mais "comando de uso" do que "capability reutilizavel". Em algumas plataformas, commands foram incorporados ao modelo de skills.

Para o harness:

- `/harness.spec`
- `/harness.plan`
- `/harness.tasks`
- `/harness.validate`
- `/harness.arch-review`
- `/harness.adr`

### Plugins, extensions e presets

Plugins/extensions sao unidades de distribuicao. Eles empacotam capabilities para instalar em varias maquinas/projetos. Podem conter skills, agents, hooks, MCP servers, comandos, templates e configuracoes.

Presets costumam significar "customizacao de comportamento ou templates existentes", enquanto extensions adicionam novas capacidades.

Use plugins/extensions para:

- distribuir padroes internos;
- versionar bundles;
- habilitar times por stack;
- reduzir copia manual;
- aplicar trust/review na instalacao.

Use presets para:

- adaptar templates de spec;
- aplicar terminologia interna;
- impor secoes obrigatorias;
- customizar fluxo sem criar nova ferramenta.

Para o harness, um pacote poderia ser:

```text
harness-backend-node-nestjs/
  skills/
  agents/
  hooks/
  templates/
  validators/
  plugin.json
```

### Memory

Memory e contexto persistente entre interacoes ou sessoes. Pode guardar preferencias, fatos duraveis, convencoes locais e aprendizados.

Use memory para:

- preferencias pessoais;
- fatos estaveis de um projeto;
- atalhos de ambiente;
- padroes recorrentes observados.

Nao use memory como fonte canonica de governanca. Arquitetura, ADRs, radar, policies e contracts devem estar versionados em repositorios ou sistemas oficiais. Memory pode apontar para eles, mas nao substitui.

### ADRs e radar tecnologico

ADRs registram decisoes arquiteturais: contexto, decisao, alternativas, consequencias. Radar tecnologico registra tecnologias aprovadas, em trial, hold ou deprecated.

Use ADRs/radar para:

- justificar escolhas;
- limitar stacks/frameworks;
- documentar excecoes;
- orientar capabilities;
- dar contexto historico ao agente.

Nao espere que o agente "obedeca ADRs" apenas porque elas existem. Exponha ADRs via skill/MCP e valide via policy.

### Templates, scaffolds e golden paths

Templates e golden paths criam projetos com estrutura, dependencias e workflows aprovados. Sao a forma mais eficiente de padronizar o inicio de um servico.

Use para:

- criar microservicos;
- criar bibliotecas;
- criar workers;
- padronizar CI/CD;
- registrar catalog metadata;
- configurar logging, health checks, observability e tests.

Mas templates envelhecem. Por isso precisam de drift detection e atualizacao automatica, como o modelo de ferramentas do tipo Cruft.

### Policies, validators e guardrails

Policies e validators transformam padroes em checagens executaveis. Podem rodar em CI, hook, pre-commit, PR bot ou ferramenta local.

Use para:

- verificar estrutura de pastas;
- checar direcao de imports;
- bloquear dependencias proibidas;
- validar uso de logger/obfuscacao;
- garantir presenca de ADR para excecao;
- validar `.harness.yaml`;
- checar se stack esta no radar.

Essa e a camada de enforcement. Se algo e obrigatorio, precisa existir aqui.

### Evals

Evals testam se um agente, skill, prompt, workflow ou tool esta funcionando bem. No contexto de harness, evals devem cobrir cenarios reais:

- criar command/handler CQS em stack X;
- adicionar query sem mutacao;
- detectar violacao hexagonal;
- gerar spec com criterios de aceite;
- aplicar logging com obfuscacao;
- rejeitar stack fora do radar.

Metrica util:

- taxa de sucesso;
- violacoes arquiteturais restantes;
- falsos positivos;
- comandos desnecessarios;
- custo de tokens;
- tempo;
- necessidade de intervencao humana;
- regressao entre versoes de skills.

Sem evals, skills viram opiniao textual sem controle de qualidade.

## Relacao entre os artefatos

Uma forma simples de organizar:

```text
Intent
  prompt
  spec
  issue

Context
  AGENTS.md / CLAUDE.md / copilot-instructions.md
  memory
  ADRs
  radar

Capability
  skill
  command
  plugin
  preset

Execution
  agent
  subagent
  tool
  MCP server
  hook

Governance
  policy
  validator
  CI gate
  eval
  compliance report
```

Outra regra pratica:

- Se e **intencao de feature**, escreva spec.
- Se e **convencao local curta**, escreva rule/instruction.
- Se e **procedimento repetivel**, crie skill.
- Se e **acao deterministica**, crie tool.
- Se e **integracao externa portavel**, exponha via MCP.
- Se e **trabalho paralelo/especializado**, use subagent.
- Se e **evento automatico do ciclo do agente**, use hook.
- Se e **distribuicao para varios projetos**, use plugin/preset.
- Se e **obrigatorio**, implemente validator/policy.
- Se e **qualidade do proprio harness**, crie eval.

## Padroes observados no mercado

### 1. Contexto em camadas

Ferramentas modernas tendem a carregar contexto por escopo:

- global/usuario;
- organizacao;
- projeto;
- diretorio/caminho;
- plugin;
- sessao.

Isso permite defaults amplos e overrides locais. O risco e conflito. O harness precisa ter precedencia explicita e capacidade de explicar "por que esta regra foi carregada".

### 2. Progressive disclosure

Skills e recursos extensos nao devem entrar todos no prompt inicial. O agente deve ver catalogo curto e carregar detalhes quando necessario.

Isso reduz custo, evita conflito e melhora foco.

### 3. Spec-driven development

O mercado esta convergindo para fluxos em fases:

- requirements/spec;
- design/plan;
- tasks;
- implementation;
- validation.

Isso e especialmente util com agentes porque reduz ambiguidade antes de editar codigo.

### 4. Subagents e agent teams

Agentes especializados aparecem para reduzir poluicao de contexto e paralelizar pesquisa/review. Sao uteis, mas aumentam custo e complexidade operacional.

### 5. MCP como camada de integracao

MCP esta virando padrao para conectar agentes a dados e ferramentas. Para empresas, isso e melhor do que criar uma integracao diferente para cada IDE/agente.

### 6. Hooks para governanca operacional

Hooks permitem inserir checks e contexto no loop do agente. O padrao e poderoso, mas precisa de trust e auditoria porque hooks podem executar codigo.

### 7. Plugins e registries

Skills e hooks isolados nao escalam bem se forem copiados manualmente. O mercado caminha para bundles, plugins, marketplaces e registries.

### 8. Policies e validadores como fonte de enforcement

O texto orienta; o validador garante. Empresas que querem padronizacao forte precisam traduzir decisoes em checks executaveis.

## Como aplicar no harness de arquitetura homogenea

### Separacao recomendada

| Necessidade do seu harness | Artefato recomendado |
| --- | --- |
| Definir arquitetura obrigatoria | Contract YAML + ADR + skill resumo |
| Ensinar como implementar em Node/NestJS | Skill de capability + templates + examples |
| Criar projeto novo | Golden path/template |
| Garantir estrutura hexagonal | Validator estrutural + import rules |
| Garantir CQS | Validator + testes + exemplos na skill |
| Garantir DDD tatico | Skill + lint/validator onde possivel |
| Garantir logs/obfuscacao | Validator + regra de dependencia + testes |
| Planejar feature | Spec local no repo |
| Explicar excecao | ADR local com expiracao |
| Compartilhar padrao entre times | Plugin/bundle versionado |
| Conectar radar/ADRs/catalogo | MCP server |
| Rodar checks automaticos | Hooks + CI |
| Medir se skills funcionam | Evals |

### Estrutura conceitual

```text
organization standards
  -> contracts
    -> capabilities by stack
      -> skills + templates + validators
        -> project profile
          -> local specs + local instructions
            -> agent execution
              -> hooks + CI + evals
```

### Exemplo de composicao

Uma tarefa "adicionar command para criar pedido no orders-api" deveria ativar:

1. `AGENTS.md` local com comandos do repo.
2. `.harness.yaml` para descobrir stack/profile.
3. Skill `architecture-homogeneous`.
4. Capability `node-nestjs-hexagonal`.
5. Spec da feature, se existir.
6. Tools/MCP para consultar ADR/radar se houver duvida.
7. Validator de estrutura, import rules, CQS e logging.
8. Hook/CI para gerar evidencia.

## Anti-padroes

### Skill monolitica de arquitetura

Problema: carrega contexto demais, mistura stacks, cria conflito e fica dificil de testar.

Melhor: skill base pequena + capabilities por stack + references sob demanda + validators.

### AGENTS.md como wiki

Problema: aumenta custo, conflita com instructions locais e pode piorar tarefas simples.

Melhor: `AGENTS.md` fino, com ponteiros para profile, comandos e excecoes locais.

### Specs centralizadas longe do codigo

Problema: drift entre spec e implementacao.

Melhor: specs locais por repo/servico; repo central so para templates, schemas e iniciativas cross-service.

### Padrao obrigatorio sem check

Problema: vira recomendacao.

Melhor: todo "must" vira validator, linter, policy ou teste.

### Subagents para tudo

Problema: custo, fragmentacao e resultados inconsistentes.

Melhor: usar subagents para tarefas paralelizaveis, read-heavy ou especializadas.

### MCP com acesso amplo demais

Problema: risco de seguranca e exfiltracao.

Melhor: tools pequenas, permissoes por escopo, auditoria e schemas de entrada/saida.

## Recomendacao final de taxonomia para o time

Use nomes consistentes:

- `standards`: decisoes e principios organizacionais.
- `contracts`: invariantes abstratos obrigatorios.
- `capabilities`: implementacoes por stack/framework.
- `skills`: workflows e conhecimentos reutilizaveis.
- `agents`: executores especializados.
- `tools`: funcoes locais ou remotas.
- `mcp`: integracoes padronizadas.
- `hooks`: automacoes no ciclo do agente.
- `templates`: golden paths.
- `specs`: artefatos de feature/produto.
- `adrs`: decisoes tecnicas.
- `policies`: regras executaveis.
- `evals`: testes do proprio harness.

Essa taxonomia evita a confusao principal: nem tudo que orienta o agente e skill; nem tudo que e regra e enforcement; nem tudo que e planejamento e spec; nem todo executor e capability.

## Fontes consultadas

- Agent Skills overview: https://agentskills.io/home
- Agent Skills specification: https://agentskills.io/specification
- Agent Skills client implementation: https://agentskills.io/client-implementation/adding-skills-support
- Claude Code skills: https://code.claude.com/docs/en/skills
- Claude Code subagents: https://code.claude.com/docs/en/sub-agents
- Claude Code hooks: https://code.claude.com/docs/en/hooks
- OpenAI Codex AGENTS.md: https://developers.openai.com/codex/guides/agents-md
- OpenAI Codex rules: https://developers.openai.com/codex/rules
- OpenAI Codex skills: https://developers.openai.com/codex/skills
- OpenAI Codex subagents: https://developers.openai.com/codex/subagents
- OpenAI Codex hooks: https://developers.openai.com/codex/hooks
- GitHub Copilot repository instructions: https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/add-custom-instructions/add-repository-instructions
- GitHub Spec Kit: https://github.com/github/spec-kit
- Kiro specs: https://kiro.dev/docs/specs/
- Model Context Protocol: https://modelcontextprotocol.io/docs/getting-started/intro
- Backstage Software Templates: https://backstage.io/docs/features/software-templates/
- Backstage TechDocs: https://backstage.io/docs/features/techdocs/
- Open Policy Agent: https://www.openpolicyagent.org/docs
- OpenRewrite: https://docs.openrewrite.org/
- Cruft: https://cruft.github.io/cruft/

