<script lang="ts">
  let { onsubmit }: { onsubmit: (input: string) => void } = $props();

  let value = $state('');

  function submit(e: SubmitEvent) {
    e.preventDefault();
    const trimmed = value.trim();
    if (!trimmed) return;
    onsubmit(trimmed);
    value = '';
  }
</script>

<form
  class="from-space fixed inset-x-0 bottom-0 bg-gradient-to-t from-65% to-transparent
         px-4 pt-6 pb-[calc(0.75rem+env(safe-area-inset-bottom))]"
  onsubmit={submit}
>
  <input
    type="text"
    bind:value
    placeholder="Ask, create, or find…"
    enterkeyhint="go"
    autocomplete="off"
    autocapitalize="off"
    spellcheck="false"
    class="glass placeholder:text-fg-muted mx-auto block w-full max-w-2xl rounded-xl
           px-4 py-3.5 text-base outline-none transition-shadow
           focus:border-ember/60 focus:shadow-glow"
  />
</form>
