<script lang="ts">
  import { Command, Dialog } from 'bits-ui';
  import { api, hitPath, type SearchHit } from './api';
  import { command } from './command.svelte';
  import { router } from './router.svelte';
  import DraftCard from './DraftCard.svelte';

  // Desktop interaction model: Cmd+K palette, fully keyboard-navigable.
  // Free text submits to the semantic router; results and drafts render
  // inside the palette. The AI never speaks — it only lists and drafts.
  let open = $state(false);
  let value = $state('');
  let hits = $state<SearchHit[]>([]);
  let mode = $state<'idle' | 'results' | 'draft'>('idle');

  const quickNav = [
    { label: 'Today', path: '/view/today', glyph: '☀' },
    { label: 'Upcoming', path: '/view/upcoming', glyph: '↗' },
    { label: 'Inbox', path: '/view/inbox', glyph: '⊡' },
    { label: 'Home', path: '/', glyph: '◌' },
  ];

  let folders = $state<{ id: number; name: string; kind: string }[]>([]);

  export function toggle() {
    open = !open;
    if (open) reset();
  }

  function reset() {
    value = '';
    hits = [];
    mode = 'idle';
    command.dismissDraft();
    api.tags().then((r) => {
      folders = r.tags.filter((t) => t.kind === 'project' || t.kind === 'context');
    });
  }

  function onkeydown(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault();
      toggle();
    } else if (e.key === 'Escape' && open) {
      open = false;
    }
  }

  function go(path: string) {
    open = false;
    router.go(path);
  }

  async function ask() {
    const trimmed = value.trim();
    if (!trimmed || command.busy) return;
    await command.run(trimmed, { navigate: false });
    if (command.draft) {
      mode = 'draft';
    } else if (!command.error) {
      hits = command.hits;
      mode = 'results';
    }
  }
</script>

<svelte:window {onkeydown} />

<Dialog.Root bind:open>
  <Dialog.Portal>
    <Dialog.Overlay class="bg-space-deep/70 fixed inset-0 z-40 backdrop-blur-sm" />
    <Dialog.Content
      class="fixed top-[15vh] left-1/2 z-50 w-[min(40rem,92vw)] -translate-x-1/2"
      onOpenAutoFocus={(e: Event) => e.preventDefault()}
    >
      {#if mode === 'draft' && command.draft}
        <DraftCard oncomplete={() => (open = false)} />
      {:else}
        <Command.Root
          class="glass shadow-glow flex flex-col overflow-hidden rounded-xl"
          shouldFilter={mode !== 'results'}
        >
          <form
            onsubmit={(e) => {
              e.preventDefault();
              ask();
            }}
          >
            <Command.Input
              bind:value
              placeholder={command.busy ? 'Thinking…' : 'Type a command — Enter asks Sophon…'}
              disabled={command.busy}
              autofocus
              class="placeholder:text-fg-muted border-line/60 w-full border-b bg-transparent
                     px-4 py-3.5 text-base outline-none"
            />
          </form>
          <Command.List class="max-h-[50vh] overflow-y-auto p-1.5">
            {#if command.error}
              <p class="text-danger px-3 py-2 text-sm">{command.error}</p>
            {/if}

            {#if mode === 'results'}
              {#if hits.length === 0}
                <p class="text-fg-muted px-3 py-2 text-sm">Nothing found.</p>
              {/if}
              {#each hits as hit (hit.source_type + hit.source_id)}
                <Command.Item
                  value={`${hit.source_type}-${hit.source_id}-${hit.title}`}
                  onSelect={() => go(hitPath(hit))}
                  class="data-selected:glass data-selected:shadow-glow-sm flex cursor-pointer
                         items-center gap-2.5 rounded-md px-3 py-2.5"
                >
                  <span class="text-ember text-sm">
                    {hit.source_type === 'task' ? '☐' : hit.source_type === 'note' ? '≣' : '▣'}
                  </span>
                  <span class="min-w-0 flex-1 truncate">{hit.title}</span>
                  {#if hit.project}
                    <span class="text-fg-muted text-xs">{hit.project}</span>
                  {/if}
                </Command.Item>
              {/each}
            {:else}
              <Command.Group>
                <Command.GroupHeading class="text-fg-muted px-3 pt-2 pb-1 text-xs uppercase">
                  Go to
                </Command.GroupHeading>
                <Command.GroupItems>
                  {#each quickNav as q (q.path)}
                    <Command.Item
                      value={q.label}
                      onSelect={() => go(q.path)}
                      class="data-selected:glass flex cursor-pointer items-center gap-2.5
                             rounded-md px-3 py-2"
                    >
                      <span class="text-ember">{q.glyph}</span>
                      {q.label}
                    </Command.Item>
                  {/each}
                  {#each folders as f (f.id)}
                    <Command.Item
                      value={f.name}
                      onSelect={() => go(`/p/${f.id}`)}
                      class="data-selected:glass flex cursor-pointer items-center gap-2.5
                             rounded-md px-3 py-2"
                    >
                      <span class={f.kind === 'project' ? 'text-ember' : 'text-ember-bright'}>
                        {f.kind === 'project' ? '▣' : '◈'}
                      </span>
                      {f.name}
                    </Command.Item>
                  {/each}
                </Command.GroupItems>
              </Command.Group>
              <Command.Empty class="text-fg-muted px-3 py-2 text-sm">
                Press Enter to ask Sophon.
              </Command.Empty>
            {/if}
          </Command.List>
        </Command.Root>
      {/if}
    </Dialog.Content>
  </Dialog.Portal>
</Dialog.Root>
