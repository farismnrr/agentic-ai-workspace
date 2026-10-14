<script lang="ts">
  import { findProject } from '../content/projects';
  import { userRules } from '../content/settings';

  interface Props {
    projectId: string;
  }

  const { projectId }: Props = $props();
  const project = $derived(findProject(projectId));
</script>

<dl class="card grid gap-4 border border-[var(--brand-border)] bg-base-100 p-5 sm:grid-cols-2">
  <div><dt class="text-sm text-[var(--brand-text-secondary)]">Website</dt><dd class="font-medium">{project?.site}</dd></div>
  <div><dt class="text-sm text-[var(--brand-text-secondary)]">Space</dt><dd class="font-medium">Coding</dd></div>
  <div><dt class="text-sm text-[var(--brand-text-secondary)]">Assistant may</dt><dd>Read files and browse. Asks before editing, running commands, or deleting.</dd></div>
  <div>
    <dt class="text-sm text-[var(--brand-text-secondary)]">Your rules here</dt>
    <dd><ul class="list-disc pl-5">{#each userRules as rule (rule.id)}<li>{rule.text}</li>{/each}</ul></dd>
  </div>
</dl>
<a class="link link-hover mt-4 inline-block font-medium text-primary" href="/app/settings/persona">Edit rules and permissions</a>
