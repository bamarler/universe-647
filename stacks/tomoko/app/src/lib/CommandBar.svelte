<script lang="ts">
  import { command } from './command.svelte';
  import DraftCard from './DraftCard.svelte';

  // Mobile interaction model: bar anchored at the bottom, thumb-friendly;
  // draft cards materialize directly above the input — no separate pages.
  let value = $state('');

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    const trimmed = value.trim();
    if (!trimmed || command.busy) return;
    await command.run(trimmed);
    if (!command.error) value = '';
  }
</script>

<div
  class="from-space fixed inset-x-0 bottom-0 bg-gradient-to-t from-55% to-transparent px-4
         pt-8 pb-[calc(0.75rem+env(safe-area-inset-bottom))] md:hidden"
>
  <div class="mx-auto flex max-w-2xl flex-col gap-3">
    <DraftCard />

    {#if command.error && !command.draft}
      <div class="glass text-danger rounded-lg px-3 py-2 text-sm">{command.error}</div>
    {/if}

    <form onsubmit={submit}>
      <input
        type="text"
        bind:value
        placeholder={command.busy ? 'Thinking…' : 'Ask, create, or find…'}
        disabled={command.busy}
        enterkeyhint="go"
        autocomplete="off"
        autocapitalize="off"
        spellcheck="false"
        class={[
          'glass placeholder:text-fg-muted w-full rounded-xl px-4 py-3.5 text-base',
          'focus:border-ember/60 focus:shadow-glow outline-none transition-shadow',
          command.busy && 'animate-pulse',
        ]}
      />
    </form>
  </div>
</div>
