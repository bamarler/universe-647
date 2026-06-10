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

<form class="bar" onsubmit={submit}>
  <input
    type="text"
    bind:value
    placeholder="Ask, create, or find…"
    enterkeyhint="go"
    autocomplete="off"
    autocapitalize="off"
    spellcheck="false"
  />
</form>

<style>
  .bar {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    padding: 0.75rem 1rem calc(0.75rem + env(safe-area-inset-bottom));
    background: linear-gradient(transparent, var(--bg) 35%);
  }
  input {
    display: block;
    width: 100%;
    max-width: 40rem;
    margin: 0 auto;
    padding: 0.8rem 1rem;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 0.75rem;
    color: var(--fg);
    font-size: 1rem;
    outline: none;
  }
  input:focus {
    border-color: var(--accent);
  }
  input::placeholder {
    color: var(--fg-muted);
  }
</style>
