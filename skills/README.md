# Atlas Agent Skills

Skills that teach coding agents to manage database schemas with [Atlas](https://atlasgo.io): migrations,
linting, testing, CI/CD, the Kubernetes Operator, and the specifics of each database.

> [!NOTE]
> The [Atlas website](https://atlasgo.io/guides/ai-tools/agent-skills) has the most up-to-date version
> of these skills: they are maintained there and copied here from time to time. For the latest
> files of a skill, use `https://atlasgo.io/skills/<name>/SKILL.md`, with its reference files under
> `references/`.

## Install

Claude Code:

```bash
claude plugin marketplace add ariga/atlas
claude plugin install atlas@ariga
```

Codex, Cursor, or GitHub Copilot, from the root of your repository:

```bash
npx skills add ariga/atlas -a codex -y   # or: -a cursor, -a github-copilot
```

To onboard a project, run `/atlas:atlas-onboard` in Claude Code, `/atlas-onboard` in Cursor or
Copilot, or `$atlas-onboard` in Codex.
