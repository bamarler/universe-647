<script lang="ts">
  import { command } from '../lib/command.svelte';
  import HitRow from '../lib/HitRow.svelte';
</script>

<div class="flex flex-col gap-3">
  <h2 class="text-fg-muted px-2 text-sm">
    Results{command.lastQuery ? ` — “${command.lastQuery}”` : ''}
  </h2>
  {#if command.hits.length === 0}
    <p class="text-fg-muted px-2 text-sm">Nothing found.</p>
  {:else}
    {#each command.hits as hit (hit.source_type + hit.source_id)}
      <HitRow {hit} />
    {/each}
  {/if}
</div>
