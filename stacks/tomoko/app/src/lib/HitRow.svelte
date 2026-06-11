<script lang="ts">
  import { hitPath, type SearchHit } from './api';
  import { router } from './router.svelte';

  let { hit }: { hit: SearchHit } = $props();

  const glyph = $derived(
    hit.source_type === 'task' ? '☐' : hit.source_type === 'note' ? '≣' : '▣',
  );
</script>

<button
  type="button"
  class="hover:glass flex w-full cursor-pointer items-start gap-3 rounded-md px-2 py-2.5
         text-left transition-shadow hover:shadow-glow-sm"
  onclick={() => router.go(hitPath(hit))}
>
  <span class="text-ember mt-0.5 text-sm">{glyph}</span>
  <span class="min-w-0 flex-1">
    <span class="block truncate text-[0.95rem]">{hit.title}</span>
    {#if hit.snippet}
      <span class="text-fg-muted block truncate text-xs">{hit.snippet}</span>
    {/if}
  </span>
  {#if hit.project}
    <span class="text-fg-muted shrink-0 text-xs">{hit.project}</span>
  {/if}
</button>
