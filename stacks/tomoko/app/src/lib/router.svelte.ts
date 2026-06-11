// Tiny history router. Routes:
//   /            home (mobile: views + tree; desktop redirects to /view/today)
//   /p/:id       folder (project/context drill-in)
//   /t/:id       task detail/editor
//   /n/:id       note detail/editor
//   /view/:name  smart views (today | upcoming | inbox)
//   /results     last command results

export type Route =
  | { name: 'home' }
  | { name: 'folder'; id: number }
  | { name: 'task'; id: number }
  | { name: 'note'; id: number }
  | { name: 'view'; view: 'today' | 'upcoming' | 'inbox' }
  | { name: 'results' };

function parse(path: string): Route {
  const m = path.match(/^\/(p|t|n)\/(\d+)$/);
  if (m) {
    const id = Number(m[2]);
    if (m[1] === 'p') return { name: 'folder', id };
    if (m[1] === 't') return { name: 'task', id };
    return { name: 'note', id };
  }
  const v = path.match(/^\/view\/(today|upcoming|inbox)$/);
  if (v) return { name: 'view', view: v[1] as 'today' | 'upcoming' | 'inbox' };
  if (path === '/results') return { name: 'results' };
  return { name: 'home' };
}

class Router {
  path = $state(window.location.pathname);
  route = $derived(parse(this.path));

  constructor() {
    window.addEventListener('popstate', () => {
      this.path = window.location.pathname;
    });
  }

  go(path: string) {
    if (path === this.path) return;
    history.pushState({}, '', path);
    this.path = path;
  }

  back() {
    if (history.length > 1) history.back();
    else this.go('/');
  }
}

export const router = new Router();
