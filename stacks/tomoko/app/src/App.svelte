<script lang="ts">
  import { router } from './lib/router.svelte';
  import Sidebar from './lib/Sidebar.svelte';
  import Omnibar from './lib/Omnibar.svelte';
  import CommandBar from './lib/CommandBar.svelte';
  import HomePage from './pages/HomePage.svelte';
  import FolderPage from './pages/FolderPage.svelte';
  import TaskPage from './pages/TaskPage.svelte';
  import NotePage from './pages/NotePage.svelte';
  import ViewPage from './pages/ViewPage.svelte';
  import ResultsPage from './pages/ResultsPage.svelte';

  // Two distinct shells over the same pages:
  //  - Mobile: drill-in stack, header back button, bottom command bar where
  //    draft cards materialize above the input.
  //  - Desktop: persistent sidebar + content pane, Cmd+K omnibar.
  let omnibar = $state<Omnibar | null>(null);

  const route = $derived(router.route);
  const atRoot = $derived(route.name === 'home');
</script>

<div class="md:flex">
  <!-- Desktop sidebar -->
  <div class="hidden md:block">
    <Sidebar onOmnibar={() => omnibar?.toggle()} />
  </div>

  <main class="mx-auto w-full max-w-2xl px-4 pt-3 pb-32 md:h-dvh md:overflow-y-auto md:pt-6 md:pb-8">
    <!-- Mobile header: logo at root, back button when deep -->
    <header class="flex items-center gap-2 py-3 md:hidden">
      {#if atRoot}
        <img src="/tomoko_logo.svg" alt="Tomoko" class="glow-drop h-11 w-auto" />
      {:else}
        <button
          type="button"
          class="text-ember -ml-1 cursor-pointer rounded-md px-2 py-1 text-xl"
          onclick={() => router.back()}
        >
          ‹
        </button>
        <img src="/tomoko_logo.svg" alt="Tomoko" class="glow-drop h-7 w-auto opacity-90" />
      {/if}
    </header>

    {#if route.name === 'home'}
      <!-- Desktop has the sidebar; home only matters on mobile -->
      <div class="md:hidden"><HomePage /></div>
      <div class="hidden md:block"><ViewPage view="today" /></div>
    {:else if route.name === 'folder'}
      <FolderPage id={route.id} />
    {:else if route.name === 'task'}
      <TaskPage id={route.id} />
    {:else if route.name === 'note'}
      <NotePage id={route.id} />
    {:else if route.name === 'view'}
      <ViewPage view={route.view} />
    {:else if route.name === 'results'}
      <ResultsPage />
    {/if}
  </main>
</div>

<!-- Mobile: anchored bottom command bar (hidden on md+) -->
<CommandBar />

<!-- Desktop: Cmd+K palette -->
<Omnibar bind:this={omnibar} />
